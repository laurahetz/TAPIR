package pp

import (
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"tapir/modules/database"
)

// Code from https://github.com/yacovm/PoL

type PP struct {
	Digest []byte
	N      int
	G1s    G1v
	G2s    G2v
	Gt     *Gt
}

func NewPublicParams(N int) *PP {

	// Servers need to use the same randomness
	// TODO use a seed to generate the randomness
	// α := c.NewRandomZr(rand.Reader)
	α := c.NewZrFromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})

	pp := &PP{N: N}

	g1 := c.GenG1.Copy()
	g2 := c.GenG2.Copy()
	gob.Register(G1(*g1))

	g1α := func(i int) *G1 {
		return g1.Mul(α.PowMod(c.NewZrFromInt(int64(i))))
	}

	g2α := func(i int) *G2 {
		return g2.Mul(α.PowMod(c.NewZrFromInt(int64(i))))
	}

	for i := 1; i <= N; i++ {
		pp.G1s = append(pp.G1s, g1α(i))
	}

	// Artificially put the generator instead of g^{a^{N+1}}
	pp.G1s = append(pp.G1s, c.GenG1.Copy())

	for i := N + 2; i <= 2*N; i++ {
		pp.G1s = append(pp.G1s, g1α(i))
	}
	gob.Register(G1v(pp.G1s))

	for i := 1; i <= N; i++ {
		pp.G2s = append(pp.G2s, g2α(i))
	}
	gob.Register(G2v(pp.G2s))

	pp.Gt = c.GenGt.Exp(α.PowMod(c.NewZrFromInt(int64(N + 1))))
	gob.Register(Gt(*pp.Gt))

	pp.SetupDigest()

	return pp
}

func (pp *PP) Size() int {
	return len(pp.G1s.Bytes()) + len(pp.G2s.Bytes()) + len(pp.Gt.Bytes())
}

func (pp *PP) SetupDigest() {
	h := sha256.New()
	for i := 0; i < len(pp.G1s); i++ {
		h.Write(pp.G1s[i].Bytes())
	}
	for i := 0; i < len(pp.G2s); i++ {
		h.Write(pp.G2s[i].Bytes())
	}
	h.Write(pp.Gt.Bytes())
	pp.Digest = h.Sum(nil)
}

func Commit(pp *PP, m Vec) *G1 {
	if len(m) != pp.N {
		panic(fmt.Sprintf("message should be of size %d but is of size %d", pp.N, len(m)))
	}

	var powersOfAlpha G1v
	for i := 0; i < pp.N; i++ {
		powersOfAlpha = append(powersOfAlpha, pp.G1s[i])
	}

	return powersOfAlpha.MulV(m).Sum()
}

func Open(pp *PP, i int, m Vec) (mi *Zr, π *G1) {
	if i >= pp.N {
		panic(fmt.Sprintf("can only open an index in [0,%d]", pp.N-1))
	}

	shift := pp.N - i

	var elements G1v
	var exponents Vec
	for j := 1; j <= pp.N; j++ {
		if j == i+1 {
			continue
		}
		index := shift + j - 1
		elements = append(elements, pp.G1s[index])
		exponents = append(exponents, m[j-1])
	}

	π = elements.MulV(exponents).Sum()
	mi = m[i]

	return
}

func Verify(pp *PP, mi *Zr, π *G1, C *G1, i int) error {
	left := G1v{C}.InnerProd(G2v{pp.G2s[pp.N-i-1]})
	right := G1v{π}.InnerProd(G2v{c.GenG2})
	right.Mul(pp.Gt.Exp(mi))

	if left.Equals(right) {
		return nil
	}
	return fmt.Errorf("%v is not an element in index %d in %v", mi, i, C)
}

func Update(pp *PP, C *G1, m Vec, mi *Zr, i int) {
	prevG := pp.G1s[i].Mul(m[i])
	nextG := pp.G1s[i].Mul(mi)
	C.Sub(prevG)
	C.Add(nextG)

}

func Aggregate(pp *PP, commitments G1v, proofs []*G1, RO func(*PP, []*G1, int) *Zr) *G1 {
	if len(proofs) != len(commitments) {
		panic(fmt.Sprintf("cannot aggregate %d proofs corresponding to %d commitments", len(proofs), len(commitments)))
	}
	var π G1v

	for j := 0; j < len(proofs); j++ {
		π = append(π, proofs[j].Mul(RO(pp, commitments, j)))
	}

	return π.Sum()
}

func VerifyAggregation(pp *PP, indices []int, commitments G1v, π *G1, Σ *Zr, RO func(*PP, []*G1, int) *Zr) error {
	var exponents []*Zr
	for i := 0; i < len(indices); i++ {
		exponents = append(exponents, RO(pp, commitments, i))
	}

	var g2s G2v
	for _, i := range indices {
		g2s = append(g2s, pp.G2s[pp.N-i-1])
	}
	left := commitments.InnerProd(g2s.Mulv(exponents))

	πg2 := G1v{π}.InnerProd(G2v{c.GenG2})
	right := pp.Gt.Exp(Σ)
	right.Mul(πg2)

	if right.Equals(left) {
		return nil
	}

	return fmt.Errorf("invalid aggregation")
}

func VerifyAggregationRecords(pp *PP, indices []int, commitments G1v, π *G1, elems []database.Record, RO func(*PP, []*G1, int) *Zr) error {
	var exponents []*Zr
	var es Vec
	for i := 0; i < len(indices); i++ {
		exponents = append(exponents, RO(pp, commitments, i))
		es = append(es, FieldElementFromBytes(elems[i]))
	}

	var g2s G2v
	for _, i := range indices {
		g2s = append(g2s, pp.G2s[pp.N-i-1])
	}
	left := commitments.InnerProd(g2s.Mulv(exponents))
	Σ := es.InnerProd(exponents)
	πg2 := G1v{π}.InnerProd(G2v{c.GenG2})
	right := pp.Gt.Exp(Σ)
	right.Mul(πg2)

	if right.Equals(left) {
		return nil
	}

	return fmt.Errorf("invalid aggregation")
}

func RO(pp *PP, cs []*G1, i int) *Zr {
	h := sha256.New()
	h.Write(pp.Digest)
	h.Write([]byte{byte(i)})
	for j := 0; j < len(cs); j++ {
		h.Write(cs[j].Bytes())
	}
	digest := h.Sum(nil)
	result := FieldElementFromBytes(digest)
	return result
}

func G1FromBytes(b []byte) (*G1, error) {
	return c.NewG1FromBytes(b)
}
