package pir

import (
	"errors"
	"log"
	"math/rand/v2"

	"tapir/modules/database"
	"tapir/modules/psetggm"
	"tapir/modules/utils"
	"tapir/modules/vc"
)

var SEED_SP = [32]byte{1}

// Offline phase types

type SinglePassDigest struct{} // There is no digest

type SinglePassHintQuery struct {
	RandSeed int
}

type SinglePassHintResp struct {
	Parities []database.Record
}

type SinglePassHint struct {
	Parities []database.Record
	// permutation maps
	IdxToSetIdx [][]uint32
	SetIdxToIdx [][]uint32
}

type SinglePassQuery struct {
	Indices []uint32
}

type SinglePassAnswer struct {
	FlatRecords []byte
}

type SinglePassServer struct {
	Db *database.DB
	Q  int
	M  int
}

type SinglePassClient struct {
	// database parameters
	N int
	Q int
	M int

	// query and refresh state
	queriedIdx int
	ind        int
	randSwaps  []uint32
	rInd       int

	// Hint
	Hint *SinglePassHint

	// randomness
	Prg *rand.ChaCha8
	// seed for the permutations
	PermSeed int
}

func (s *SinglePassServer) Equals(other APIRServer) (bool, error) {
	s2 := other.(*SinglePassServer)
	if b, err := s.Db.Equals(s2.Db); !b {
		return false, err
	}
	if s.Q != s2.Q || s.M != s2.M {
		return false, errors.New("server parameters not equal")
	}
	return true, nil
}

func (s *SinglePassServer) GetVCType() vc.VcType {
	return vc.VcType(0)
}
func (s *SinglePassServer) SetVC(vc.VcType) {
	// Server has no VC
}

////////////////////////////////////////////////////////////
// OFFLINE PHASE
////////////////////////////////////////////////////////////

// There is no offline phase, so these functions do nothing

func (s *SinglePassServer) GenDigest() (Digest, error) {
	return &SinglePassDigest{}, nil
}

func (c *SinglePassClient) RequestHint() (HintQuery, HintQuery, error) {
	seed := SEED_SP
	c.Prg = rand.NewChaCha8(seed)

	bigPermSeed := utils.RandomPRGKey(c.Prg)
	c.PermSeed = 0
	for i := 0; i < len(bigPermSeed); i++ {
		c.PermSeed += int(bigPermSeed[i])
	}

	return &SinglePassHintQuery{RandSeed: c.PermSeed}, nil, nil
}

func (c *SinglePassClient) EqualDigests(_, _ Digest) bool {
	return true
}

func (s *SinglePassServer) GenHint(hintQuery HintQuery) (HintResp, error) {
	if s.Db.RecSize < 16 {
		return nil, errors.New("Record size must be at least 16 bytes, otherwise the SIMD instructions won't work properly")
	}

	if hintQuery == nil {
		return nil, nil
	}
	hq := hintQuery.(*SinglePassHintQuery)

	hints := make([]database.Record, s.M)
	hintsBuf := make([]byte, s.M*s.Db.RecSize)

	permutations := make([]uint32, s.Db.N)
	inverse_permutations := make([]uint32, s.Db.N)

	psetggm.SinglePassAnswer(s.Db.Data, s.Db.N, s.Q, s.Db.RecSize, hintsBuf, hq.RandSeed, permutations, inverse_permutations)

	for i := 0; i < s.M; i++ {
		hints[i] = database.Record(hintsBuf[s.Db.RecSize*i : s.Db.RecSize*(i+1)])
	}

	return &SinglePassHintResp{Parities: hints}, nil
}

func (c *SinglePassClient) VerSetup(d0 Digest, d1 Digest, resp0 HintResp, resp1 HintResp) (Digest, Hint, error) {
	// make sure we only get one hint (resp0)
	if resp1 != nil {
		return nil, nil, errors.New("SinglePassClient only recieves one hint (from server 0)")
	}

	hintResp := resp0.(*SinglePassHintResp)

	// generate permutations locally
	permutations := make([]uint32, c.N)
	inverse_permutations := make([]uint32, c.N)

	psetggm.GeneratePerms(c.N, c.Q, c.PermSeed, permutations, inverse_permutations)

	if len(permutations) != c.N || len(inverse_permutations) != c.N {
		return nil, nil, errors.New("permutations length does not match N")
	}

	// save hint
	c.Hint = &SinglePassHint{
		Parities: hintResp.Parities,
	}

	// Set up permutation maps
	// Each array in idxToSetIdx is a permutation of the set {0, 1, ..., m-1}
	// Within that array, the encoding is as follows:
	// idxToSetIdx[i][j] = k means that the jth element of the ith set
	// is mapped under the permutation to the kth element of the database
	c.Hint.IdxToSetIdx = make([][]uint32, c.Q)
	c.Hint.SetIdxToIdx = make([][]uint32, c.Q)

	ind := 0
	i := 0
	for ind < len(permutations) {
		c.Hint.IdxToSetIdx[i] = permutations[ind : ind+c.M]
		c.Hint.SetIdxToIdx[i] = inverse_permutations[ind : ind+c.M]
		ind += c.M
		i += 1
	}

	return &SinglePassDigest{}, c.Hint, nil
}
func (s *SinglePassServer) GetDigest() Digest {
	return &SinglePassDigest{}
}
func (s *SinglePassServer) GetDB() *database.DB {
	return s.Db
}
func (s *SinglePassServer) Update(_ []database.Update) (Nt, Qt int, dt Digest, opst []database.Update) {
	log.Fatalf("not implemented")
	return
}

