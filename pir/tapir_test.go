package pir

import (
	"bytes"
	"log"
	"math/rand"
	rand2 "math/rand/v2"
	"tapir/modules/database"
	"tapir/modules/vc"
	"testing"
)

func TestTapirNonRandom(t *testing.T) {

	// Tests that the TAPIR protocol OFFLINE PHASE does not crash
	n := 100
	recSize := 16
	Q := 4

	db := database.MakeNumberDB(n, recSize)
	db2 := database.MakeNumberDB(n, recSize)

	for _, vctype := range []vc.VcType{vc.VC_MerkleTree, vc.VC_PointProof} {
		// NOTE MAY FAIL ON RECORDS OF LESS THAN 128 BITS DUE TO SIMD INSTRUCTIONS
		log.Println("TestTapirNonRandom with VC Type:", vctype)

		server0 := NewServer(APIR_TAPIR, db, 0, Q, vctype).(*TAPIRServer)
		server1 := NewServer(APIR_TAPIR, db2, 1, Q, vctype).(*TAPIRServer)

		client := NewClient(APIR_TAPIR, db.N, Q, recSize, vctype)

		// Generate a digest for the database
		d0, err := server0.GenDigest()
		if err != nil {
			t.Fatal(err)
		}
		d1, err := server1.GenDigest()
		if err != nil {
			t.Fatal(err)
		}
		if !client.(*TAPIRClient).EqualDigests(d0, d1) {
			t.Fatal("digests differ")
		}

		// Request a hint from the server
		hq0, hq1, err := client.RequestHint()
		if err != nil {
			t.Fatal(err)
		}
		// Generate a hint for the database
		hint0, err := server0.GenHint(hq0)
		if err != nil {
			t.Fatal(err)
		}
		hint1, err := server1.GenHint(hq1)
		if err != nil {
			t.Fatal(err)
		}

		// Verify setup
		digest, hint, err := client.VerSetup(d0, d1, hint0, hint1)
		if err != nil {
			t.Fatal(err)
		}

		// ONLINE
		for i := range n {
			// // Generate a query for record 1
			query0, query1, err := client.Query(i)
			if err != nil {
				t.Fatal(err)
			}

			// // Answer the query
			answer0, err := server0.Answer(query0)
			if err != nil {
				t.Fatal(err)
			}
			answer1, err := server1.Answer(query1)
			if err != nil {
				t.Fatal(err)
			}
			// Reconstruct the record
			// The DPF PIR protocol does not have a digest or hint, so we pass in a
			// dummy hint and digest values, this allows us to use the same TAPIR API
			record, err := client.Reconstruct(digest, hint, answer0, answer1)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(record, db.GetRecord(i)) {
				log.Println("retrieved:\t", record)
				log.Println("requested:\t", db.GetRecord(i))
				t.Fatal("retrieved record for ", i, " is incorrect")
			}
		}

		log.Println("Passed TAPIR small non-random test for VC type:", vctype, "now doing next type...")
	}
}

