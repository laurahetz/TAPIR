package pir

import (
	"log"
	"tapir/modules/database"
	"tapir/modules/utils"
	"tapir/modules/vc"

	"github.com/dkales/dpf-go/dpf" // TODO does this use the local version in modules due to the go.mod file?
)

// There is no offline phase in this protocol, define dummy types
type DPFDigest struct{}
type DPFHintQuery struct{}
type DPFHintResp struct{}
type DPFHint struct{}

// Online phase types
type DPFQuery struct {
	QueryKey dpf.DPFkey
}
type DPFAnswer struct {
	QueryRecord database.Record
}

type DPFServer struct {
	Db *database.DB
}
type DPFClient struct {
	N          int
	queriedIdx int
}

func (s *DPFServer) Equals(other APIRServer) (bool, error) {
	s2 := other.(*DPFServer)
	if b, err := s.Db.Equals(s2.Db); !b {
		return false, err
	}
	return true, nil
}
func (s *DPFServer) GetVCType() vc.VcType {
	return vc.VcType(0)
}
func (s *DPFServer) SetVC(vc.VcType) {
	return
}

////////////////////////////////////////////////////////////
// OFFLINE PHASE
////////////////////////////////////////////////////////////

func (s *DPFServer) Update(_ []database.Update) (Nt, Qt int, dt Digest, opst []database.Update) {
	log.Fatalf("not implemented")
	return
}

// There is no offline phase, so these functions do nothing

func (s *DPFServer) GenDigest() (Digest, error) {
	return &DPFDigest{}, nil
}

func (s *DPFServer) GenHint(hq HintQuery) (HintResp, error) {
	return &DPFHintResp{}, nil
}

func (c *DPFClient) RequestHint() (HintQuery, HintQuery, error) {
	return &DPFHintQuery{}, &DPFHintQuery{}, nil
}

func (c *DPFClient) VerSetup(d0 Digest, d1 Digest, resp0 HintResp, resp1 HintResp) (Digest, Hint, error) {
	return &DPFDigest{}, DPFHint{}, nil
}

func (c *DPFClient) EqualDigests(_, _ Digest) bool {
	return true
}
func (s *DPFServer) GetDigest() Digest {
	return &DPFDigest{}
}
func (s *DPFServer) GetDB() *database.DB {
	return s.Db
}

////////////////////////////////////////////////////////////
// ONLINE PHASE
////////////////////////////////////////////////////////////

func (c *DPFClient) Query(i int) (Query, Query, error) {
	q0, q1 := dpf.Gen(uint64(i), utils.LogN(c.N))
	return &DPFQuery{q0}, &DPFQuery{q1}, nil
}

func (s *DPFServer) Answer(query Query) (Answer, error) {
	q := query.(*DPFQuery)
	expandedKey := dpf.EvalFull(q.QueryKey, utils.LogN(s.Db.N))
	return &DPFAnswer{s.Db.VectorProd(expandedKey)}, nil
}

func (c *DPFClient) Reconstruct(_ Digest, _ Hint, answer0 Answer, answer1 Answer) (database.Record, error) {
	a0, a1 := answer0.(*DPFAnswer), answer1.(*DPFAnswer)
	// XOR the two answers
	database.XorInto(a0.QueryRecord, a1.QueryRecord)
	// Return the result
	return a0.QueryRecord, nil
}
func (c *DPFClient) UpdateHint(newN0, newN1, newQ0, newQ1 int, newDigest0, newDigest1 Digest, ops0, ops1 []database.Update) (N int, Q int, d Digest, hint Hint, err error) {
	log.Fatal("not implemented yet")
	return
}
