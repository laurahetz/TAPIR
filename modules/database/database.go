package database

import (
	"bufio"
	"bytes"
	"errors"
	"log"
	"math/rand/v2"
	"os"

	"tapir/modules/psetggm"

	"github.com/lukechampine/fastxor"
)

type DB struct {
	N        int
	RecSize  int
	Capacity int

	Data []byte
}

// One database record
type Record []byte

func (a Record) Equals(b Record) bool {
	return len(a) == len(b) && bytes.Equal(a, b)
}

func (db *DB) Equals(db2 *DB) (bool, error) {
	if !bytes.Equal(db.Data, db2.Data) {
		return false, errors.New("data not equal")
	}
	if db.RecSize != db2.RecSize {
		return false, errors.New("record size not equal")
	}
	if db.N != db2.N {
		return false, errors.New("N not equal")
	}
	if db.Capacity != db2.Capacity {
		return false, errors.New("capacity not equal")
	}
	return true, nil
}

func PadRecord(r Record, size int) Record {
	if len(r) > size {
		panic("Record is too big to pad to desired size")
	}
	if len(r) == size {
		return r
	}
	padded := make(Record, size)
	copy(padded[size-len(r):], r)
	return padded
}

func (db *DB) Slice(start, end int) []byte {
	return db.Data[start*db.RecSize : end*db.RecSize]
}

func (db *DB) GetRecord(i int) Record {
	if i >= db.Capacity {
		return nil
	}

	rec := make([]byte, db.RecSize)
	copy(rec, db.Data[i*db.RecSize:(i+1)*db.RecSize])
	return rec
}

func (db *DB) SetRecord(i int, val []byte) error {
	if i >= db.Capacity {
		return errors.New("index out of range")
	}
	copy(db.Data[i*db.RecSize:(i+1)*db.RecSize], val[:])
	return nil
}

func (db *DB) ExtendCapacity(ext int) {
	// Copy existing data
	newData := make([]byte, (db.N+ext)*db.RecSize)
	copy(newData, db.Data)

	// Update database
	db.Capacity = db.N + ext
	db.Data = newData
}

// do not use this for partitioned databases!
func (db *DB) Update(ops []Update) {

	var additions []Update

	for _, op := range ops {
		if op.Op == ADD {
			additions = append(additions, op)
		} else if op.Op == EDIT {
			db.SetRecord(op.Idx, op.Val)
		}
	}
	if len(additions) > 0 {
		db.ExtendCapacity(len(additions))
		for _, op := range additions {
			db.SetRecord(db.N, op.Val)
			db.N++
		}
	}
}

// func (db *DB) UpdateWithPartitions(ops []Update, partSize int) { //(updatedPartitions map[int]struct{}) {

// 	var additions []Update
// 	for _, op := range ops {

// 		// using this map as a set to track the changed partitions
// 		// updatedPartitions[op.Idx/partSize] = struct{}{}

// 		if op.Op == ADD {
// 			additions = append(additions, op)
// 		} else if op.Op == EDIT {
// 			db.SetRecord(op.Idx, op.Val)
// 		}
// 	}
// 	if len(additions) > 0 {
// 		Nnew := db.N + len(additions)
// 		// new number of db partitions
// 		numPart := int(math.Ceil(float64(Nnew) / float64(partSize)))

// 		db.ExtendCapacity(numPart * partSize)
// 		for _, op := range additions {
// 			db.SetRecord(db.N, op.Val)
// 			db.N++
// 		}
// 		// // fill
// 		// for i := range numPart * partSize -1 - db.N, i < numPart*partSize, i++ {
// 		// 	db.SetRecord(db.N, 0)
// 		// 	db.N++
// 		// }
// 	}
// }

// Return consecutive records in (start, start+end)
func (db *DB) GetRecords(start, num int) []Record {

	if start >= db.Capacity {
		return nil
	}
	recs := make([]Record, num)
	for i := 0; i < num; i++ {
		recs[i] = db.GetRecord(start + i)
	}
	return recs
}

func DBFromRecords(records []Record) *DB {
	if len(records) < 1 {
		return &DB{0, 0, 0, nil}
	}

	// if len(records[0]) != 16 {
	// 	panic("Number DB only supports 16-byte records")
	// }

	// Make new DB struct with the correct dimensions for N and RecSize
	db := &DB{N: len(records), RecSize: len(records[0]), Capacity: len(records), Data: nil}
	data := make([]byte, db.RecSize*db.N)

	for i, v := range records {
		if len(v) != db.RecSize {
			log.Printf("Got row[%v] %v %v\n", i, len(v), db.RecSize)
			panic("Database rows must all be of the same length")
		}
		copy(data[i*db.RecSize:], v[:])
	}
	db.Data = data
	return db
}

