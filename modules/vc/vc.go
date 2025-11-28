package vc

import (
	"encoding/gob"
	"tapir/modules/database"
)

const VC_SEED = 42

type VCParams interface {
	VectorFromRecords([]database.Record) Vector
	Commit(v Vector) Commitment
	Open(v Vector, idx int, c Commitment) Proof
	Verify(c Commitment, p Proof, idx int, elem database.Record) bool
	UpdateMulti(c Commitment, vec Vector, ops []database.Update) (Commitment, Vector)
	Update(c Commitment, vec Vector, op database.Update) (Commitment, Vector)
	ProofToBytes(p Proof) []byte
	BytesToProof(in []byte) (Proof, error)
	Type() VcType
	EqualCommitments(c0 Commitment, c1 Commitment) bool
	EqualProofs(p0 Proof, p1 Proof) bool
	Aggregate(*[]Proof, *[]Commitment) AggProof
	VerifyAggregation(AggProof, *[]Commitment, []int, []database.Record) bool
}

type AggProof interface{}

type Commitment interface{}

type Vector interface{}

type Proof interface{}

// Enum for different PIR types
type VcType int

const (
	None VcType = iota
	VC_PointProof
	VC_MerkleTree
)

func (t VcType) String() string {
	return [...]string{
		"None",
		"PointProof",
		"MerkleTree",
	}[t]
}

func NewVc(t VcType, n int) VCParams {
	switch t {
	case VC_PointProof:
		vc := SetupPointProof(n, VC_SEED)
		gob.Register(VCParams(vc))
		return vc
	case VC_MerkleTree:
		vc := SetupMerkle(n)
		gob.Register(VCParams(vc))
		return vc
	case None:
		return nil
	default:
		panic("Unknown VC type")
	}
}
