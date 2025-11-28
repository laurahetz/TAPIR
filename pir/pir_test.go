package pir

// import (
// 	"bytes"
// 	"log"
// 	"math"
// 	"math/rand"
// 	"tapir/modules/database"
// 	"tapir/modules/vc"
// 	"testing"
// )

// func TestPIR_RandomDB(t *testing.T) {

// 	// use random seed each time
// 	var seed [32]byte
// 	N := 100
// 	_, err := rand.Read(seed[:])
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	recSize := 32
// 	db := database.MakeRandomDB(seed, N, recSize)

// 	Q := int(math.Sqrt(float64(N)))
// 	if N%Q != 0 {
// 		t.Fatalf("Q does not divide N")
// 	}

// 	pirTypes := []PirType{
// 		// PIR_MATRIX,
// 		// PIR_DPF,
// 		PIR_SinglePass,
// 		// APIR_MATRIX,
// 		// APIR_DPF128,
// 		APIR_TAPIR,
// 	}
// 	vctypes := [][]vc.VcType{
// 		[]vc.VcType{vc.None},
// 		// []vc.VcType{vc.None},
// 		// []vc.VcType{vc.None},
// 		// []vc.VcType{vc.VC_PointProof, vc.VC_MerkleTree},
// 		// []vc.VcType{vc.None},
// 		[]vc.VcType{vc.VC_PointProof, vc.VC_MerkleTree},
// 	}
// 	qs := []int{
// 		// -1,
// 		// -1,
// 		Q,
// 		// -1,
// 		// -1,
// 		Q,
// 	}

// 	for i := range pirTypes {
// 		for _, vcType := range vctypes[i] {

// 			log.Println("Testing PIR type:", pirTypes[i], "with VC type:", vcType, "for N:", N, "and Q:", qs[i])

// 			// Create a new PIR server
// 			server0 := NewServer(pirTypes[i], db, 0, qs[i], vcType)
// 			server1 := NewServer(pirTypes[i], db, 1, qs[i], vcType)

// 			// Create a new PIR client (pass in PirType and N)
// 			client := NewClient(pirTypes[i], db.N, qs[i], recSize, vcType)

// 			d0, err := server0.GenDigest()
// 			if err != nil {
// 				t.Fatalf("error generating digest 0:%v", err)
// 			}
// 			d1, err := server1.GenDigest()
// 			if err != nil {
// 				t.Fatalf("error generating digest 1:%v", err)
// 			}

// 			hq0, hq1, err := client.RequestHint()
// 			if err != nil {
// 				t.Fatalf("error creating hint request:%v", err)
// 			}
// 			h0, err := server0.GenHint(hq0)
// 			if err != nil {
// 				t.Fatalf("error answering hint request:%v", err)
// 			}
// 			h1, err := server1.GenHint(hq1)
// 			if err != nil {
// 				t.Fatalf("error answering hint request:%v", err)
// 			}
// 			// Only need to get Parameters (= HInt) from one server to initialize client
// 			d, h, err := client.VerSetup(d0, d1, h0, h1)
// 			if err != nil {
// 				t.Fatalf("error verifying setup:%v", err)
// 			}
// 			for j := 0; j < N; j++ {
// 				// Generate a query for record 1
// 				query0, query1, err := client.Query(j)
// 				if err != nil {
// 					t.Fatalf("error generating query:%v", err)
// 				}

// 				// Answer the query
// 				answer0, err := server0.Answer(query0)
// 				if err != nil {
// 					t.Fatalf("error answering query 0:%v", err)
// 				}
// 				answer1, err := server1.Answer(query1)
// 				if err != nil {
// 					t.Fatalf("error answering query 1: %v", err)
// 				}

// 				// Reconstruct the record
// 				record, err := client.Reconstruct(d, h, answer0, answer1)
// 				if err != nil {
// 					t.Fatalf("test failed: %v", err)
// 				}
// 				target := db.GetRecord(j)
// 				if !bytes.Equal(record, target) {
// 					t.Log("Correct record:\t", target)
// 					t.Log("Reconstructed record:\t", record)
// 					t.Fatalf("Failure, records do not match!")
// 				}

// 			}
// 		}
// 	}
// }
