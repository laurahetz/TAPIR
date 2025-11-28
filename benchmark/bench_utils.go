package benchmark

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"tapir/modules/database"
	"tapir/modules/vc"
	"tapir/pir"
	"time"
)

// Experiment Config
type Config struct {
	PirType     int
	Repetitions uint32
	DbSize      int
	NumParts    int
	RecSize     int
	VcType      int
	NumUpdates  int
	UpdateTypes int // 0 = ADD, 1 = EDIT, 2 = BOTH
}

// Experiment Suite
// used to read experiment configs from file
type DriverConfig struct {
	Configs []Config
}

func GetUpdateTypesFromConfig(updateTypes int) []database.OpType {
	if updateTypes == 2 {
		return []database.OpType{database.ADD, database.EDIT}
	}
	return []database.OpType{database.OpType(updateTypes)}
}

// read in JSON file, that contains multiple configs
// each config is one benchmark test for which a new client is generated
func ReadBenchConfigs(path string) *DriverConfig {

	content, err := os.ReadFile(path)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}
	var rConfig DriverConfig
	err = json.Unmarshal(content, &rConfig)

	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}
	return &rConfig

}

type Experiment struct {
	*Config
	BW map[string]uint32
	RT map[string]time.Duration
}

func NewExperiment(config *Config) *Experiment {
	exp := Experiment{}
	exp.Config = config
	exp.ResetBenchVars()

	return &exp
}

func (exp *Experiment) ResetBenchVars() {
	exp.BW = map[string]uint32{
		"Digests":   0,
		"HintReqs":  0,
		"HintResps": 0,
		"Queries":   0,
		"Answers":   0,
		"Updates":   0,
	}
	exp.RT = map[string]time.Duration{
		"GenDigest":   0,
		"RequestHint": 0,
		"GenHint":     0,
		"VerSetup":    0,
		"Query":       0,
		"Answer":      0,
		"Reconstruct": 0,
		"UpdateS":     0,
		"UpdateC":     0,
	}
}

func (exp *Experiment) ResetOnlineRTVars() {
	exp.BW = map[string]uint32{
		"Digests":   0,
		"HintReqs":  0,
		"HintResps": 0,
		"Queries":   0,
		"Answers":   0,
		"Updates":   0,
	}
	exp.RT = map[string]time.Duration{
		"GenDigest":   exp.RT["GenDigest"],
		"RequestHint": exp.RT["RequestHint"],
		"GenHint":     exp.RT["GenHint"],
		"VerSetup":    exp.RT["VerSetup"],
		"Query":       0,
		"Answer":      0,
		"Reconstruct": 0,
		"UpdateS":     0,
		"UpdateC":     0,
	}
}

func (exp *Experiment) StoreSerialized(in [][]interface{}, names []string) {
	if len(in) != len(names) {
		log.Fatalf("Length of input and names must be the same")
	}
	for i, name := range names {
		out, err := SerializedSizeList(in[i])
		if err != nil {
			log.Fatalf("Error in calculating size of %s: %v", name, err)
		}
		exp.BW[name] += uint32(out)
	}
}

// Input Config, DBParams and number of repetition
func GetOutputString(exp *Experiment, i int, cmdPrint bool) []string {
	var out []string

	for _, key := range Headers {
		if key == "pir_type" {
			pirT := pir.PirType(exp.PirType)
			if cmdPrint {
				log.Println("\ndb_type:", key, ":", pirT.String())
			}
			out = append(out, pirT.String())
		} else if key == "vc_type" {
			if cmdPrint {
				log.Println("\nvc_type:", key, ":", vc.VcType(exp.VcType).String())
			}
			out = append(out, vc.VcType(exp.VcType).String())
		} else if key == "db_size" {
			if cmdPrint {
				log.Println("\ndb_size:", key, ":", exp.DbSize)
			}
			out = append(out, strconv.Itoa(int(exp.DbSize)))
		} else if key == "part_size" {
			if cmdPrint {
				log.Println("\npart_size:", key, ":", exp.NumParts)
			}
			out = append(out, strconv.Itoa(int(exp.NumParts)))
		} else if key == "rec_size" {
			if cmdPrint {
				log.Println("rec_size:", strconv.Itoa(exp.RecSize))
			}
			out = append(out, strconv.Itoa(exp.RecSize))
		} else if key == "repetition" {
			if cmdPrint {
				log.Println("rep:", strconv.Itoa(i))
			}
			out = append(out, strconv.Itoa(i))
		} else if key[:3] == "BW_" {
			if cmdPrint {
				log.Println("BW:", key, ":", strconv.Itoa(int(exp.BW[key[3:]])))
			}
			out = append(out, strconv.Itoa(int(exp.BW[key[3:]])))
		} else if key[:3] == "RT_" {
			if cmdPrint {
				log.Println("RT:", key, ":", strconv.Itoa(int(exp.RT[key[3:]].Microseconds())))
			}
			out = append(out, strconv.Itoa(int(exp.RT[key[3:]].Microseconds())))
		}
	}
	return out
}

