package pir

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"

	"tapir/modules/database"
	"tapir/modules/psetggm"
	"tapir/modules/utils"
	"tapir/modules/vc"
)

var seedHint = [32]byte{2}

type TAPIRDigest struct {
	Coms []vc.Commitment
}

type TAPIRHintQuery struct {
}

type TAPIRHintResp struct {
	Answers []database.Record
}

type TAPIRHint struct {
	Parities []database.Record
	// permutation maps
	IdxToSetIdx [][]uint32
	SetIdxToIdx [][]uint32
}

type TAPIRQuery struct {
	Indices []uint32
}

type TAPIRAnswer struct {
	FlatRecords []byte
	AggProof    vc.AggProof
}

type TAPIRServer struct {
	Db     *database.DB //used offline
	Proofs []vc.Proof
	Q      int
	M      int
	Role   int
	Digest *TAPIRDigest // contains commitments
	Vc     vc.VCParams
	VcType vc.VcType
}

type TAPIRClient struct {
	// database parameters
	N       int
	Q       int
	M       int
	RecSize int

	// query and refresh state
	queriedIdx int
	randSwaps  []uint32
	setOffline []uint32 // query q_0 indices
	setOnline  []uint32 // query q_1 indices

	// Digest
	Digest *TAPIRDigest
	// Hint
	Hint *TAPIRHint

	// randomness
	Prg *rand.ChaCha8

	Vc vc.VCParams
}

////////////////////////////////////////////////////////////
// OFFLINE PHASE
////////////////////////////////////////////////////////////

func NewTAPIRClient(n, q, recSize int, vcType vc.VcType) *TAPIRClient {
	c := &TAPIRClient{N: n, Q: q, M: n / q}
	c.Vc = vc.NewVc(vcType, c.M)
	c.RecSize = recSize
	return c
}

func NewTAPIRServer(db *database.DB, Q int, role int, vcType vc.VcType) *TAPIRServer {
	s := &TAPIRServer{Db: db, Q: Q, M: db.N / Q, Role: role, VcType: vcType}
	s.Vc = vc.NewVc(vcType, s.M)
	return s
}

func (s *TAPIRServer) Equals(other APIRServer) (bool, error) {
	s2 := other.(*TAPIRServer)
	if b, err := s.Db.Equals(s2.Db); !b {
		return false, err
	}
	for i := range s.Proofs {
		p2 := (s2.Proofs)[i]
		p1 := (s.Proofs)[i]

		if !s2.Vc.EqualProofs(p1, p2) {
			return false, errors.New("proof db not equal")
		}
		if !s.Vc.EqualProofs(p1, p2) {
			return false, errors.New("proof db not equal")
		}
	}
	if s.Q != s2.Q || s.M != s2.M || s.Role != s2.Role {
		return false, errors.New("server parameters not equal")
	}

	for i := range s.Digest.Coms {
		if !s.Vc.EqualCommitments((s.Digest.Coms[i]), (s2.Digest.Coms[i])) {
			return false, errors.New("digests db not equal")
		}
	}
	return true, nil
}
func (s *TAPIRServer) GetVCType() vc.VcType {
	return s.VcType
}
func (s *TAPIRServer) SetVC(vctype vc.VcType) {
	s.Vc = vc.NewVc(vctype, s.M)
}

func (s *TAPIRServer) GenDigest() (Digest, error) {
	// initialize TAPIRDigest
	d := TAPIRDigest{}
	d.Coms = make([]vc.Commitment, s.Q)
	proofs := make([]vc.Proof, s.Db.N)

	// OUTER LOOP: Compute vector commitments for each row
	var vec vc.Vector
	for q := 0; q < s.Q; q++ { // there are Q rows
		// get s.m records starting from index q*s.m
		recs := s.Db.GetRecords(q*s.M, s.M)
		if recs == nil {
			return nil, errors.New("error getting records when generating digest")
		}
		vec = s.Vc.VectorFromRecords(recs)
		d.Coms[q] = s.Vc.Commit(vec)

		// INNER LOOP: Compute opening proof, save in augmented database
		for i := 0; i < s.M; i++ {
			proofs[i+q*s.M] = s.Vc.Open(vec, i, d.Coms[q])
		}
	}
	s.Proofs = proofs
	s.Digest = &d
	return &d, nil
}

func (s *TAPIRServer) GetDigest() Digest {
	return s.Digest
}
func (s *TAPIRServer) GetDB() *database.DB {
	return s.Db
}