func TestTapirRandomDB(t *testing.T) {

	n := 4096
	recSize := 16
	Q := 32
	var seed [32]byte
	_, err := rand.Read(seed[:])
	if err != nil {
		t.Fatal(err)
	}
	db := database.MakeRandomDB(seed, n, recSize)
	db2 := database.MakeRandomDB(seed, n, recSize)

	for _, vctype := range []vc.VcType{vc.VC_MerkleTree, vc.VC_PointProof} {
		// NOTE MAY FAIL ON RECORDS OF LESS THAN 128 BITS DUE TO SIMD INSTRUCTIONS
		log.Println("TestTapirRandomDB with VC Type:", vctype)

		server0 := NewServer(APIR_TAPIR, db, 0, Q, vctype).(*TAPIRServer)
		server1 := NewServer(APIR_TAPIR, db2, 1, Q, vctype).(*TAPIRServer)

		client := NewClient(APIR_TAPIR, db.N, Q, recSize, vctype)

		// Generate a digest for the database
		d0, err := server0.GenDigest()
		if err != nil {
			t.Fatal(err)
		}
		d1, err := server1.GenDigest()
		if err != nil {
			t.Fatal(err)
		}

		// Request a hint from the server
		hq0, hq1, err := client.RequestHint()
		if err != nil {
			t.Fatal(err)
		}

		// Generate a hint for the database
		hint0, err := server0.GenHint(hq0)
		if err != nil {
			t.Fatal(err)
		}
		hint1, err := server1.GenHint(hq1)
		if err != nil {
			t.Fatal(err)
		}

		// Verify setup
		digest, hint, err := client.VerSetup(d0, d1, hint0, hint1)
		if err != nil {
			t.Fatal(err)
		}

		// ONLINE
		for i := range n {
			// // Generate a query for record 1
			query0, query1, err := client.Query(i)
			if err != nil {
				t.Fatal(err)
			}

			// // Answer the query
			answer0, err := server0.Answer(query0)
			if err != nil {
				t.Fatal(err)
			}
			answer1, err := server1.Answer(query1)
			if err != nil {
				t.Fatal(err)
			}

			// // Reconstruct the record
			// // The DPF PIR protocol does not have a digest or hint, so we pass in a
			// // dummy hint and digest values, this allows us to use the same TAPIR API
			record, err := client.Reconstruct(digest, hint, answer0, answer1)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(record, db.GetRecord(i)) {
				log.Println("retrieved:\t", record)
				log.Println("requested:\t", db.GetRecord(i))
				t.Fatal("retrieved record is incorrect")
			}
		}

		log.Println("Passed TAPIR randomized test for VC type:", vctype, "now doing next type...")

	}
}

