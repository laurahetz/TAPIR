package vc

import (
	"bytes"
	"fmt"
	"log"
	"math/rand/v2"
	"reflect"
	"testing"

	"tapir/modules/database"
	"tapir/modules/utils"
)

const RECSIZE = 32

func TestVC(t *testing.T) {
	n := 1000          // 1000
	recSize := RECSIZE // NOTE: was 16 before RECSIZE, so it is the same // 128 does not work

	vcTypes := []VcType{VC_MerkleTree, VC_PointProof}
	for _, vcType := range vcTypes {

		log.Println("TestVC:", vcType)

		// Create a new PointProofParams
		vc := NewVc(vcType, n)
		vc2 := NewVc(vcType, n)
		seed := [32]byte{34}
		prg := rand.NewChaCha8(seed)
		db := database.MakeRandomRows(prg, n, recSize)
		v := vc.VectorFromRecords(db)
		v2 := vc2.VectorFromRecords(db)

		// Commit the vector
		commitment := vc.Commit(v)
		commitment2 := vc2.Commit(v)

		if !vc.EqualCommitments(commitment, commitment2) {
			t.Fatalf("commitments not equal (check via vc)")
		}
		if !vc2.EqualCommitments(commitment, commitment2) {
			t.Fatalf("commitments not equal (check via vc2)")
		}

		for i, rec := range db {
			proof := vc.Open(v, i, commitment)
			proof2 := vc2.Open(v2, i, commitment2)

			if !vc.EqualProofs(proof, proof2) {
				t.Fatalf("proofs not equal (check via vc)")
			}
			if !vc2.EqualProofs(proof, proof2) {
				t.Fatalf("proofs not equal (check via vc2)")
			}

			if !vc.Verify(commitment2, proof2, i, rec) {
				t.Fatal(vcType, ": proof did not verify but should have")
			}
			if !vc2.Verify(commitment, proof, i, rec) {
				t.Fatal(vcType, ": proof did not verify but should have")
			}
		}

	}

}

func TestProofEncodeDecode(t *testing.T) {

	n := 1000          // 1000
	recSize := RECSIZE // NOTE: was 16 before RECSIZE, so it is the same // 128 does not work

	vcTypes := []VcType{VC_MerkleTree, VC_PointProof}
	for _, vcType := range vcTypes {

		log.Println("TestProofEncodeDecode:", vcType)

		// Create a new PointProofParams
		vc := NewVc(vcType, n)
		seed := [32]byte{34}
		prg := rand.NewChaCha8(seed)
		db := database.MakeRandomRows(prg, n, recSize)
		v := vc.VectorFromRecords(db)

		// Commit the vector
		commitment := vc.Commit(v)

		for i, rec := range db {
			proof := vc.Open(v, i, nil)
			encP := vc.ProofToBytes(proof)
			decP, err := vc.BytesToProof(encP)
			if err != nil {
				t.Fatal("error BytesToProof for ", vcType, ":", err)
			}

			if vcType == VC_MerkleTree {
				ogP := proof.(*MerkleProof)
				if !reflect.DeepEqual(ogP, decP) {
					t.Fatal(vcType.String(), ": proof and decrypted proof are not equal")
				}
			} else if vcType == VC_PointProof {
				ogP := proof.(*PointProof)
				if !reflect.DeepEqual(ogP.Point, decP.(*PointProof).Point) {
					t.Fatal(vcType.String(), ": proof and decrypted proof are not equal")
				}
			}
			//log.Println("len", i, ":", lens[i])
			if !vc.Verify(commitment, proof, i, rec) {
				// TODO Verify always fails for Point Proofs although it should be successful
				t.Fatal(vcType, ": proof did not verify but should have")
			}
			// test some other index -> verify should fail
			if vc.Verify(commitment, proof, (i+1)%n, rec) {
				t.Fatal(vcType, ": proof should not have verified")
			}
		}

	}

}