// Input Config, DBParams and number of repetition
func GetOutputStringUpdates(exp *Experiment, i int, cmdPrint bool) []string {
	var out []string

	for _, key := range HeadersUpdate {
		if key == "pir_type" {
			pirT := pir.PirType(exp.PirType)
			if cmdPrint {
				log.Println("\ndb_type:", key, ":", pirT.String())
			}
			out = append(out, pirT.String())
		} else if key == "vc_type" {
			if cmdPrint {
				log.Println("\nvc_type:", key, ":", vc.VcType(exp.VcType).String())
			}
			out = append(out, vc.VcType(exp.VcType).String())
		} else if key == "db_size" {
			if cmdPrint {
				log.Println("\ndb_size:", key, ":", exp.DbSize)
			}
			out = append(out, strconv.Itoa(int(exp.DbSize)))
		} else if key == "part_size" {
			if cmdPrint {
				log.Println("\npart_size:", key, ":", exp.NumParts)
			}
			out = append(out, strconv.Itoa(int(exp.NumParts)))
		} else if key == "rec_size" {
			if cmdPrint {
				log.Println("rec_size:", strconv.Itoa(exp.RecSize))
			}
			out = append(out, strconv.Itoa(exp.RecSize))
		} else if key == "num_updates" {
			if cmdPrint {
				log.Println("num_updates:", strconv.Itoa(exp.NumUpdates))
			}
			out = append(out, strconv.Itoa(exp.NumUpdates))
		} else if key == "update_type" {
			if cmdPrint {
				log.Println("update_type:", strconv.Itoa(exp.UpdateTypes))
			}
			out = append(out, strconv.Itoa(exp.UpdateTypes))
		} else if key == "repetition" {
			if cmdPrint {
				log.Println("rep:", strconv.Itoa(i))
			}
			out = append(out, strconv.Itoa(i))
		} else if key[:3] == "BW_" {
			if cmdPrint {
				log.Println("BW:", key, ":", strconv.Itoa(int(exp.BW[key[3:]])))
			}
			out = append(out, strconv.Itoa(int(exp.BW[key[3:]])))
		} else if key[:3] == "RT_" {
			if cmdPrint {
				log.Println("RT:", key, ":", strconv.Itoa(int(exp.RT[key[3:]].Microseconds())))
			}
			out = append(out, strconv.Itoa(int(exp.RT[key[3:]].Microseconds())))
		}
	}
	return out
}

// define column headers
var Headers = []string{
	"pir_type",
	"vc_type",
	"db_size",   // N
	"part_size", // Q
	"rec_size",
	"repetition",
	"BW_Digests",
	"BW_HintReqs",
	"BW_HintResps",
	"BW_Queries",
	"BW_Answers",
	"RT_GenDigest",
	"RT_RequestHint",
	"RT_GenHint",
	"RT_VerSetup",
	"RT_Query",
	"RT_Answer",
	"RT_Reconstruct",
}

var HeadersUpdate = []string{
	"pir_type",
	"vc_type",
	"db_size",   // N
	"part_size", // Q
	"rec_size",
	"num_updates",
	"update_type",
	"repetition",
	"BW_Digests",
	"BW_HintReqs",
	"BW_HintResps",
	"BW_Queries",
	"BW_Answers",
	"BW_Updates",
	"RT_GenDigest",
	"RT_RequestHint",
	"RT_GenHint",
	"RT_VerSetup",
	"RT_Query",
	"RT_Answer",
	"RT_Reconstruct",
	"RT_UpdateS",
	"RT_UpdateC",
}