// Updates TAPIR Database returns N, Q, digest, updateOps
func (s *TAPIRServer) Update(ops []database.Update) (int, int, Digest, []database.Update) {
	// get updates for each partition
	partitionOps := make([][]database.Update, s.Q)
	addCtr := 0
	for i := range ops {
		if ops[i].Op == database.ADD {
			ops[i].Idx = s.Db.N + addCtr
			addCtr++
		}
		q := ops[i].Idx / s.M
		if q >= len(partitionOps) {
			partitionOps = append(partitionOps, []database.Update{ops[i]})
		} else {
			partitionOps[q] = append(partitionOps[q], ops[i])
		}
	}

	// Apply updates to each partition
	for q, partitionOp := range partitionOps {
		if len(partitionOp) == 0 {
			continue
		}
		var vec vc.Vector

		// add new partition
		if q >= s.Q {
			if len(partitionOp) > s.M {
				log.Fatalln("number of ops too big for new partition")
			}
			// extend DB capacity by another partition of size M
			s.Db.ExtendCapacity(s.M)

			newProofs := make([]vc.Proof, s.M)
			s.Proofs = append(s.Proofs, newProofs...)

			// Add new records to DB
			for _, op := range partitionOp {
				// Copy the new record data
				s.Db.SetRecord(op.Idx, op.Val)
				s.Db.N++
			}

			recs := s.Db.GetRecords(q*s.M, s.M)
			if recs == nil {
				log.Fatal("error getting db records")
			}
			// Commit to new partition
			vec = s.Vc.VectorFromRecords(recs)
			s.Digest.Coms = append(s.Digest.Coms, s.Vc.Commit(vec))

		} else { // q < s.Q // edit existing partition
			recs := s.Db.GetRecords(q*s.M, s.M)
			if recs == nil {
				log.Fatal("error getting db records")
			}
			vec = s.Vc.VectorFromRecords(recs)
			for i, op := range partitionOp {
				valOld := s.Db.GetRecord(op.Idx)

				s.Db.SetRecord(op.Idx, op.Val)
				if op.Op == database.ADD {
					s.Db.N++
				}
				// update commitment
				s.Digest.Coms[q], vec = s.Vc.Update(
					s.Digest.Coms[q],
					vec,
					database.Update{Idx: op.Idx % s.M, Val: op.Val, Op: op.Op},
				)

				// Save delta of op: val_old XOR val_new to set
				psetggm.FastXorInto(partitionOps[q][i].Val, valOld, s.Db.RecSize)
				// }
			}
		}
		// for each value in partition q update opening proof
		if len(partitionOp) != 0 {
			for m := range s.M {
				s.Proofs[m+q*s.M] = s.Vc.Open(vec, m, s.Digest.Coms[q])
			}
		}
	}
	s.Q = len(partitionOps)
	return s.Db.N, s.Q, s.Digest, ops
}

func (c *TAPIRClient) UpdateHint(newN0, newN1, newQ0, newQ1 int, newDigest0, newDigest1 Digest, ops0, ops1 []database.Update) (int, int, Digest, Hint, error) {
	if newN0 != newN1 || newQ0 != newQ1 || !c.EqualDigests(newDigest0, newDigest1) || len(ops0) != len(ops1) {
		return -1, -1, nil, nil, errors.New("update parameters from servers do not match")
	}
	oldQ := c.Q
	c.Q = newQ0

	c.Prg = rand.NewChaCha8(seedHint)
	randSeed := *utils.RandomPRGKey(c.Prg)

	squash_rand := 0
	for i := range len(randSeed) {
		squash_rand += int(randSeed[i])
	}

	// Apply updates to each partition
	for q := range newQ0 {
		// Check if need to add new partition
		if q >= oldQ {
			// Generate a new permutation for this partition & set up permutation maps
			c.Hint.IdxToSetIdx = append(c.Hint.IdxToSetIdx, make([]uint32, c.M))
			c.Hint.SetIdxToIdx = append(c.Hint.SetIdxToIdx, make([]uint32, c.M))
			psetggm.GenerateSinglePerm(c.M, squash_rand, c.Hint.IdxToSetIdx[oldQ], c.Hint.SetIdxToIdx[oldQ])

			// get all ops for this new partition
			for i, op := range ops0 {
				if op.Idx < (q+1)*c.M && op.Idx >= (q)*c.M {
					// Check Equality of ops
					if !ops0[i].Equals(ops1[i]) {
						return -1, -1, nil, nil, errors.New("update operations from servers do not match")
					}
					if op.Op != database.ADD {
						return -1, -1, nil, nil, errors.New("only ADD ops in new partition possible")
					}
					_, _, pos := c.findIndex(op.Idx)
					// update hint
					psetggm.FastXorInto(c.Hint.Parities[pos], op.Val, c.RecSize)
				}
			}
		} else { // q < s.Q
			for i, op := range ops0 {
				if op.Idx < (q+1)*c.M && op.Idx >= (q)*c.M {
					// Check Equality of ops
					if !ops0[i].Equals(ops1[i]) {
						return -1, -1, nil, nil, errors.New("update operations from servers do not match")
					}
					_, _, pos := c.findIndex(op.Idx)
					// update hint
					psetggm.FastXorInto(c.Hint.Parities[pos], op.Val, c.RecSize)
				}
			}
		}
	}
	c.N = newN0
	c.Digest = newDigest0.(*TAPIRDigest)

	return c.N, c.Q, newDigest0.(*TAPIRDigest), &c.Hint, nil
}

