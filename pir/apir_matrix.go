package pir

import (
	"errors"
	"log"
	"math/rand"
	"tapir/modules/database"
	"tapir/modules/utils"
	"tapir/modules/vc"
)

// There is no offline phase in this protocol, define dummy types
type APIR_MatrixDigest struct {
	Digest    vc.Commitment
	ProofSize int
}
type APIR_MatrixHintQuery struct{}
type APIR_MatrixHintResp struct {
	Digest vc.Commitment
}
type APIR_MatrixHint struct{}

// Online phase types
type APIR_MatrixQuery struct {
	BitVector []bool
}
type APIR_MatrixAnswer struct {
	FlatRecords []byte
	FlatProofs  []byte
}

type APIR_MatrixServer struct {
	Db        *database.DB
	AugDB     *database.DB
	ProofSize int
	Digest    *APIR_MatrixDigest // contains commitments
	VcType    vc.VcType
	Vc        vc.VCParams
}

type APIR_MatrixClient struct {
	N       int // Number of DB records
	Height  int
	Width   int
	RecSize int

	RandSource *rand.Rand

	vc vc.VCParams

	// State
	queriedIdx int
}

func (s *APIR_MatrixServer) Equals(other APIRServer) (bool, error) {
	s2 := other.(*APIR_MatrixServer)
	if b, err := s.Db.Equals(s2.Db); !b {
		return false, err
	}
	if b, err := s.AugDB.Equals(s2.AugDB); !b {
		return false, err
	}
	if s.ProofSize != s2.ProofSize {
		return false, errors.New("proofSize not equal")
	}
	return true, nil
}

func (s *APIR_MatrixServer) GetVCType() vc.VcType {
	return s.VcType
}
func (s *APIR_MatrixServer) SetVC(vctype vc.VcType) {
	s.Vc = vc.NewVc(vctype, s.Db.N)
}

////////////////////////////////////////////////////////////
// OFFLINE PHASE
////////////////////////////////////////////////////////////

func SetupAPIR_MatrixClient(N, recSize int, vctype vc.VcType) *APIR_MatrixClient {
	c := APIR_MatrixClient{N: N}
	c.RandSource = rand.New(utils.NewBufPRG(utils.NewPRG(&masterKey)))
	c.RecSize = recSize
	c.vc = vc.NewVc(vctype, N)
	return &c
}
func SetupAPIR_MatrixServer(db *database.DB, vctype vc.VcType) *APIR_MatrixServer {
	s := APIR_MatrixServer{Db: db, Vc: vc.NewVc(vctype, db.N), VcType: vctype}
	return &s
}

func (c *APIR_MatrixClient) RequestHint() (HintQuery, HintQuery, error) {
	return &APIR_MatrixHintQuery{}, &APIR_MatrixHintQuery{}, nil
}

func (c *APIR_MatrixClient) VerSetup(d0 Digest, d1 Digest, resp0 HintResp, resp1 HintResp) (Digest, Hint, error) {
	if !c.EqualDigests(d0, d1) {
		return nil, nil, errors.New("digests do not match")
	}
	c.Width, c.Height = getHeightWidth(c.N, c.RecSize+d0.(*APIR_MatrixDigest).ProofSize)

	return d0, &APIR_MatrixHint{}, nil
}

func (s *APIR_MatrixServer) GenDigest() (Digest, error) {
	vec := s.Vc.VectorFromRecords(s.Db.GetRecords(0, s.Db.N))
	com := s.Vc.Commit(vec)

	s.ProofSize = len(s.Vc.ProofToBytes(s.Vc.Open(vec, 0, com)))
	augDB := make([]byte, s.Db.N*(s.Db.RecSize+s.ProofSize))

	for i := 0; i < s.Db.N; i++ {
		p_b := s.Vc.ProofToBytes(s.Vc.Open(vec, i, com))
		copy(augDB[i*(s.Db.RecSize+s.ProofSize):(i)*(s.Db.RecSize+s.ProofSize)+s.Db.RecSize], s.Db.GetRecord(i))
		copy(augDB[i*(s.Db.RecSize+s.ProofSize)+s.Db.RecSize:(i+1)*(s.Db.RecSize+s.ProofSize)], p_b)
	}
	s.AugDB = &database.DB{N: s.Db.N, RecSize: s.Db.RecSize + s.ProofSize, Data: augDB}

	d := APIR_MatrixDigest{Digest: com, ProofSize: s.ProofSize}
	s.Digest = &d
	return s.Digest, nil
}

