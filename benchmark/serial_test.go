package benchmark

import (
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"reflect"
	"tapir/modules/database"
	"tapir/modules/vc"
	"tapir/pir"
	"testing"
)

func TestServerSerialize(t *testing.T) {

	// use random seed each time
	var seed [32]byte
	// NOTE: Pointproofs take a longer time to create.
	// For tests its best to choose smaller DB size or increase test timeout.
	N := 16384
	_, err := rand.Read(seed[:])
	if err != nil {
		t.Fatal(err)
	}

	NUM_SERVERS := 2
	recSize := 32
	db := database.MakeRandomDB(seed, N, recSize)

	Q := int(math.Sqrt(float64(N)))
	if N%Q != 0 {
		t.Fatalf("Q does not divide N")
	}

	pirTypes := []pir.PirType{
		pir.PIR_MATRIX,
		pir.PIR_DPF,
		pir.PIR_SinglePass,
		pir.APIR_MATRIX,
		// pir.APIR_DPF128,
		pir.APIR_TAPIR,
	}
	vctypes := [][]vc.VcType{
		[]vc.VcType{vc.None},
		[]vc.VcType{vc.None},
		[]vc.VcType{vc.None},
		// []vc.VcType{vc.VC_MerkleTree, vc.VC_PointProof},
		[]vc.VcType{vc.VC_MerkleTree},
		// []vc.VcType{vc.None},
		[]vc.VcType{vc.VC_MerkleTree, vc.VC_PointProof},
	}
	qs := []int{
		-1,
		-1,
		Q,
		-1,
		-1,
		Q,
	}

	for i, pirType := range pirTypes {
		for _, vcType := range vctypes[i] {

			log.Println("Testing PIR type:", pirType, "with VC type:", vcType, "for N:", N, "and Q:", qs[i])

			digests := make([]pir.Digest, NUM_SERVERS)
			// hintqueries := make([]pir.HintQuery, NUM_SERVERS)
			// hintresps := make([]pir.HintResp, NUM_SERVERS)
			// queries := make([]pir.Query, NUM_SERVERS)
			// answers := make([]pir.Answer, NUM_SERVERS)

			servers := []pir.APIRServer{
				pir.NewServer(pirType, db, 0, qs[i], vcType),
				pir.NewServer(pirType, db, 1, qs[i], vcType),
			}

			for i, server := range servers {
				digests[i], err = server.GenDigest()
				if err != nil {
					t.Fatalf("error generating digest 0:%v", err)
				}
			}

			for i, server := range servers {
				file, err := os.Create(fmt.Sprintf("server%d.gob", i))
				if err != nil {
					t.Fatalf("error creating file: %v", err)
				}
				defer file.Close()
				encoder := gob.NewEncoder(file)
				err = encoder.Encode(server)
				if err != nil {
					t.Fatalf("error encoding server[%d]: %v", i, err)
				}

				// Now read back and decode
				file, err = os.Open(fmt.Sprintf("server%d.gob", i))
				if err != nil {
					t.Fatalf("error opening file: %v", err)
				}
				defer file.Close()

				decoded := reflect.New(reflect.TypeOf(server).Elem()).Interface()
				err = gob.NewDecoder(file).Decode(decoded)
				if err != nil {
					t.Fatalf("error decoding server[0]: %v", err)
				}
				decodedServer := decoded.(pir.APIRServer)

				vcType := decodedServer.GetVCType()
				decodedServer.SetVC(vcType)

				if b, err := server.Equals(decodedServer); !b {
					// if !reflect.DeepEqual(server, decodedServer) {
					t.Fatalf("Decoded server does not match original for PIR type %v, VC type %v:%e", pirTypes[i], vcType, err)
				}

				digest, err := decodedServer.GenDigest()
				if err != nil {
					t.Fatalf("error generating digest 0:%v", err)
				}
				if !reflect.DeepEqual(digests[i], digest) {
					t.Fatalf("digest not equal")
				}
			}
		}
	}
}
