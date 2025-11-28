/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/IBM/mathlib/driver"
	"github.com/IBM/mathlib/driver/amcl"
	"github.com/IBM/mathlib/driver/gurvy"
	"github.com/pkg/errors"
)

type CurveID int

const (
	FP256BN_AMCL CurveID = iota
	BN254
	FP256BN_AMCL_MIRACL
)

var Curves []*Curve = []*Curve{
	{
		C:          &amcl.Fp256bn{},
		GenG1:      &G1{G1: (&amcl.Fp256bn{}).GenG1(), CurveID: FP256BN_AMCL},
		GenG2:      &G2{G2: (&amcl.Fp256bn{}).GenG2(), CurveID: FP256BN_AMCL},
		GenGt:      &Gt{Gt: (&amcl.Fp256bn{}).GenGt(), CurveID: FP256BN_AMCL},
		GroupOrder: &Zr{Zr: (&amcl.Fp256bn{}).GroupOrder(), CurveID: FP256BN_AMCL},
		FieldBytes: (&amcl.Fp256bn{}).FieldBytes(),
		CurveID:    FP256BN_AMCL,
	},
	{
		C:          &gurvy.Bn254{},
		GenG1:      &G1{G1: (&gurvy.Bn254{}).GenG1(), CurveID: BN254},
		GenG2:      &G2{G2: (&gurvy.Bn254{}).GenG2(), CurveID: BN254},
		GenGt:      &Gt{Gt: (&gurvy.Bn254{}).GenGt(), CurveID: BN254},
		GroupOrder: &Zr{Zr: (&gurvy.Bn254{}).GroupOrder(), CurveID: BN254},
		FieldBytes: (&gurvy.Bn254{}).FieldBytes(),
		CurveID:    BN254,
	},
	{
		C:          &amcl.Fp256Miraclbn{},
		GenG1:      &G1{G1: (&amcl.Fp256Miraclbn{}).GenG1(), CurveID: FP256BN_AMCL_MIRACL},
		GenG2:      &G2{G2: (&amcl.Fp256Miraclbn{}).GenG2(), CurveID: FP256BN_AMCL_MIRACL},
		GenGt:      &Gt{Gt: (&amcl.Fp256Miraclbn{}).GenGt(), CurveID: FP256BN_AMCL_MIRACL},
		GroupOrder: &Zr{Zr: (&amcl.Fp256Miraclbn{}).GroupOrder(), CurveID: FP256BN_AMCL_MIRACL},
		FieldBytes: (&amcl.Fp256Miraclbn{}).FieldBytes(),
		CurveID:    FP256BN_AMCL_MIRACL,
	},
}

// Needed to export pp.go Params using gob
func init() {
	gob.Register(driver.G1(Curves[1].GenG1.G1))
	gob.Register(driver.G2(Curves[1].GenG2.G2))
	gob.Register(driver.Gt(Curves[1].GenGt.Gt))
}

/*********************************************************************/

type Zr struct {
	Zr      driver.Zr
	CurveID CurveID
}

func (z *Zr) Plus(a *Zr) *Zr {
	return &Zr{Zr: z.Zr.Plus(a.Zr), CurveID: z.CurveID}
}

func (z *Zr) Mul(a *Zr) *Zr {
	return &Zr{Zr: z.Zr.Mul(a.Zr), CurveID: z.CurveID}
}

func (z *Zr) Mod(a *Zr) {
	z.Zr.Mod(a.Zr)
}

func (z *Zr) PowMod(a *Zr) *Zr {
	return &Zr{Zr: z.Zr.PowMod(a.Zr), CurveID: z.CurveID}
}

func (z *Zr) InvModP(a *Zr) {
	z.Zr.InvModP(a.Zr)
}

func (z *Zr) Bytes() []byte {
	return z.Zr.Bytes()
}

func (z *Zr) Equals(a *Zr) bool {
	return z.Zr.Equals(a.Zr)
}

func (z *Zr) Copy() *Zr {
	return &Zr{Zr: z.Zr.Copy(), CurveID: z.CurveID}
}

func (z *Zr) Clone(a *Zr) {
	z.Zr.Clone(a.Zr)
}

func (z *Zr) String() string {
	return z.Zr.String()
}

var zerobytes = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var onebytes = []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}

func (z *Zr) Int() (int64, error) {
	b := z.Bytes()
	if !bytes.Equal(zerobytes, b[:32-8]) && !bytes.Equal(onebytes, b[:32-8]) {
		return 0, fmt.Errorf("out of range")
	}

	return int64(binary.BigEndian.Uint64(b[32-8:])), nil
}

/*********************************************************************/

type G1 struct {
	G1      driver.G1
	CurveID CurveID
}

func (g *G1) Clone(a *G1) {
	g.G1.Clone(a.G1)
}

func (g *G1) Copy() *G1 {
	return &G1{G1: g.G1.Copy(), CurveID: g.CurveID}
}

func (g *G1) Add(a *G1) {
	g.G1.Add(a.G1)
}

func (g *G1) Mul(a *Zr) *G1 {
	return &G1{G1: g.G1.Mul(a.Zr), CurveID: g.CurveID}
}

func (g *G1) Mul2(e *Zr, Q *G1, f *Zr) *G1 {
	return &G1{G1: g.G1.Mul2(e.Zr, Q.G1, f.Zr), CurveID: g.CurveID}
}

func (g *G1) Equals(a *G1) bool {
	return g.G1.Equals(a.G1)
}

func (g *G1) Bytes() []byte {
	return g.G1.Bytes()
}

func (g *G1) Sub(a *G1) {
	g.G1.Sub(a.G1)
}