func (c *TAPIRClient) EqualDigests(d0, d1 Digest) bool {
	for i := 0; i < c.Q; i++ {
		if !c.Vc.EqualCommitments(d0.(*TAPIRDigest).Coms[i], d1.(*TAPIRDigest).Coms[i]) {
			return false
		}
	}
	return true
}

func (c *TAPIRClient) RequestHint() (HintQuery, HintQuery, error) {

	c.Hint = &TAPIRHint{} // initialize hint
	c.Prg = rand.NewChaCha8(seedHint)
	randSeed := *utils.RandomPRGKey(c.Prg)

	squash_rand := 0
	for i := range len(randSeed) {
		squash_rand += int(randSeed[i])
	}

	// We initially assume that N = Q*M, but this might not hold after updates
	permutations := make([]uint32, c.N)
	inverse_permutations := make([]uint32, c.N)

	psetggm.GeneratePerms(c.N, c.Q, squash_rand, permutations, inverse_permutations)

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

	return &TAPIRHintQuery{}, &TAPIRHintQuery{}, nil
}

func (s *TAPIRServer) GenHint(hintQuery HintQuery) (HintResp, error) {
	if s.Db.RecSize%16 != 0 {
		return nil, errors.New("record size must be at least 16 bytes, otherwise the SIMD instructions won't work properly")
	}
	return &TAPIRHintResp{Answers: s.Db.GetRecords(0, s.Db.N)}, nil
}

func (c *TAPIRClient) VerSetup(d0 Digest, d1 Digest, resp0 HintResp, resp1 HintResp) (Digest, Hint, error) {

	// PROCESS DIGEST RESPONSES /////////////////////////////////
	// check equality of vector commitments
	if !c.EqualDigests(d0, d1) {
		return nil, nil, errors.New("vector commitments are not equal")
	}
	digest := d0.(*TAPIRDigest)
	// save digest data
	c.Digest = digest

	// PROCESS HINT RESPONSES ///////////////////////////////////

	db0 := resp0.(*TAPIRHintResp).Answers
	db1 := resp1.(*TAPIRHintResp).Answers

	// initialize hint parities
	c.Hint.Parities = make([]database.Record, c.M)

	for m := range c.M {
		c.Hint.Parities[m] = make([]byte, c.RecSize)
		for q := range c.Q {
			idx := q*c.M + int(c.Hint.IdxToSetIdx[q][m])

			// verify equality of records
			if !bytes.Equal(db0[idx], db1[idx]) {
				return nil, nil, errors.New("received databases not equal")
			}
			psetggm.FastXorInto(c.Hint.Parities[m], db0[idx], c.RecSize)
		}
	}
	return c.Digest, c.Hint, nil

}

////////////////////////////////////////////////////////////
// ONLINE PHASE
////////////////////////////////////////////////////////////