func TestTapirSingleUpdates(t *testing.T) {

	for _, vctype := range []vc.VcType{vc.VC_MerkleTree, vc.VC_PointProof} {
		log.Println("TestTapirRandomUpdates with VC Type:", vctype)

		n := 1024
		numUpdates := 10
		recSize := 16
		Q := 32

		var seed1 [32]byte
		_, err := rand.Read(seed1[:])
		if err != nil {
			t.Fatal(err)
		}
		db0 := database.MakeRandomDB(seed1, n, recSize)
		db1 := database.MakeRandomDB(seed1, n, recSize)

		seed := [32]byte{42}
		prg0 := rand2.NewChaCha8(seed)
		prg1 := rand2.NewChaCha8(seed)
		ops0 := database.MakeRandomUpdates(prg0, n, numUpdates, recSize, []database.OpType{database.ADD, database.EDIT})
		ops1 := database.MakeRandomUpdates(prg1, n, numUpdates, recSize, []database.OpType{database.ADD, database.EDIT})

		server0 := NewServer(APIR_TAPIR, db0, 0, Q, vctype).(*TAPIRServer)
		server1 := NewServer(APIR_TAPIR, db1, 1, Q, vctype).(*TAPIRServer)

		client := NewClient(APIR_TAPIR, db0.N, Q, recSize, vctype)

		// Generate a digest for the database
		d0, err := server0.GenDigest()
		if err != nil {
			t.Fatal(err)
		}
		d1, err := server1.GenDigest()
		if err != nil {
			t.Fatal(err)
		}
		if !client.(*TAPIRClient).EqualDigests(d0, d1) {
			t.Fatal("digests differ")
		}

		// Request a hint from the server
		hq0, hq1, err := client.RequestHint()
		if err != nil {
			t.Fatal(err)
		}
		// Generate a hint for the database
		hint0, err := server0.GenHint(hq0)
		if err != nil {
			t.Fatal(err)
		}
		hint1, err := server1.GenHint(hq1)
		if err != nil {
			t.Fatal(err)
		}

		// Verify setup
		_, _, err = client.VerSetup(d0, d1, hint0, hint1)
		if err != nil {
			t.Fatal(err)
		}

		// ONLINE
		for i, op := range ops1 {

			if op.Op == database.EDIT {
				log.Println("Idx:", op.Idx, "\tOp:", op.Op)
				log.Println("old value:\t", server0.Db.GetRecord(op.Idx))
				log.Println("new value:\t", op.Val)
			} else {
				log.Println("Idx:", op.Idx, "\tOp:", op.Op)
				log.Println("new value:\t", op.Val)
			}

			if !ops0[i].Equals(ops1[i]) {
				t.Fatal("error creating duplicate update sets")
			}

			// SERVER UPDATE
			N0, Q0, d0, opsDelta0 := server0.Update([]database.Update{ops0[i]})
			N1, Q1, d1, opsDelta1 := server1.Update([]database.Update{ops1[i]})

			// Add checks for N values
			if N0 != N1 {
				t.Fatal("updated DB sizes not equal")
			}
			// Add checks for Q values
			if Q0 != Q1 {
				t.Fatal("updated Q values not equal")
			}

			// Check if digests are equal
			if !client.EqualDigests(d0, d1) {
				t.Fatal("updated digests not equal")
			}

			// Check if update operations are equal
			if len(opsDelta0) != len(opsDelta1) {
				t.Fatal("update operations length not equal")
			}
			for j := range opsDelta0 {
				if !opsDelta0[j].Equals(opsDelta1[j]) {
					t.Fatal("update operations not equal")
				}
			}

			// CLIENT UPDATE
			N, Q, digest, hint, err := client.(*TAPIRClient).UpdateHint(N0, N1, Q0, Q1, d0, d1, opsDelta0, opsDelta1)
			if err != nil {
				t.Fatal(err)
			}

			if N != client.(*TAPIRClient).N || client.(*TAPIRClient).N != server0.Db.N || server0.Db.N != server1.Db.N {
				t.Fatalf("db sizes not equal after update")
			}
			if Q != client.(*TAPIRClient).Q || client.(*TAPIRClient).Q != server0.Q || server0.Q != server1.Q {
				t.Fatalf("num partitions not equal after update")
			}

			idx := int(prg0.Uint64() % uint64(len(opsDelta0)))
			qryIdx := opsDelta0[idx].Idx

			// // Generate a query for the updated object
			query0, query1, err := client.Query(qryIdx)
			if err != nil {
				t.Fatal(err)
			}

			// // Answer the query
			answer0, err := server0.Answer(query0)
			if err != nil {
				t.Fatal(err)
			}
			answer1, err := server1.Answer(query1)
			if err != nil {
				t.Fatal(err)
			}
			// Reconstruct the record
			record, err := client.Reconstruct(digest, hint, answer0, answer1)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(record, server0.Db.GetRecord(qryIdx)) {
				log.Println("retrieved:\t", record)
				log.Println("requested:\t", server0.Db.GetRecord(qryIdx))
				t.Fatal("retrieved record for ", qryIdx, " is incorrect")
			}
		}

		log.Println("Passed TAPIR small non-random test for VC type:", vctype, "now doing next type...")
	}
}