func (g *G1) IsInfinity() bool {
	return g.G1.IsInfinity()
}

func (g *G1) String() string {
	return g.G1.String()
}

/*********************************************************************/

type G2 struct {
	G2      driver.G2
	CurveID CurveID
}

func (g *G2) Clone(a *G2) {
	g.G2.Clone(a.G2)
}

func (g *G2) Copy() *G2 {
	return &G2{G2: g.G2.Copy(), CurveID: g.CurveID}
}

func (g *G2) Mul(a *Zr) *G2 {
	return &G2{G2: g.G2.Mul(a.Zr), CurveID: g.CurveID}
}

func (g *G2) Add(a *G2) {
	g.G2.Add(a.G2)
}

func (g *G2) Sub(a *G2) {
	g.G2.Sub(a.G2)
}

func (g *G2) Affine() {
	g.G2.Affine()
}

func (g *G2) Bytes() []byte {
	return g.G2.Bytes()
}

func (g *G2) String() string {
	return g.G2.String()
}

func (g *G2) Equals(a *G2) bool {
	return g.G2.Equals(a.G2)
}

/*********************************************************************/

type Gt struct {
	Gt      driver.Gt
	CurveID CurveID
}

func (g *Gt) Equals(a *Gt) bool {
	return g.Gt.Equals(a.Gt)
}

func (g *Gt) Inverse() {
	g.Gt.Inverse()
}

func (g *Gt) Mul(a *Gt) {
	g.Gt.Mul(a.Gt)
}

func (g *Gt) Exp(z *Zr) *Gt {
	return &Gt{Gt: g.Gt.Exp(z.Zr), CurveID: g.CurveID}
}

func (g *Gt) IsUnity() bool {
	return g.Gt.IsUnity()
}

func (g *Gt) String() string {
	return g.Gt.ToString()
}

func (g *Gt) Bytes() []byte {
	return g.Gt.Bytes()
}

/*********************************************************************/

type Curve struct {
	C          driver.Curve
	GenG1      *G1
	GenG2      *G2
	GenGt      *Gt
	GroupOrder *Zr
	FieldBytes int
	CurveID    CurveID
}

func (c *Curve) Rand() (io.Reader, error) {
	return c.C.Rand()
}

func (c *Curve) NewRandomZr(rng io.Reader) *Zr {
	return &Zr{Zr: c.C.NewRandomZr(rng), CurveID: c.CurveID}
}

func (c *Curve) NewZrFromBytes(b []byte) *Zr {
	return &Zr{Zr: c.C.NewZrFromBytes(b), CurveID: c.CurveID}
}

func (c *Curve) NewG1FromBytes(b []byte) (p *G1, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("failure [%s]", r)
			p = nil
		}
	}()

	p = &G1{G1: c.C.NewG1FromBytes(b), CurveID: c.CurveID}
	return
}

func (c *Curve) NewG2FromBytes(b []byte) (p *G2, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("failure [%s]", r)
			p = nil
		}
	}()

	p = &G2{G2: c.C.NewG2FromBytes(b), CurveID: c.CurveID}
	return
}

func (c *Curve) NewGtFromBytes(b []byte) (p *Gt, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("failure [%s]", r)
			p = nil
		}
	}()

	p = &Gt{Gt: c.C.NewGtFromBytes(b), CurveID: c.CurveID}
	return
}

func (c *Curve) NewZrFromInt(i int64) *Zr {
	return &Zr{Zr: c.C.NewZrFromInt(i), CurveID: c.CurveID}
}

// func (c *Curve) NewG1FromCoords(ix, iy *Zr) *G1 {
// 	return &G1{c.c.NewG1FromCoords(ix.zr, iy.zr)}
// }

func (c *Curve) NewG2() *G2 {
	return &G2{G2: c.C.NewG2(), CurveID: c.CurveID}
}

func (c *Curve) NewG1() *G1 {
	return &G1{G1: c.C.NewG1(), CurveID: c.CurveID}
}

func (c *Curve) Pairing(a *G2, b *G1) *Gt {
	return &Gt{Gt: c.C.Pairing(a.G2, b.G1), CurveID: c.CurveID}
}

func (c *Curve) Pairing2(p *G2, q *G1, r *G2, s *G1) *Gt {
	return &Gt{Gt: c.C.Pairing2(p.G2, r.G2, q.G1, s.G1), CurveID: c.CurveID}
}

func (c *Curve) FExp(a *Gt) *Gt {
	return &Gt{Gt: c.C.FExp(a.Gt), CurveID: c.CurveID}
}

func (c *Curve) HashToZr(data []byte) *Zr {
	return &Zr{Zr: c.C.HashToZr(data), CurveID: c.CurveID}
}

func (c *Curve) HashToG1(data []byte) *G1 {
	return &G1{G1: c.C.HashToG1(data), CurveID: c.CurveID}
}

func (c *Curve) ModSub(a, b, m *Zr) *Zr {
	return &Zr{Zr: c.C.ModSub(a.Zr, b.Zr, m.Zr), CurveID: c.CurveID}
}

func (c *Curve) ModAdd(a, b, m *Zr) *Zr {
	return &Zr{Zr: c.C.ModAdd(a.Zr, b.Zr, m.Zr), CurveID: c.CurveID}
}

func (c *Curve) ModMul(a1, b1, m *Zr) *Zr {
	return &Zr{Zr: c.C.ModMul(a1.Zr, b1.Zr, m.Zr), CurveID: c.CurveID}
}

func (c *Curve) ModNeg(a1, m *Zr) *Zr {
	return &Zr{Zr: c.C.ModNeg(a1.Zr, m.Zr), CurveID: c.CurveID}
}

/*********************************************************************/
