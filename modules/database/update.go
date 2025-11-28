package database

import (
	"bytes"
	"log"
)

type Update struct {
	Op  OpType
	Idx int
	Val []byte
}

// Enum for different PIR types
type OpType int

const (
	ADD OpType = iota
	EDIT
)

func (t OpType) String() string {
	return [...]string{
		"ADD",  // 0
		"EDIT", // 1
	}[t]
}

func (op1 *Update) Equals(op2 Update) bool {
	if op1.Op != op2.Op {
		log.Println("ops not equal type")
		return false
	}
	if op1.Idx != op2.Idx {
		log.Println("ops not equal indices")
		return false
	}
	if !bytes.Equal(op1.Val, op2.Val) {
		log.Println("ops not equal values")
		return false
	}
	return true
}
