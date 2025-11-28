package database

import (
	"bytes"
	"fmt"
	"log"
)

func ExampleDatabase() {
	// Create a new database with 3 records of 4 bytes each
	db := DBFromRecords([]Record{
		[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		[]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		[]byte{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
	})
	// Get the second record
	record := db.GetRecord(1)
	fmt.Println(record)

	// Output: [1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1]
}

func ExampleDatabaseSlice() {
	// Create a new database with 3 records of 4 bytes each
	db := DBFromRecords([]Record{
		[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		[]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		[]byte{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
	})
	// Get the first two records
	slice := db.Slice(0, 2)
	fmt.Println(slice)

	// Output: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1]
}

func ExampleXorInto() {
	// Create two byte slices
	a := []byte{1, 0, 1, 1}
	b := []byte{0, 1, 1, 1}
	// XOR the two slices
	XorInto(a, b)
	fmt.Println(a)

	// Output: [1 1 0 0]
}

func ExampleDBVectorProd() {
	// Create a new database with 3 records of 4 bytes each
	db := DBFromRecords([]Record{
		[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		[]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		[]byte{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
	})

	// Create a bit vector that includes the 0th and 2nd records
	// Each bit in the bit vector corresponds to a record in the database
	// Here, we do the bitvector 101
	// The output should be {0, 0, 0, 0 ... } XOR {2, 2, 2, 2 ...} = {2, 2, 2, 2...}
	bitVector := []byte{0b101}
	// Compute the vector product
	result := db.VectorProd(bitVector)
	fmt.Println(result)

	// The database values should not change
	fmt.Println(db.Slice(0, 3))

	// Output:
	// [2 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2]
	// [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2]
}

func ExampleDatabaseWrite() {
	n := 3
	// Create a new database with 3 records of 4 bytes each
	db := DBFromRecords([]Record{
		[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		[]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		[]byte{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2},
	})
	// Write the database to a file
	err := db.WriteToFile("test.db")
	if err != nil {
		log.Println("Error writing to file:", err)
		return
	}

	// Read the database from the file
	db2, err := ReadFromFile("test.db", n)
	if err != nil {
		log.Println("Error reading from file:", err)
		return
	}
	// Check if the two databases are equal
	if !bytes.Equal(db.Data, db2.Data) {
		fmt.Println("Databases are not equal")
	} else {
		fmt.Println("Databases are equal")
	}

	record := db.GetRecord(1)
	fmt.Println(record)

	// Output:
	// Databases are equal
	// [1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1]

}