func (c *TAPIRClient) findIndex(index int) (int, int, int) {
	// if index not in partition
	if index >= (c.M * c.Q) {
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
func (c *TAPIRClient) randomIdx(rangeMax int) uint32 {
	return uint32(c.Prg.Uint64() % uint64(rangeMax))
}

func (c *TAPIRClient) Query(i int) (Query, Query, error) {
	if len(c.Hint.Parities) < 1 {
		return nil, nil, errors.New("Hint is not set")
	}
	if i >= (c.M*c.Q) || i < 0 {
		return nil, nil, errors.New("Query index out of bounds of database")
	}

	c.queriedIdx = i
	// Where pos is "ind" in the paper
	row, _, pos := c.findIndex(i)

	c.setOnline = make([]uint32, c.Q)
	c.setOffline = make([]uint32, c.Q)
	randSwaps := make([]uint32, c.Q)

	for j := range c.Q {
		c.setOnline[j] = c.Hint.IdxToSetIdx[j][pos]
		randSwaps[j] = c.randomIdx(c.M)
		c.setOffline[j] = c.Hint.IdxToSetIdx[j][randSwaps[j]]
	}
	c.setOnline[row] = c.Hint.IdxToSetIdx[row][randSwaps[row]]
	c.randSwaps = randSwaps

	return &TAPIRQuery{Indices: c.setOffline},
		&TAPIRQuery{Indices: c.setOnline},
		nil
}

func (s *TAPIRServer) Answer(query Query) (Answer, error) {
	q := query.(*TAPIRQuery)
	answer := TAPIRAnswer{FlatRecords: make([]byte, s.Q*s.Db.RecSize)}

	currOffset := 0
	ps := make([]vc.Proof, s.Q)
	for i := range s.Q {
		psetggm.CopyIn(answer.FlatRecords[i*s.Db.RecSize:(i+1)*s.Db.RecSize], s.Db.Data, s.M*i+int(q.Indices[i]), s.Db.RecSize)
		currOffset += s.M
		ps[i] = s.Proofs[s.M*i+int(q.Indices[i])]
	}

	// For aggregated verification
	answer.AggProof = s.Vc.Aggregate(&ps, &s.Digest.Coms)

	return &answer, nil
}

func (c *TAPIRClient) Reconstruct(digest Digest, hint Hint, answer0 Answer, answer1 Answer) (database.Record, error) {

	// Verify individual elements sent in the response
	a0 := answer0.(*TAPIRAnswer)
	a1 := answer1.(*TAPIRAnswer)
	recSize := len(c.Hint.Parities[0])

	if len(a0.FlatRecords) != len(a1.FlatRecords) {
		return nil, errors.New("answer lengths are not equal")
	}
	if len(a1.FlatRecords)/(recSize) != c.Q {
		return nil, errors.New("answer not of expected length")
	}

	///////	 Aggregated verification 	///////
	// NOTE: MT does not allow for proof aggregation, the standard verification for each element is used instead

	recsOff := make([]database.Record, c.Q)
	recsOn := make([]database.Record, c.Q)
	indicesOff := make([]int, c.Q)
	indicesOn := make([]int, c.Q)
	for i := range c.Q {
		recsOff[i] = a0.FlatRecords[i*recSize : (i+1)*recSize]
		recsOn[i] = a1.FlatRecords[i*recSize : (i+1)*recSize]
		indicesOff[i] = int(c.setOffline[i])
		indicesOn[i] = int(c.setOnline[i])
	}
	okOff := c.Vc.VerifyAggregation(a0.AggProof, &c.Digest.Coms, indicesOff, recsOff)
	if !okOff {
		return nil, fmt.Errorf("answer verification failed for offline server")
	}
	okOn := c.Vc.VerifyAggregation(a1.AggProof, &c.Digest.Coms, indicesOn, recsOn)
	if !okOn {
		return nil, fmt.Errorf("answer verification failed for online server")
	}

	///////	 Iterative verification without aggregation 	///////
	// for i := range c.Q {
	// 	colOff := int(c.setOffline[i])
	// 	colOn := int(c.setOnline[i])

	// 	okOff := c.vc.Verify(c.digest.Coms[i], a0.Proofs[i], colOff, a0.FlatRecords[i*recSize:(i+1)*recSize])
	// 	okOn := c.vc.Verify(c.digest.Coms[i], a1.Proofs[i], colOn, a1.FlatRecords[i*recSize:(i+1)*recSize])

	// 	if !okOff || !okOn {
	// 		return nil, fmt.Errorf("record verify failed for record %v", i)
	// 	}
	// }

	///////	 Refresh Hint 	///////
	// (same as Singlepass Reconstruct)
	row, _, pos := c.findIndex(c.queriedIdx)
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
	for i := range c.Q {
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