func TestVCAggregation(t *testing.T) {

	n := 1000 // 1000
	numParts := 10
	partSize := n / numParts

	recSize := RECSIZE // NOTE: was 16 before RECSIZE, so it is the same // 128 does not work

	seed := [32]byte{1}
	prg := rand.NewChaCha8(seed)
	db := database.MakeRandomRows(prg, n, recSize)

	vcTypes := []VcType{VC_MerkleTree} //, VC_PointProof}

	vecs := make([]Vector, numParts)
	coms := make([]Commitment, numParts)
	proofs := make([][]Proof, numParts)

	for _, vcType := range vcTypes {

		vc := NewVc(vcType, partSize)
		log.Println("TestVCAggregation:", vcType)

		for i := 0; i < numParts; i++ {
			partition := db[i*partSize : (i+1)*partSize]
			// Create a new PointProofParams
			vecs[i] = vc.VectorFromRecords(partition)
			coms[i] = vc.Commit(vecs[i])
			proofs[i] = make([]Proof, partSize)

			for j, rec := range partition {
				proofs[i][j] = vc.Open(vecs[i], j, coms[i])
				// test proof correctness for first instance
				if vc.Verify(coms[i], proofs[i][j], (j+1)%partSize, rec) {
					t.Fatalf("%v proof %v should not have verified", vcType, j)
				}
				if !vc.Verify(coms[i], proofs[i][j], j, rec) {
					t.Fatalf("%v proof %v did not verify but should have", vcType, j)
				}
			}
		}
		// pick random index for each new experiment
		b := utils.NewBufPRG(utils.NewPRG(&utils.PRGKey{0}))
		indices := make([]int, numParts)
		toaggProofs := make([]Proof, numParts)
		aggRecs := make([]database.Record, numParts)
		for i := 0; i < numParts; i++ {
			indices[i] = b.RandInt(partSize)
			toaggProofs[i] = proofs[i][indices[i]]
			aggRecs[i] = db[i*partSize+indices[i]]
		} // Aggregate the proofs
		aggProof := vc.Aggregate(&toaggProofs, &coms)
		if aggProof == nil {
			t.Fatalf("%v aggregation failed", vcType)
		}
		if !vc.VerifyAggregation(aggProof, &coms, indices, aggRecs) {
			t.Fatalf("%v aggregation verification failed", vcType)
		}
	}
}

func TestPointProof(t *testing.T) {
	log.Println("TestPointProof")

	n := 1000
	// Create a new PointProofParams
	vc := NewVc(VC_PointProof, n)

	// Create a new PointProofVector

	seed := [32]byte{1}
	prg := rand.NewChaCha8(seed)
	records := database.MakeRandomRows(prg, n, RECSIZE)
	v := vc.VectorFromRecords(records)

	// Commit the vector
	commitment := vc.Commit(v)

	for i, rec := range records {

		// Open the vector at index 1
		proof := vc.Open(v, i, nil)

		// Verify the proof
		if !vc.Verify(commitment, proof, i, rec) {
			t.Fatal("Proof", i, "did not verify but should have")
		}

		// Try to verify the proof with wrong element
		if vc.Verify(commitment, proof, i, records[(i+1)%n]) {
			t.Fatal("Proof", i, " verified for wrong element")
		}

		// Try to verify the proof with wrong index
		if vc.Verify(commitment, proof, (i+1)%n, rec) {
			fmt.Println("Proof", i, " verified for wrong index")
		}
	}
}

func TestPointProofAggregation(t *testing.T) {

	log.Println("TestPointProofAggregation")
	n := 1000
	// Create a new PointProofParams
	vc := NewVc(VC_PointProof, n)

	// Create a new PointProofVector

	seed := [32]byte{1}
	prg := rand.NewChaCha8(seed)
	records := database.MakeRandomRows(prg, n, RECSIZE)
	v := vc.VectorFromRecords(records)

	// Commit the vector
	commitment := vc.Commit(v)

	for i, rec := range records {

		// Open the vector at index 1
		proof := vc.Open(v, i, nil)

		// Verify the proof
		if !vc.Verify(commitment, proof, i, rec) {
			t.Fatal("Proof", i, "did not verify but should have")
		}

		// Try to verify the proof with wrong element
		if vc.Verify(commitment, proof, i, records[(i+1)%n]) {
			t.Fatal("Proof", i, " verified for wrong element")
		}

		// Try to verify the proof with wrong index
		if vc.Verify(commitment, proof, (i+1)%n, rec) {
			fmt.Println("Proof", i, " verified for wrong index", i+1, "instead of", i)
		}
	}
}

func TestVCUpdate(t *testing.T) {
	n := 1000
	numUpdates := 100
	recSize := RECSIZE

	seed := [32]byte{42}
	prg := rand.NewChaCha8(seed)

	vcTypes := []VcType{VC_MerkleTree, VC_PointProof}
	for _, vcType := range vcTypes {
		rows := database.MakeRandomRows(prg, n, recSize)
		db := database.DBFromRecords(rows)

		ops := database.MakeRandomUpdates(
			prg,
			n,
			numUpdates,
			recSize,
			[]database.OpType{database.EDIT},
			// []database.OpType{database.ADD, database.EDIT},
		)

		db2 := database.DB{N: db.N, RecSize: db.RecSize, Capacity: db.Capacity, Data: db.Data}

		log.Println("TestVCUpdate:", vcType)

		// Create a new PointProofParams
		vc := NewVc(vcType, n)
		v := vc.VectorFromRecords(rows)

		// Commit the vector
		c := vc.Commit(v)

		for i, op := range ops {
			var oldVal = make([]byte, recSize)

			db2.Update([]database.Update{op})
			newVal := db2.GetRecord(i)

			c, v := vc.Update(c, v, op)

			if i < db.N {
				oldVal = db.GetRecord(i)
			}

			proof := vc.Open(v, i, c)

			if !vc.Verify(c, proof, i, newVal) {
				t.Fatal(vcType, ": proof did not verify but should have")
			}
			if !bytes.Equal(make([]byte, recSize), oldVal) && !bytes.Equal(oldVal, newVal) {
				if vc.Verify(c, proof, i, oldVal) {
					t.Fatal(vcType, ": old value incorrectly verified with new proof and commitment")
				}
			}
		}
	}

}