func (s *APIR_MatrixServer) GetDigest() Digest {
	return s.Digest
}
func (s *APIR_MatrixServer) GetDB() *database.DB {
	return s.Db
}
func (s *APIR_MatrixServer) Update(_ []database.Update) (Nt, Qt int, dt Digest, opst []database.Update) {
	log.Fatalf("not implemented")
	return
}

func (s *APIR_MatrixServer) GenHint(hq HintQuery) (HintResp, error) {
	return &APIR_MatrixHintResp{}, nil
}
func (c *APIR_MatrixClient) EqualDigests(d0, d1 Digest) bool {
	if !c.vc.EqualCommitments(d0.(*APIR_MatrixDigest).Digest, d1.(*APIR_MatrixDigest).Digest) {
		return false
	}

	return d0.(*APIR_MatrixDigest).ProofSize == d1.(*APIR_MatrixDigest).ProofSize
}

func (c *APIR_MatrixClient) Query(idx int) (Query, Query, error) {
	c.queriedIdx = idx
	if idx >= c.N || idx < 0 {
		return nil, nil, errors.New("Query index out of bounds of database")
	}
	rowNum := idx / c.Width
	// colNum := idx % c.width
	qL := make([]bool, c.Height)
	qR := make([]bool, c.Height)
	for i := 0; i < c.Height; i++ {
		qL[i] = (c.RandSource.Uint64()&1 == 0)
		qR[i] = (qL[i] != (i == rowNum))
	}
	return &APIR_MatrixQuery{qL}, &APIR_MatrixQuery{qR}, nil
}

func (c *APIR_MatrixClient) Reconstruct(digest Digest, _ Hint, answer0 Answer, answer1 Answer) (database.Record, error) {

	a0 := answer0.(*APIR_MatrixAnswer)
	a1 := answer1.(*APIR_MatrixAnswer)
	colNum := c.queriedIdx % c.Width
	rowNum := c.queriedIdx / c.Width

	database.XorInto(a0.FlatRecords, a1.FlatRecords)

	proofSize := digest.(*APIR_MatrixDigest).ProofSize

	for i := range len(a0.FlatProofs) / (c.RecSize + proofSize) {

		rec := a0.FlatRecords[i*(c.RecSize+proofSize) : i*(c.RecSize+proofSize)+c.RecSize]
		proof, err := c.vc.BytesToProof(a0.FlatRecords[i*(c.RecSize+proofSize)+c.RecSize : (i+1)*(c.RecSize+proofSize)])
		if err != nil {
			return nil, err
		}
		if !c.vc.Verify(digest.(*APIR_MatrixDigest).Digest, proof, rowNum+i, rec) {
			return nil, errors.New("failed to verify proof")
		}
	}
	return a0.FlatRecords[(c.RecSize+proofSize)*colNum : ((c.RecSize+proofSize)*(colNum) + c.RecSize)], nil
}

func (s *APIR_MatrixServer) Answer(q Query) (Answer, error) {
	recs := matBoolVecProduct(s.AugDB.Data, s.AugDB.N, s.AugDB.RecSize, q.(*APIR_MatrixQuery).BitVector)

	return &APIR_MatrixAnswer{
		FlatRecords: recs,
	}, nil
}
func (c *APIR_MatrixClient) UpdateHint(newN0, newN1, newQ0, newQ1 int, newDigest0, newDigest1 Digest, ops0, ops1 []database.Update) (N int, Q int, d Digest, hint Hint, err error) {
	log.Fatal("not implemented yet")
	return
}
