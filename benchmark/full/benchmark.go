package main

import (
	"bytes"
	"encoding/csv"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"tapir/benchmark"
	"tapir/modules/database"
	"tapir/modules/utils"
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

	if *pathPreprocessing != "" {
		// Ensure folder path exists
		err := os.MkdirAll(*pathPreprocessing, os.ModePerm)
		if err != nil {
			log.Fatalf("error creating directory %s: %v", *pathPreprocessing, err)
		}
	}

	// abort := false

	// Open file for writing
	file, err := os.Create(*pathWrite)
	if err != nil {
		log.Fatal("error creating file", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// write column headers
	writer.Write(benchmark.Headers)

	for _, config := range configs.Configs {

		// pick random index for each new experiment
		b := utils.NewBufPRG(utils.NewPRG(&utils.PRGKey{0}))

		// Create a new database of random records
		db := database.MakeRandomDB(seed, config.DbSize, config.RecSize)

		exp := benchmark.NewExperiment(&config)

		///////////////////////////////////////////////////////////////////
		// INITIALIZE CLIENT & SERVERS ////////////////////////////////////

		// Create a new APIR server
		servers := make([]pir.APIRServer, NUM_SERVERS)
		for i := range NUM_SERVERS {
			servers[i] = pir.NewServer(pir.PirType(exp.PirType), db, i, exp.NumParts, vc.VcType(exp.VcType))
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
		digest, hint, err := client.VerSetup(digests[0], digests[1], hintResps[0], hintResps[1])
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

			idx := b.RandInt(config.DbSize)

			// Generate queries for record idx
			start = time.Now()
			query0, query1, _ := client.Query(idx)
			queries := []pir.Query{query0, query1}
			exp.RT["Query"] += time.Since(start)
			fmt.Println("Finished Query in ", exp.RT["Query"], ". Start Answer.")

			// Answer the queries
			wg.Add(NUM_SERVERS)
			answers := make([]pir.Answer, NUM_SERVERS)

			start = time.Now()
			for i := range NUM_SERVERS {
				go func(i int) {
					answers[i], err = servers[i].Answer(queries[i])
					if err != nil {
						log.Fatalln("Error in Answer: ", err)
					}
					wg.Done()
					// log.Println("Answer: thread", i, "did work")
				}(i)
			}
			wg.Wait()
			exp.RT["Answer"] += time.Since(start)
			fmt.Println("Finished Answer in ", exp.RT["Answer"], ". Start Answer.")

			// Reconstruct the record
			start = time.Now()
			record, _ := client.Reconstruct(digest, hint, answers[0], answers[1])
			exp.RT["Reconstruct"] += time.Since(start)

			fmt.Println("Finished Reconstruct in ", exp.RT["Reconstruct"])

			// Store serizalized Bandwidth information for this experiment
			exp.StoreSerialized(
				[][]interface{}{
					[]interface{}{digests},
					[]interface{}{hintReqs},
					[]interface{}{hintResps},
					[]interface{}{queries},
					[]interface{}{answers},
				},
				[]string{
					"Digests",
					"HintReqs",
					"HintResps",
					"Queries",
					"Answers",
				})

			if !bytes.Equal(record, db.GetRecord(idx)) {
				log.Println("Reconstructing Record", idx, "failed.")
				// log.Fatalln("Reconstructing Record", idx, "failed.\nDB Record:\t",
				// db.GetRecord(idx), "\nReconstructed:\t", record)
			}

			// Get output string and write to file
			// for verbose output set last argument to true
			out := benchmark.GetOutputString(exp, rep, *printToCMD)
			writer.Write(out)
			writer.Flush()

			exp.ResetOnlineRTVars()
		}
	}
}
