package pir

import (
	"bytes"
	"errors"
	"log"
	"math/rand"
	"tapir/modules/database"
	oc "tapir/modules/osu_crypto"
	"tapir/modules/vc"
)

// //////////////////////////////////////////////////////////
// GLOBAL MATHEMATICAL CONSTANTS
// //////////////////////////////////////////////////////////

const alphaLen = 16
const test_numpoints = 1
const BLOCKSIZE = 16

////////////////////////////////////////////////////////////
// DPF128 INTERFACE
////////////////////////////////////////////////////////////

// There is no offline phase in this protocol, define dummy types
type DPF128Digest struct{}
type DPF128HintQuery struct{}
type DPF128HintResp struct{}
type DPF128Hint struct{}

// Online phase types
type DPF128Query struct {
	QueryKey []byte
	AuthKey  []byte
	KeySize  uint64
}
type DPF128Answer struct {
	QueryRecord []byte
	AuthRecord  []byte
}

type DPF128Server struct {
	Db   *database.DB
	Role byte
}
type DPF128Client struct {
	N          int
	queriedIdx int
	alpha      *oc.FieldElem
}

////////////////////////////////////////////////////////////
// OFFLINE PHASE
////////////////////////////////////////////////////////////

// There is no offline phase, so these functions do nothing

func (s *DPF128Server) Equals(other APIRServer) (bool, error) {
	s2 := other.(*DPF128Server)
	if b, err := s.Db.Equals(s2.Db); !b {
		return false, err
	}
	if s.Role != s2.Role {
		return false, errors.New("roles not equal")
	}
	return true, nil
}

func (s *DPF128Server) GetVCType() vc.VcType {
	return vc.VcType(0)
}
func (s *DPF128Server) SetVC(vc.VcType) {
	// Server has no VC
}

func (s *DPF128Server) GenDigest() (Digest, error) {
	return &DPF128Digest{}, nil
}

func (s *DPF128Server) GetDigest() Digest {
	return &DPF128Digest{}
}

func (s *DPF128Server) Update(_ []database.Update) (Nt, Qt int, dt Digest, opst []database.Update) {
	log.Fatalf("not implemented")
	return
}

func (s *DPF128Server) GetDB() *database.DB {
	return s.Db
}

func (c *DPF128Client) RequestHint() (HintQuery, HintQuery, error) {
	return &DPF128HintQuery{}, &DPF128HintQuery{}, nil
}

func (s *DPF128Server) GenHint(hq HintQuery) (HintResp, error) {
	return &DPF128HintResp{}, nil
}

func (c *DPF128Client) VerSetup(d0 Digest, d1 Digest, resp0 HintResp, resp1 HintResp) (Digest, Hint, error) {
	return &DPF128Digest{}, DPF128Hint{}, nil
}

func (c *DPF128Client) EqualDigests(d0 Digest, d1 Digest) bool {
	return true
}

////////////////////////////////////////////////////////////
// ONLINE PHASE
////////////////////////////////////////////////////////////

func (c *DPF128Client) Query(i int) (Query, Query, error) {

	// make random seed of 16 bytes
	seed := rand.Uint64()
	seedAuth := rand.Uint64()

	domain := uint64(c.N)

	points := make([]uint64, test_numpoints)
	points[0] = uint64(i)

	// AUTH PIR /////////////////////////////////////////////////

	if c.alpha == nil {
		c.alpha = oc.NewFieldElem()
	}

	// make some keys
	keyAuth0, keyAuth1, keySizeAuth := oc.KeyGen(domain, points, c.alpha, seedAuth)

	// NORMAL PIR ////////////////////////////////////////////////

	// must be 1 because ALPHA is 1 here!
	one := oc.FieldElemOne()

	// make some keys
	queryKey0, queryKey1, keySize := oc.KeyGen(domain, points, one, seed)

	if keySize != keySizeAuth {
		return nil, nil, errors.New("key sizes do not match")
	}

	return &DPF128Query{queryKey0, keyAuth0, keySize},
		&DPF128Query{queryKey1, keyAuth1, keySize}, nil
}

func (c *DPF128Client) QueryAlpha(i int, alpha *oc.FieldElem) (Query, Query, error) {

	c.alpha = alpha

	return c.Query(i)
}

func (s *DPF128Server) Answer(query Query) (Answer, error) {
	q := query.(*DPF128Query)

	domain := uint64(s.Db.N)

	keyExp := oc.Expand(uint64(s.Role), domain, test_numpoints, q.QueryKey, q.KeySize)

	keyExpAuth := oc.Expand(uint64(s.Role), domain, test_numpoints, q.AuthKey, q.KeySize)

	// multiply the keys by the database
	resQuery := oc.MultiplyDB(keyExp, s.Db.Data, int(domain))
	resAuth := oc.MultiplyDB(keyExpAuth, s.Db.Data, int(domain))

	return &DPF128Answer{resQuery.Data, resAuth.Data}, nil
}

func (c *DPF128Client) Reconstruct(_ Digest, _ Hint, answer0 Answer, answer1 Answer) (database.Record, error) {

	a0, a1 := answer0.(*DPF128Answer), answer1.(*DPF128Answer)

	a0Rec := a0.QueryRecord
	a1Rec := a1.QueryRecord
	a0Auth := a0.AuthRecord
	a1Auth := a1.AuthRecord

	// XOR the results
	authRecon := oc.NewFieldElem()
	oc.XorDPF(a0Auth, a1Auth, authRecon.Data, BLOCKSIZE)
	queriedRecord := oc.NewFieldElem()
	oc.XorDPF(a0Rec, a1Rec, queriedRecord.Data, BLOCKSIZE)

	// MULTIPLY FOR AUTH CHECK ////////////////////////////////////

	// if normal * alpha = auth, we are good
	prod := oc.FieldMul(queriedRecord, c.alpha)

	if !bytes.Equal(authRecon.Data, prod.Data) {
		return nil, errors.New("authentication failed during DPF128 reconstruction")
	}

	return queriedRecord.Data, nil
}

func (c *DPF128Client) ReconstructAlpha(_ Digest, _ Hint, answer0 Answer, answer1 Answer, alpha *oc.FieldElem) (database.Record, error) {
	a0, a1 := answer0.(*DPF128Answer), answer1.(*DPF128Answer)

	c.alpha = alpha

	return c.Reconstruct(nil, nil, a0, a1)
}

func (c *DPF128Client) UpdateHint(newN0, newN1, newQ0, newQ1 int, newDigest0, newDigest1 Digest, ops0, ops1 []database.Update) (N int, Q int, d Digest, hint Hint, err error) {
	log.Fatal("not implemented yet")
	return
}
