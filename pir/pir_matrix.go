package pir

import (
	"errors"
	"log"
	"math"
	"math/rand"
	"tapir/modules/database"
	"tapir/modules/utils"
	"tapir/modules/vc"
)

// TODO replace this or put it in testing util
var masterKey utils.PRGKey = [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 'A', 'B', 'C', 'D', 'E', 'F'}

// There is no offline phase in this protocol, define dummy types
type MatrixDigest struct{}
type MatrixHintQuery struct{}
type MatrixHintResp struct{}
type MatrixHint struct{}

// Online phase types
type MatrixQuery struct {
	BitVector []bool
}
type MatrixAnswer struct {
	FlatRecords []byte
}

type MatrixServer struct {
	Db   *database.DB
	Role byte
}

type MatrixClient struct {
	N       int // Number of DB records
	Height  int
	Width   int
	RecSize int

	RandSource *rand.Rand

	// State
	queriedIdx int
}

func (s *MatrixServer) Equals(other APIRServer) (bool, error) {
	s2 := other.(*MatrixServer)
	if b, err := s.Db.Equals(s2.Db); !b {
		return false, err
	}
	if s.Role != s2.Role {
		return false, errors.New("server parameters not equal")
	}

	return true, nil
}

func (s *MatrixServer) GetVCType() vc.VcType {
	return vc.VcType(0)
}
func (s *MatrixServer) SetVC(vc.VcType) {
	return
}

func getHeightWidth(nRows int, rowLen int) (int, int) {
	// h^2 = n * rowlen
	width := int(math.Ceil(math.Sqrt(float64(nRows*rowLen)) / float64(rowLen)))
	height := (nRows-1)/width + 1

	return width, height
}

func matBoolVecProduct(db_data []byte, numRec, recSize int, bitVector []bool) []byte {
	width, height := getHeightWidth(numRec, recSize)
	out := make([]byte, width*recSize)

	//cnt := 0
	tableWidth := recSize * width
	//flatDb := db.Slice(0, db.N)
	for j := 0; j < height; j++ {
		if bitVector[j] {
			start := tableWidth * j
			length := tableWidth
			if start+length >= len(db_data) {
				length = len(db_data) - start
			}
			//psetggm.FastXorInto(out[0:length], db.Data[start:start+length], length)
			database.XorInto(out[0:length], db_data[start:start+length])
			//cnt = cnt + tableWidth
		}
	}
	return out
}

////////////////////////////////////////////////////////////
// OFFLINE PHASE
////////////////////////////////////////////////////////////

// There is no offline phase, so these functions do nothing

func SetupMatrixClient(N, recSize int, vctype vc.VcType) *MatrixClient {
	c := MatrixClient{N: N}
	c.RandSource = rand.New(utils.NewBufPRG(utils.NewPRG(&masterKey)))
	c.RecSize = recSize
	c.Width, c.Height = getHeightWidth(N, recSize)
	return &c
}

func (c *MatrixClient) RequestHint() (HintQuery, HintQuery, error) {
	return &MatrixHintQuery{}, &MatrixHintQuery{}, nil
}

func (c *MatrixClient) VerSetup(d0 Digest, d1 Digest, resp0 HintResp, resp1 HintResp) (Digest, Hint, error) {
	return &MatrixDigest{}, &MatrixHint{}, nil
}

func (s *MatrixServer) GenDigest() (Digest, error) {
	return &MatrixDigest{}, nil
}
func (s *MatrixServer) GenHint(hq HintQuery) (HintResp, error) {
	return &MatrixHintResp{}, nil
}
func (c *MatrixClient) EqualDigests(_, _ Digest) bool {
	return true
}

func (s *MatrixServer) GetDigest() Digest {
	return &APIR_MatrixDigest{}
}
func (s *MatrixServer) GetDB() *database.DB {
	return s.Db
}
func (s *MatrixServer) Update(_ []database.Update) (Nt, Qt int, dt Digest, opst []database.Update) {
	log.Fatalf("not implemented")
	return
}

func (c *MatrixClient) Query(idx int) (Query, Query, error) {
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
	return &MatrixQuery{qL}, &MatrixQuery{qR}, nil
}

func (c *MatrixClient) Reconstruct(digest Digest, hint Hint, answer0 Answer, answer1 Answer) (database.Record, error) {

	a0 := answer0.(*MatrixAnswer)
	a1 := answer1.(*MatrixAnswer)
	colNum := c.queriedIdx % c.Width

	database.XorInto(a0.FlatRecords, a1.FlatRecords)

	return a0.FlatRecords[c.RecSize*colNum : (c.RecSize * (colNum + 1))], nil
}

func (s *MatrixServer) Answer(q Query) (Answer, error) {
	return &MatrixAnswer{matBoolVecProduct(*&s.Db.Data, s.Db.N, s.Db.RecSize, q.(*MatrixQuery).BitVector)}, nil
}
func (c *MatrixClient) UpdateHint(newN0, newN1, newQ0, newQ1 int, newDigest0, newDigest1 Digest, ops0, ops1 []database.Update) (N int, Q int, d Digest, hint Hint, err error) {
	log.Fatal("not implemented yet")
	return
}