////////////////////////////////////////////////////////////
// ONLINE PHASE
////////////////////////////////////////////////////////////

func (c *SinglePassClient) findIndex(index int) (int, int, int) {
	if index >= c.N {
		return -1, -1, -1
	}
	// find the row index (i* in the paper)
	row := int(index / c.M)
	// find the column index (j* in the paper)
	col := index - (row * c.M)
	// finally, the last element is pInverse_row(col) in the paper
	return row, col, int(c.Hint.SetIdxToIdx[row][col])
}

// Sample a random element within range.
func (c *SinglePassClient) randomIdx(rangeMax int) uint32 {
	return uint32(c.Prg.Uint64() % uint64(rangeMax)) // TODO this is sketchy
}

func (c *SinglePassClient) Query(i int) (Query, Query, error) {
	if len(c.Hint.Parities) < 1 {
		return nil, nil, errors.New("Hint is not set")
	}
	if i >= c.N || i < 0 {
		return nil, nil, errors.New("Query index out of bounds of database")
	}

	c.queriedIdx = i

	// Paper's Query pseudocode line 1
	// Where pos is "ind" in the paper
	row, _, pos := c.findIndex(i)

	setOnline := make([]uint32, c.Q)
	setOffline := make([]uint32, c.Q)
	randSwaps := make([]uint32, c.Q)

	for j := 0; j < c.Q; j++ {
		// Paper's Query pseudocode line 2
		setOnline[j] = c.Hint.IdxToSetIdx[j][pos]
		// Paper's Query pseudocode line 3
		randSwaps[j] = c.randomIdx(c.M)
		// Paper's Query pseudocode line 4
		setOffline[j] = c.Hint.IdxToSetIdx[j][randSwaps[j]]
	}
	setOnline[row] = c.Hint.IdxToSetIdx[row][randSwaps[row]]

	c.randSwaps = randSwaps

	return &SinglePassQuery{Indices: setOffline},
		&SinglePassQuery{Indices: setOnline},
		nil
}

func (s *SinglePassServer) Answer(query Query) (Answer, error) {

	q := query.(*SinglePassQuery)

	answer := SinglePassAnswer{FlatRecords: make([]byte, s.Q*s.Db.RecSize)}

	currOffset := 0
	for i := 0; i < s.Q; i++ {
		psetggm.CopyIn(answer.FlatRecords[i*s.Db.RecSize:(i*s.Db.RecSize)+s.Db.RecSize], s.Db.Data, s.M*i+int(q.Indices[i]), s.Db.RecSize)
		currOffset += s.M
	}

	return &answer, nil
}

func (c *SinglePassClient) Reconstruct(digest Digest, hint Hint, answer0 Answer, answer1 Answer) (database.Record, error) {

	row, _, pos := c.findIndex(c.queriedIdx)

	a0 := answer0.(*SinglePassAnswer)
	a1 := answer1.(*SinglePassAnswer)

	recSize := len(c.Hint.Parities[0])

	out := make(database.Record, recSize)

	xorResp0 := make(database.Record, recSize)
	xorResp1 := make(database.Record, recSize)

	psetggm.XorBlocksTogether(a0.FlatRecords, xorResp0, recSize, c.Q)
	psetggm.XorBlocksTogether(a1.FlatRecords, xorResp1, recSize, c.Q)

	upos := uint32(pos)

	psetggm.FastXorInto(out, xorResp1, recSize)
	psetggm.FastXorInto(out, c.Hint.Parities[pos], recSize)
	psetggm.FastXorInto(out, a1.FlatRecords[row*recSize:(row+1)*recSize], recSize)

	c.Hint.Parities[pos] = xorResp0
	for i := 0; i < c.Q; i++ {
		//3)
		psetggm.FastXorInto(c.Hint.Parities[c.randSwaps[i]], a0.FlatRecords[i*recSize:(i+1)*recSize], recSize)
		psetggm.FastXorInto(c.Hint.Parities[c.randSwaps[i]], a1.FlatRecords[i*recSize:(i+1)*recSize], recSize)
		//4)
		temp1 := c.Hint.IdxToSetIdx[i][pos]
		//can remove temp2 if not updatable
		temp2 := c.Hint.IdxToSetIdx[i][c.randSwaps[i]]
		//5)
		c.Hint.IdxToSetIdx[i][pos] = c.Hint.IdxToSetIdx[i][c.randSwaps[i]]
		//6)
		c.Hint.IdxToSetIdx[i][c.randSwaps[i]] = temp1
		//7)
		//for updatable: need to update new datastructure setIdxToIdx
		c.Hint.SetIdxToIdx[i][temp1] = c.randSwaps[i]
		c.Hint.SetIdxToIdx[i][temp2] = upos

	}
	//fix xoring once more than necessary
	psetggm.FastXorInto(c.Hint.Parities[c.randSwaps[row]], a1.FlatRecords[row*recSize:(row+1)*recSize], recSize)
	psetggm.FastXorInto(c.Hint.Parities[c.randSwaps[row]], out, recSize)

	return database.Record(out), nil
}
func (c *SinglePassClient) UpdateHint(newN0, newN1, newQ0, newQ1 int, newDigest0, newDigest1 Digest, ops0, ops1 []database.Update) (N int, Q int, d Digest, hint Hint, err error) {
	log.Fatal("not implemented yet")
	return
}
