package vc

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"tapir/modules/database"
	"tapir/modules/merkle"
)

///////////////////////////////////////////////////////
// MERKLE
///////////////////////////////////////////////////////

func init() {
	mp := MerkleProof{}
	gob.Register(Proof(&mp))
	mc := MerkleCommitment{}
	gob.Register(Commitment(&mc))
}

type MerkleParams struct {
	// constant N which is the length of the vectors in the scheme
	N int
}

type MerkleCommitment struct {
	Root []byte
}

type MerkleVector struct {
	*merkle.MerkleTree
}

type MerkleProof struct {
	Proof merkle.Proof
}

type MerkleAggProof struct {
	Proofs []Proof
}

func SetupMerkle(n int) *MerkleParams {
	params := &MerkleParams{N: n}
	return params
}
func (params *MerkleParams) Equals(other VCParams) (bool, error) {
	if params.N != other.(*MerkleParams).N {
		return false, errors.New("VC Params not equal")
	}
	return true, nil
}

// Generate Merkle Tree and store in MerkleVector
func (params *MerkleParams) VectorFromRecords(v []database.Record) Vector {
	if len(v) != params.N {
		panic("Vector length does not match setup")
	}
	tree, err := merkle.NewFromRecords(&v)
	if err != nil {
		panic(err)
	}
	return &MerkleVector{tree}
}

func (params *MerkleParams) Commit(v Vector) Commitment {
	mc := MerkleCommitment{Root: v.(*MerkleVector).Root()}
	return &mc
}

func (params *MerkleParams) Open(v Vector, idx int, c Commitment) Proof {
	tree := v.(*MerkleVector)
	proof, err := tree.GenerateProofIndex(uint32(idx))
	if err != nil {
		panic(err)
	}
	return &MerkleProof{Proof: *proof}
}

func (params *MerkleParams) Verify(c Commitment, p Proof, idx int, elem database.Record) bool {
	b, err := merkle.VerifyProof(elem, &p.(*MerkleProof).Proof, uint32(idx), c.(*MerkleCommitment).Root)
	if err != nil {
		panic(err)
	}
	return b
}

// No aggregation for Merkle trees, just naive saving all proofs
func (params *MerkleParams) Aggregate(proofs *[]Proof, _ *[]Commitment) AggProof {
	return &MerkleAggProof{Proofs: *proofs}
}

// No aggregation for Merkle trees, just naive verification of all proofs
func (params *MerkleParams) VerifyAggregation(aggProof AggProof, c *[]Commitment, idxs []int, elems []database.Record) bool {
	mp := aggProof.(*MerkleAggProof)
	if len(idxs) != len(elems) || len(idxs) != len(mp.Proofs) {
		panic("Index and element length mismatch")
	}
	for i, index := range idxs {
		b, err := merkle.VerifyProof(elems[i], &mp.Proofs[i].(*MerkleProof).Proof, uint32(index), (*c)[i].(*MerkleCommitment).Root)
		if err != nil {
			panic(err)
		}
		if !b {
			return false
		}
	}
	return true
}

func (params *MerkleParams) ProofToBytes(p Proof) []byte {
	return merkle.EncodeProof(&p.(*MerkleProof).Proof)
}

func (params *MerkleParams) BytesToProof(in []byte) (Proof, error) {
	return &MerkleProof{Proof: *merkle.DecodeProof(in)}, nil
}

func (params *MerkleParams) Type() VcType {
	return VC_MerkleTree
}

func (params *MerkleParams) EqualCommitments(c1, c2 Commitment) bool {
	return bytes.Equal(c1.(*MerkleCommitment).Root, c2.(*MerkleCommitment).Root)
}

func (params *MerkleParams) EqualProofs(c1, c2 Proof) bool {
	mc1 := c1.(*MerkleProof)
	mc2 := c2.(*MerkleProof)
	if mc1.Proof.Index == mc2.Proof.Index || len(mc1.Proof.Hashes) == len(mc2.Proof.Hashes) {
		for i := 0; i < len(mc1.Proof.Hashes); i++ {
			if !bytes.Equal(mc1.Proof.Hashes[i], mc2.Proof.Hashes[i]) {
				return false
			}
		}
		return true
	}
	return false
}

// Works only for additions
func (params *MerkleParams) UpdateMulti(c Commitment, vec Vector, ops []database.Update) (Commitment, Vector) {
	root, err := vec.(*MerkleVector).MerkleTree.UpdateMulti(ops)
	if err != nil {
		log.Fatal(err)
	}
	return &MerkleCommitment{Root: root}, vec
}
func (params *MerkleParams) Update(c Commitment, vec Vector, op database.Update) (Commitment, Vector) {
	root, err := vec.(*MerkleVector).MerkleTree.Update(op)
	if err != nil {
		log.Fatal(err)
	}
	return &MerkleCommitment{Root: root}, vec.(*MerkleVector)
}
