package main

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	rand2 "math/rand/v2"
	"os"
	"sync"
	"tapir/benchmark"
	"tapir/modules/database"
	"tapir/modules/vc"
	"tapir/pir"
	"time"
)

const (
	defaultConfig         = "app/configs.json"
	defaultOut            = "app/results_offline.csv"
	defaultPreProcessPath = "app/preprocessing/"
	NUM_SERVERS           = 2
	SAFE                  = false
)

var (
	pathRead          = flag.String("path", defaultConfig, "path for reading benchmark configs.")
	pathWrite         = flag.String("out", defaultOut, "path for writing benchmark results.")
	printToCMD        = flag.Bool("print", false, "print results to command line.")
	pathPreprocessing = flag.String("file", "", "path to where files for offline phase are stored/read from")
)

func main() {

	///////////////////////////////////////////////////////////////////
	// PROCESS ARGS ///////////////////////////////////////////////////
	flag.Parse()
	configs := benchmark.ReadBenchConfigs(*pathRead)

	var wg sync.WaitGroup
	var start time.Time

	seed := [32]byte{42}

	// Open file for writing
	file, err := os.Create(*pathWrite)
	if err != nil {
		log.Fatal("error creating file", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// write column headers for UPDATES
	writer.Write(benchmark.HeadersUpdate)

	for _, config := range configs.Configs {
		// Create a new database of random records
		dbs := []*database.DB{
			database.MakeRandomDB(seed, config.DbSize, config.RecSize),
			database.MakeRandomDB(seed, config.DbSize, config.RecSize),
		}
		exp := benchmark.NewExperiment(&config)

		///////////////////////////////////////////////////////////////////
		// INITIALIZE CLIENT & SERVERS ////////////////////////////////////

		// Create a new APIR server
		servers := make([]pir.APIRServer, NUM_SERVERS)
		for i := range NUM_SERVERS {
			servers[i] = pir.NewServer(pir.PirType(exp.PirType), dbs[i], i, exp.NumParts, vc.VcType(exp.VcType))
		}

		// Create a new APIR client (pass in PirType and N)
		client := pir.NewClient(pir.PirType(exp.PirType), exp.DbSize, exp.NumParts, exp.RecSize, vc.VcType(exp.VcType))

		///////////////////////////////////////////////////////////////////
		// OFFLINE PHASE //////////////////////////////////////////////////

		digests := make([]pir.Digest, NUM_SERVERS)
		wg.Add(NUM_SERVERS)
		fmt.Println("Start GenDigest:")
		start = time.Now()
		for i := range NUM_SERVERS {
			go func(i int) {
				digests[i], err = servers[i].GenDigest()
				if err != nil {
					log.Fatalln("Error in GenDigest: ", err)
				}
				wg.Done()
				// log.Println("GenDigest: thread", i, "did work")
			}(i)
		}
		wg.Wait()
		exp.RT["GenDigest"] += time.Since(start)
		fmt.Println("Finished GenDigest in ", exp.RT["GenDigest"], ". Start RequestHint.")

		// Request a hint from the server
		start = time.Now()
		hintReq0, hintReq1, _ := client.RequestHint()
		exp.RT["RequestHint"] += time.Since(start)
		hintReqs := []pir.HintQuery{hintReq0, hintReq1}

		fmt.Println("Finished RequestHint in ", exp.RT["RequestHint"], ". Start HintResp.")
		// Generate a hint for the database
		wg.Add(NUM_SERVERS)
		hintResps := make([]pir.HintResp, NUM_SERVERS)

		start = time.Now()
		for i := range NUM_SERVERS {
			go func(i int) {
				hintResps[i], err = servers[i].GenHint(hintReqs[i])
				if err != nil {
					log.Fatalln("Error in GenHint: ", err)
				}
				wg.Done()
				// log.Println("GenHint: thread", i, "did work")
			}(i)
		}
		wg.Wait()
		exp.RT["GenHint"] += time.Since(start)
		fmt.Println("Finished GenHint in ", exp.RT["GenHint"], ". Start VerSetup.")

		// Verify the setup
		start = time.Now()
		_, _, err := client.VerSetup(digests[0], digests[1], hintResps[0], hintResps[1])
		if err != nil {
			log.Fatalln("Error in VerSetup: ", err)
		}
		exp.RT["VerSetup"] += time.Since(start)
		fmt.Println("Finished VerSetup in ", exp.RT["VerSetup"], ". Start Query.")
		if *pathPreprocessing != "" {
			for i := range NUM_SERVERS {
				// filename based on config params
				preprocessFilePath := fmt.Sprintf("%s%d_%d_%d_%d_%d.s%d", *pathPreprocessing,
					config.PirType, config.VcType, config.DbSize, config.NumParts, config.RecSize, i)
				preprocessFile, err := os.Create(preprocessFilePath)
				if err != nil {
					log.Fatalf("error creating preprocessing file: %v", err)
				}
				defer preprocessFile.Close()

				encoder := gob.NewEncoder(preprocessFile)
				err = encoder.Encode(servers[0])
				if err != nil {
					log.Fatalf("error encoding server[0]: %v", err)
				}
			}
		}

		///////////////////////////////////////////////////////////////////
		// ONLINE PHASE ///////////////////////////////////////////////////
		for rep := 0; rep < int(config.Repetitions); rep++ {

			// UPDATES ///////////////////////////////////////////////////

			// Make Updates
			// get random int
			var seed [32]byte
			_, err := rand.Read(seed[:])
			if err != nil {
				log.Fatal(err)
			}

			upTypes := benchmark.GetUpdateTypesFromConfig(config.UpdateTypes)

			prg0 := rand2.NewChaCha8(seed)
			prg1 := rand2.NewChaCha8(seed)
			ops := [][]database.Update{
				database.MakeRandomUpdates(prg0, exp.DbSize, config.NumUpdates, config.RecSize, upTypes),
				database.MakeRandomUpdates(prg1, exp.DbSize, config.NumUpdates, config.RecSize, upTypes),
			}

			// SERVER UPDATE
			wg.Add(NUM_SERVERS)
			nsUpdate := make([]int, NUM_SERVERS)
			qsUpdate := make([]int, NUM_SERVERS)
			opsUpdate := make([][]database.Update, NUM_SERVERS)
			digestsUpdate := make([]pir.Digest, NUM_SERVERS)

			start = time.Now()
			for i := range NUM_SERVERS {
				go func(i int) {
					nsUpdate[i], qsUpdate[i], digestsUpdate[i], opsUpdate[i] = servers[i].Update(ops[i])
					wg.Done()
					// log.Println("Answer: thread", i, "did work")
				}(i)
			}
			wg.Wait()
			exp.RT["UpdateS"] += time.Since(start)
			fmt.Println("Finished UpdateS in ", exp.RT["UpdateS"], ". Start UpdateC.")

			// CLIENT UPDATE
			start = time.Now()
			_, _, _, _, err = client.UpdateHint(
				nsUpdate[0],
				nsUpdate[1],
				qsUpdate[0],
				qsUpdate[1],
				digestsUpdate[0],
				digestsUpdate[1],
				opsUpdate[0],
				opsUpdate[1],
			)
			if err != nil {
				log.Fatalln("Error in UpdateHint: ", err)
			}
			exp.RT["UpdateC"] += time.Since(start)
			fmt.Println("Finished UpdateC in ", exp.RT["UpdateC"], ".")

			// Store serizalized Bandwidth information for this experiment
			exp.StoreSerialized(
				[][]interface{}{
					[]interface{}{digests},
					[]interface{}{hintReqs},
					[]interface{}{hintResps},
					[]interface{}{opsUpdate},
				},
				[]string{
					"Digests",
					"HintReqs",
					"HintResps",
					"Updates",
				})

			// Get output string and write to file
			// for verbose output set last argument to true

			// If reconstruction fails, don't exit program,
			// instead run repetition again, but don't write results
			// if !abort {
			out := benchmark.GetOutputStringUpdates(exp, rep, *printToCMD)
			writer.Write(out)
			writer.Flush()
			// } else {
			// 	rep--
			// 	abort = false
			// }
			exp.ResetBenchVars()
		}
	}
}