func MakeNumberRows(n, recSize int) []Record {
	// This was adapted from the SinglePass code
	db := make([]Record, n)
	for i := range db {
		db[i] = make([]byte, recSize)
		for j := range db[i] {
			db[i][j] = byte(i)
		}
	}
	return db
}

func MakeNumberDB(n int, recSize int) *DB {
	return DBFromRecords(MakeNumberRows(n, recSize))
}

func MakeRandomRows(prg *rand.ChaCha8, n, recSize int) []Record {
	// This was adapted from the SinglePass code
	db := make([]Record, n)
	for i := range db {
		db[i] = make([]byte, recSize)
		_, err := prg.Read(db[i])
		if err != nil {
			panic(err)
		}
	}
	return db
}

func MakeRandomUpdates(prg *rand.ChaCha8, dbsize, numUpdates, recSize int, types []OpType) []Update {
	// This was adapted from the SinglePass code
	ops := make([]Update, numUpdates)
	idxMap := make(map[int]bool)
	for i := range ops {
		// Get random operation type from types slice
		ops[i].Op = types[int(prg.Uint64()%uint64(len(types)))]
		ops[i].Val = make([]byte, recSize)
		_, err := prg.Read(ops[i].Val)
		if err != nil {
			panic(err)
		}
		if ops[i].Op == ADD {
			ops[i].Idx = -1
		} else {
			// Get random unique int from 0 to dbsize-1
			for {
				idx := int(prg.Uint64() % uint64(dbsize))
				if !idxMap[idx] {
					idxMap[idx] = true
					ops[i].Idx = idx
					break
				}
			}
		}

	}
	return ops
}

func MakeUpdatesFixedValues(prg *rand.ChaCha8, dbsize, numUpdates, recSize int, types []OpType) []Update {
	ops := make([]Update, numUpdates)
	idxMap := make(map[int]bool)
	for i := range ops {
		// Get random operation type from types slice
		ops[i].Op = types[int(prg.Uint64()%uint64(len(types)))]

		ops[i].Val = make([]byte, recSize)
		// for j := range recSize {
		ops[i].Val[0] = byte(1)

		if ops[i].Op == ADD {
			ops[i].Idx = -1
		} else {
			// Get random unique int from 0 to dbsize-1
			for {
				idx := int(prg.Uint64() % uint64(dbsize))
				if !idxMap[idx] {
					idxMap[idx] = true
					ops[i].Idx = idx
					break
				}
			}
		}
	}
	return ops
}

func MakeRandomDB(seed [32]byte, n int, recSize int) *DB {
	prg := rand.NewChaCha8(seed)
	return DBFromRecords(MakeRandomRows(prg, n, recSize))
}

func XorInto(a []byte, b []byte) {
	if len(a) != len(b) {
		panic("Tried to XOR byte-slices of unequal length.")
	}

	fastxor.Bytes(a, a, b)
}

func (db *DB) VectorProd(bitVector []byte) []byte {
	out := make(Record, db.RecSize)
	if db.RecSize == 32 {
		psetggm.XorHashesByBitVector(db.Data, bitVector, out)
	} else {
		var j uint
		for j = 0; j < uint(db.N); j++ {
			if ((1 << (j % 8)) & bitVector[j/8]) != 0 {
				XorInto(out, db.Data[j*uint(db.RecSize):(j+1)*uint(db.RecSize)])
			}
		}
	}
	return out
}

func (db *DB) WriteToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	for i := 0; i < db.N; i++ {
		w.Write(db.Slice(i, i+1)) // this is stupid why don't we do it in larger chunks??
	}
	return nil
}

// TODO extend to variable record size
func ReadFromFile(filename string, N int) (*DB, error) {
	log.Println("ReadFromFile does not support record sizes other than 16B")
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	db := &DB{N: N, RecSize: 16, Data: nil}
	var rec Record
	for {
		rec = make([]byte, 16)
		_, err := r.Read(rec)
		if err != nil {
			break
		}
		db.Data = append(db.Data, rec...)
	}
	if err != nil && err.Error() != "EOF" {
		log.Fatal(err)
		return nil, err
	}
	return db, nil
}
