package vc

import (
	"encoding/gob"
	"errors"
	"fmt"
	"tapir/modules/database"
	"tapir/modules/pp"
)

func init() {
	pp := PointProof{}
	gob.Register(Proof(&pp))
	pc := PPCommitment{}
	gob.Register(Commitment(&pc))
}

type PPParams struct {
	*pp.PP
}

type PPCommitment struct {
	Commitment pp.G1
}

type PPVector struct {
	Vec pp.Vec
}
type PPAggProof struct {
	Proof *pp.G1
}
type PointProof struct {
	Point pp.G1
}

func (params *PPParams) Equals(other VCParams) (bool, error) {
	if params.N != other.(*PPParams).N {
		return false, errors.New("VC Params not equal")
	}
	return true, nil
}

func (params *PPParams) Type() VcType {
	return VC_PointProof
}

func (params *PPParams) VectorFromRecords(recs []database.Record) Vector {
	if len(recs) != params.N {
		panic("Vector length does not match setup")
	}

	v := make([]*pp.Zr, params.N)

	for i, elem := range recs {
		vecElem := pp.FieldElementFromBytes(elem)
		v[i] = vecElem
	}

	return &PPVector{Vec: v}
}

func (params *PPParams) ProofToBytes(p Proof) []byte {
	return p.(*PointProof).Point.Bytes()
}

func (params *PPParams) BytesToProof(in []byte) (Proof, error) {
	p, err := pp.G1FromBytes(in)
	if err != nil {
		return nil, fmt.Errorf("error converting bytes to proof: %w", err)
	}
	return &PointProof{Point: *p}, nil
}

func SetupPointProof(n int, seed int) *PPParams {

	params := pp.NewPublicParams(n)
	return &PPParams{params}
}

func (params *PPParams) Commit(v Vector) Commitment {
	return &PPCommitment{Commitment: *pp.Commit(params.PP, v.(*PPVector).Vec)}
}
func (params *PPParams) Open(v Vector, idx int, _ Commitment) Proof {

	vec := v.(*PPVector)

	_, proof := pp.Open(params.PP, idx, vec.Vec)
	return &PointProof{Point: *proof}
}

func (params *PPParams) Verify(c Commitment, p Proof, idx int, elem database.Record) bool {
	// Making sure in index lies in the boundaries
	if !(0 <= idx && idx < params.N) {
		panic("out of range index")
	}
	err := pp.Verify(params.PP, pp.FieldElementFromBytes(elem), &p.(*PointProof).Point, &c.(*PPCommitment).Commitment, idx)

	return err == nil
}

func (params *PPParams) EqualCommitments(c1, c2 Commitment) bool {
	return c1.(*PPCommitment).Commitment.Equals(&c2.(*PPCommitment).Commitment)
}

func (params *PPParams) EqualProofs(c1, c2 Proof) bool {
	return c1.(*PointProof).Point.Equals(&c2.(*PointProof).Point)
}

func (params *PPParams) Aggregate(proofs *[]Proof, coms *[]Commitment) AggProof {
	if len(*proofs) != len(*coms) {
		panic(fmt.Sprintf("cannot aggregate %d proofs corresponding to %d commitments", len(*proofs), len(*coms)))
	}
	ps := make([]*pp.G1, len(*proofs))
	cs := make([]*pp.G1, len(*proofs))

	for i := range *proofs {
		ps[i] = &(*proofs)[i].(*PointProof).Point
		cs[i] = &(*coms)[i].(*PPCommitment).Commitment
	}

	aggP := pp.Aggregate(params.PP, cs, ps, pp.RO)

	return AggProof(aggP)
}

func (params *PPParams) VerifyAggregation(aggProof AggProof, coms *[]Commitment, indices []int, elems []database.Record) bool {
	cs := make([]*pp.G1, len(*coms))
	for i := range *coms {
		cs[i] = &(*coms)[i].(*PPCommitment).Commitment
	}
	res := pp.VerifyAggregationRecords(params.PP, indices, cs, aggProof.(*pp.G1), elems, pp.RO)
	return res == nil
}

func (params *PPParams) UpdateMulti(c Commitment, v Vector, ops []database.Update) (Commitment, Vector) {
	vec := v.(*PPVector)
	for _, op := range ops {
		fieldElem := pp.FieldElementFromBytes(op.Val)
		pp.Update(params.PP, &(c.(*PPCommitment).Commitment), vec.Vec, fieldElem, op.Idx%len(vec.Vec))

		vec.Vec[op.Idx] = fieldElem

	}
	return c, vec
}
func (params *PPParams) Update(c Commitment, v Vector, op database.Update) (Commitment, Vector) {
	vec := v.(*PPVector)
	fieldElem := pp.FieldElementFromBytes(op.Val)
	com := c.(*PPCommitment)
	pp.Update(params.PP, &(com.Commitment), vec.Vec, fieldElem, op.Idx)
	vec.Vec[op.Idx] = fieldElem

	return com, vec
}