// func TestVCUpdateMulti(t *testing.T) {
// 	n := 1000
// 	numUpdates := 100
// 	recSize := RECSIZE

// 	seed := [32]byte{34}
// 	prg := rand.NewChaCha8(seed)
// 	rows := database.MakeRandomRows(prg, n, recSize)
// 	db := database.DBFromRecords(rows)

// 	ops := database.MakeRandomUpdates(
// 		prg,
// 		n,
// 		numUpdates,
// 		recSize,
// 		[]database.OpType{database.EDIT},
// 		// []database.OpType{database.ADD, database.EDIT},
// 	)
// 	db2 := database.DB{N: db.N, RecSize: db.RecSize, Capacity: db.Capacity, Data: db.Data}
// 	db2.Update(ops)

// 	vcTypes := []VcType{VC_MerkleTree, VC_PointProof}
// 	for _, vcType := range vcTypes {

// 		log.Println("TestVCUpdate:", vcType)

// 		// Create a new PointProofParams
// 		vc := NewVc(vcType, n)
// 		v := vc.VectorFromRecords(rows)

// 		// Commit the vector
// 		commitment := vc.Commit(v)
// 		c2, v := vc.UpdateMulti(commitment, v, ops)

// 		rows2 := db2.GetRecords(0, db2.N)

// 		v2 := vc.VectorFromRecords(rows2)

// 		for i := range db2.N {
// 			var oldVal = make([]byte, recSize)
// 			newVal := db2.GetRecord(i)
// 			if i < db.N {
// 				oldVal = db.GetRecord(i)
// 			}

// 			proof := vc.Open(v2, i, c2)

// 			if !vc.Verify(c2, proof, i, newVal) {
// 				t.Fatal(vcType, ": proof did not verify but should have")
// 			}
// 			if !bytes.Equal(make([]byte, recSize), oldVal) && !bytes.Equal(oldVal, newVal) {
// 				if vc.Verify(c2, proof, i, oldVal) {
// 					t.Fatal(vcType, ": proof verified but should not have")
// 				}
// 			}
// 		}
// 	}

// }

func ExamplePointProof() {

	n := 3
	// Create a new PointProofParams
	vc := NewVc(VC_PointProof, n)

	// Create a new PointProofVector
	v := vc.VectorFromRecords([]database.Record{
		[]byte{0, 0, 0, 0},
		[]byte{1, 1, 1, 1},
		[]byte{2, 2, 2, 2},
	})

	// Commit the vector
	commitment := vc.Commit(v)

	// Open the vector at index 1
	proof := vc.Open(v, 1, nil)

	// Verify the proof
	if vc.Verify(commitment, proof, 1, []byte{1, 1, 1, 1}) {
		fmt.Println("Proof verified")
	} else {
		fmt.Println("Proof not verified")
	}

	// Try to verify the proof with wrong element
	if vc.Verify(commitment, proof, 1, []byte{2, 2, 2, 2}) {
		fmt.Println("Proof verified")
	} else {
		fmt.Println("Proof not verified")
	}

	// Try to verify the proof with wrong index
	if vc.Verify(commitment, proof, 2, []byte{1, 1, 1, 1}) {
		fmt.Println("Proof verified")
	} else {
		fmt.Println("Proof not verified")
	}

	// Output:
	// Proof verified
	// Proof not verified
	// Proof not verified
}

func ExampleMerkleProof() {

	n := 10
	// Create new MerkleParams
	vc := NewVc(VC_MerkleTree, n)

	// Create a new PointProofVector
	v := vc.VectorFromRecords([]database.Record{
		[]byte{0, 0, 0, 0},
		[]byte{1, 1, 1, 1},
		[]byte{2, 2, 2, 2},
		[]byte{3, 3, 3, 3},
		[]byte{4, 4, 4, 4},
		[]byte{5, 5, 5, 5},
		[]byte{6, 6, 6, 6},
		[]byte{7, 7, 7, 7},
		[]byte{8, 8, 8, 8},
		[]byte{9, 9, 9, 9},
	})

	// Commit the vector
	commitment := vc.Commit(v)

	// Open the vector at index 1
	proof := vc.Open(v, 1, commitment)

	// Verify the proof
	if vc.Verify(commitment, proof, 1, []byte{1, 1, 1, 1}) {
		fmt.Println("Proof verified")
	} else {
		fmt.Println("Proof not verified")
	}

	// Try to verify the proof with wrong element
	if vc.Verify(commitment, proof, 1, []byte{2, 2, 2, 2}) {
		fmt.Println("Proof verified")
	} else {
		fmt.Println("Proof not verified")
	}

	// Try to verify the proof with wrong index
	if vc.Verify(commitment, proof, 2, []byte{1, 1, 1, 1}) {
		fmt.Println("Proof verified")
	} else {
		fmt.Println("Proof not verified")
	}

	// Output:
	// Proof verified
	// Proof not verified
	// Proof not verified
}
