package pir

import (
	"tapir/modules/database"
	"tapir/modules/vc"
)

const SecParam = 128 // Security parameter in bits
// const RECSIZE = 32
const SAFE = false

// Offline phase types
type Digest interface{}
type HintQuery interface{}
type HintResp interface{}
type Hint interface{}

// Online phase types
type Query interface{}
type Answer interface{}

type APIRServer interface {
	GenDigest() (Digest, error)
	GenHint(hq HintQuery) (HintResp, error)
	Answer(q Query) (Answer, error)
	Equals(other APIRServer) (bool, error)
	GetVCType() vc.VcType
	SetVC(vc.VcType)
	GetDigest() Digest
	GetDB() *database.DB
	Update(ops []database.Update) (Nt, Qt int, dt Digest, opst []database.Update)
}

type APIRClient interface {
	RequestHint() (HintQuery, HintQuery, error)
	VerSetup(d0 Digest, d1 Digest, resp0 HintResp, resp1 HintResp) (Digest, Hint, error)
	Query(i int) (Query, Query, error)
	Reconstruct(digest Digest, hint Hint, a0 Answer, a1 Answer) (database.Record, error)
	EqualDigests(d0, d1 Digest) bool
	UpdateHint(newN0, newN1, newQ0, newQ1 int, newDigest0, newDigest1 Digest, ops0, ops1 []database.Update) (int, int, Digest, Hint, error)
}

// Enum for different PIR types
type PirType int

const (
	PIR_MATRIX PirType = iota
	PIR_DPF
	PIR_SinglePass
	APIR_MATRIX
	APIR_DPF128
	APIR_TAPIR
)

func (t PirType) String() string {
	return [...]string{
		"PIR_MATRIX",     // 0
		"PIR_DPF",        // 1
		"PIR_SinglePass", // 2
		"APIR_Matrix",    // 3
		"APIR_DPF128",    // 4
		"APIR_TAPIR",     // 5
	}[t]
}

// Usage: Q is -1 if not needed
func NewClient(t PirType, n int, Q int, recSize int, vctype vc.VcType) APIRClient {
	switch t {
	case PIR_DPF:
		if Q != -1 {
			panic("DPF does not use Q")
		}
		return &DPFClient{N: n}
	case PIR_MATRIX:
		if Q != -1 {
			panic("DPF does not use Q")
		}
		return SetupMatrixClient(n, recSize, vctype)
	case PIR_SinglePass:
		if n%Q != 0 {
			panic("Q does not divide N")
		}
		if Q < 1 {
			panic("Q is smaller than 1")
		}
		return &SinglePassClient{N: n, Q: Q, M: n / Q}
	case APIR_TAPIR:
		if n%Q != 0 {
			panic("Q does not divide N")
		}
		return NewTAPIRClient(n, Q, recSize, vctype)
	case APIR_DPF128:
		if Q != -1 {
			panic("DPF does not use Q")
		}
		panic("Unknown PIR type")
		//return //&DPF128Client{N: n}
	case APIR_MATRIX:
		if Q != -1 {
			panic("APIR_Matrix does not use Q")
		}
		return SetupAPIR_MatrixClient(n, recSize, vctype)
	default:
		panic("Unknown PIR type")
	}
}

// Usage: Q is -1 if not needed
func NewServer(t PirType, db *database.DB, role int, Q int, vctype vc.VcType) APIRServer {
	switch t {
	case PIR_DPF:
		if Q != -1 {
			panic("DPF does not use Q")
		}
		return &DPFServer{Db: db}
	case PIR_MATRIX:
		if Q != -1 {
			panic("Matrix does not use Q")
		}
		return &MatrixServer{Db: db}
	case PIR_SinglePass:
		if db.N%Q != 0 {
			panic("Q does not divide N")
		}
		return &SinglePassServer{Db: db, Q: Q, M: db.N / Q}
	case APIR_TAPIR:
		if db.N%Q != 0 {
			panic("Q does not divide N")
		}
		return NewTAPIRServer(db, Q, role, vctype)
	case APIR_DPF128:
		if Q != -1 {
			panic("DPF does not use Q")
		}
		panic("Unknown PIR type")
		//return &DPF128Server{Db: db, Role: byte(role)}
	case APIR_MATRIX:
		if Q != -1 {
			panic("APIR_Matrix does not use Q")
		}
		return SetupAPIR_MatrixServer(db, vctype)
	default:
		panic("Unknown PIR type")
	}
}