func TestTapirBatchUpdates(t *testing.T) {

	for _, vctype := range []vc.VcType{vc.VC_MerkleTree, vc.VC_PointProof} {
		log.Println("TestTapirRandomUpdates with VC Type:", vctype)

		n := 1024
		numUpdates := 10
		recSize := 16
		Q := 32

		var seed1 [32]byte
		_, err := rand.Read(seed1[:])
		if err != nil {
			t.Fatal(err)
		}
		db0 := database.MakeRandomDB(seed1, n, recSize)
		db1 := database.MakeRandomDB(seed1, n, recSize)

		seed := [32]byte{42}
		prg0 := rand2.NewChaCha8(seed)
		prg1 := rand2.NewChaCha8(seed)
		ops0 := database.MakeRandomUpdates(prg0, n, numUpdates, recSize, []database.OpType{database.ADD, database.EDIT})
		ops1 := database.MakeRandomUpdates(prg1, n, numUpdates, recSize, []database.OpType{database.ADD, database.EDIT})

		server0 := NewServer(APIR_TAPIR, db0, 0, Q, vctype).(*TAPIRServer)
		server1 := NewServer(APIR_TAPIR, db1, 1, Q, vctype).(*TAPIRServer)

		client := NewClient(APIR_TAPIR, db0.N, Q, recSize, vctype)

		// Generate a digest for the database
		d0, err := server0.GenDigest()
		if err != nil {
			t.Fatal(err)
		}
		d1, err := server1.GenDigest()
		if err != nil {
			t.Fatal(err)
		}
		if !client.(*TAPIRClient).EqualDigests(d0, d1) {
			t.Fatal("digests differ")
		}

		// Request a hint from the server
		hq0, hq1, err := client.RequestHint()
		if err != nil {
			t.Fatal(err)
		}
		// Generate a hint for the database
		hint0, err := server0.GenHint(hq0)
		if err != nil {
			t.Fatal(err)
		}
		hint1, err := server1.GenHint(hq1)
		if err != nil {
			t.Fatal(err)
		}

		// Verify setup
		_, _, err = client.VerSetup(d0, d1, hint0, hint1)
		if err != nil {
			t.Fatal(err)
		}

		// SERVER UPDATE
		N0, Q0, d0, opsDelta0 := server0.Update(ops0)
		N1, Q1, d1, opsDelta1 := server1.Update(ops1)

		// Add checks for N values
		if N0 != N1 {
			t.Fatal("updated DB sizes not equal")
		}
		// Add checks for Q values
		if Q0 != Q1 {
			t.Fatal("updated Q values not equal")
		}

		// Check if digests are equal
		if !client.EqualDigests(d0, d1) {
			t.Fatal("updated digests not equal")
		}

		// Check if update operations are equal
		if len(opsDelta0) != len(opsDelta1) {
			t.Fatal("update operations length not equal")
		}
		for j := range opsDelta0 {
			if !opsDelta0[j].Equals(opsDelta1[j]) {
				t.Fatal("update operations not equal")
			}
		}

		// CLIENT UPDATE
		N, Q, digest, hint, err := client.(*TAPIRClient).UpdateHint(N0, N1, Q0, Q1, d0, d1, opsDelta0, opsDelta1)
		if err != nil {
			t.Fatal(err)
		}

		if N != client.(*TAPIRClient).N || client.(*TAPIRClient).N != server0.Db.N || server0.Db.N != server1.Db.N {
			t.Fatalf("db sizes not equal after update")
		}
		if Q != client.(*TAPIRClient).Q || client.(*TAPIRClient).Q != server0.Q || server0.Q != server1.Q {
			t.Fatalf("num partitions not equal after update")
		}

		// ONLINE
		for i, _ := range opsDelta0 {

			// log.Println("Idx:", op.Idx, "\tOp:", op.Op)
			// log.Println("old value:\t", server0.Db.GetRecord(op.Idx))
			// log.Println("new value:\t", op.Val)

			if !opsDelta0[i].Equals(opsDelta1[i]) {
				t.Fatal("error creating duplicate update sets")
			}
			qryIdx := opsDelta0[i].Idx

			// // Generate a query for the updated object
			query0, query1, err := client.Query(qryIdx)
			if err != nil {
				t.Fatal(err)
			}

			// // Answer the query
			answer0, err := server0.Answer(query0)
			if err != nil {
				t.Fatal(err)
			}
			answer1, err := server1.Answer(query1)
			if err != nil {
				t.Fatal(err)
			}
			// Reconstruct the record
			record, err := client.Reconstruct(digest, hint, answer0, answer1)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(record, server0.Db.GetRecord(qryIdx)) {
				log.Println("retrieved:\t", record)
				log.Println("requested:\t", server0.Db.GetRecord(qryIdx))
				t.Fatal("retrieved record for ", qryIdx, " is incorrect")
			}
		}

		log.Println("Passed TAPIR small non-random test for VC type:", vctype, "now doing next type...")
	}
}
