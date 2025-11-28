package benchmark

import (
	"bytes"
	"log"
	"math/big"
	"os"
	"reflect"
	"runtime"
	"runtime/pprof"

	"tapir/modules/database"
	"tapir/modules/merkle"
	"tapir/modules/pp"
	"tapir/modules/vc"
	"tapir/pir"

	math "github.com/IBM/mathlib"
	"github.com/ugorji/go/codec"
)

///////////////////////////////////////////////////////////////////
// MEMORY
///////////////////////////////////////////////////////////////////

type Profiler struct {
	f        *os.File
	filename string
}

func NewProfiler(filename string) *Profiler {
	prof := new(Profiler)
	prof.filename = filename
	if filename != "" {
		var err error
		prof.f, err = os.Create(filename)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(prof.f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}
	return prof
}

func (p *Profiler) Close() {
	if p.f == nil {
		return
	}
	pprof.StopCPUProfile()
	p.f.Close()

	runtime.GC()
	if memProf, err := os.Create(p.filename + "-mem.prof"); err != nil {
		panic(err)
	} else {
		pprof.WriteHeapProfile(memProf)
		memProf.Close()
	}
}

///////////////////////////////////////////////////////////////////
// BANDWIDTH
///////////////////////////////////////////////////////////////////

var registeredObjs = []interface{}{

	database.Record{},
	database.Update{},
	vc.MerkleParams{},
	vc.MerkleCommitment{},
	vc.MerkleProof{},
	vc.PPParams{},
	vc.PPCommitment{},
	vc.PointProof{},
	vc.PPAggProof{},
	big.Int{},
	merkle.Proof{},
	math.G1{},
	pp.PP{},

	pir.DPF128Digest{},
	pir.DPF128HintQuery{},
	pir.DPF128HintResp{},
	pir.DPF128Hint{},
	pir.DPF128Query{},
	pir.DPF128Answer{},

	pir.TAPIRDigest{},
	pir.TAPIRHintQuery{},
	pir.TAPIRHintResp{},
	pir.TAPIRHint{},
	pir.TAPIRQuery{},
	pir.TAPIRAnswer{},

	pir.DPFDigest{},
	pir.DPFHintQuery{},
	pir.DPFHintResp{},
	pir.DPFHint{},
	pir.DPFQuery{},
	pir.DPFAnswer{},

	pir.MatrixDigest{},
	pir.MatrixHintQuery{},
	pir.MatrixHintResp{},
	pir.MatrixHint{},
	pir.MatrixQuery{},
	pir.MatrixAnswer{},

	pir.APIR_MatrixAnswer{},
	pir.APIR_MatrixDigest{},
	pir.APIR_MatrixHintQuery{},
	pir.APIR_MatrixHintResp{},
	pir.APIR_MatrixHint{},
	pir.APIR_MatrixQuery{},

	pir.SinglePassDigest{},
	pir.SinglePassHintQuery{},
	pir.SinglePassHintResp{},
	pir.SinglePassHint{},
	pir.SinglePassQuery{},
	pir.SinglePassAnswer{},
}

func registeredTypes() []reflect.Type {
	types := make([]reflect.Type, 0, len(registeredObjs))
	for _, obj := range registeredObjs {
		types = append(types, reflect.TypeOf(obj))
	}
	return types
}

func codecHandle(types []reflect.Type) codec.Handle {
	h := codec.BincHandle{}
	h.StructToArray = true
	h.OptimumSize = true
	h.PreferPointerForStructOrArray = false

	for i, t := range types {
		err := h.SetBytesExt(t, uint64(0x10+i), codec.SelfExt)
		if err != nil {
			panic(err)
		}
	}

	return &h
}

func SerializedSize(e interface{}) (int, error) {
	var buf bytes.Buffer
	enc := codec.NewEncoder(&buf, codecHandle((registeredTypes())))
	err := enc.Encode(e)
	if err != nil {
		panic(err)
	}
	return buf.Len(), nil
}

func SerializedSizeList(e []interface{}) (int, error) {
	var total int
	for _, i := range e {
		if size, err := SerializedSize(i); err != nil {
			return -1, err
		} else {
			total += size
		}
	}
	return total, nil
}
