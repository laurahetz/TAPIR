package pp

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Code from https://github.com/yacovm/PoL

func TestPointProofCommitment(t *testing.T) {
	N := 128
	pp := NewPublicParams(N)

	var m Vec
	for i := 0; i < N; i++ {
		m = append(m, c.NewRandomZr(rand.Reader))
	}

	C := Commit(pp, m)

	for i := 0; i < pp.N; i++ {
		mi, π := Open(pp, i, m)

		err := Verify(pp, mi, π, C, i)
		assert.NoError(t, err)

		err = Verify(pp, mi, π, C, (i+1)%pp.N)
		assert.Error(t, err)
	}
}

func TestUpdate(t *testing.T) {
	N := 128
	pp := NewPublicParams(N)

	var m Vec
	for i := 0; i < N; i++ {
		m = append(m, c.NewRandomZr(rand.Reader))
	}

	C := Commit(pp, m)

	var m2 Vec
	for i := 0; i < N; i++ {
		m2 = append(m, c.NewRandomZr(rand.Reader))
	}

	for i := 0; i < pp.N; i++ {
		Update(pp, C, m, m2[i], i)
		m[i] = m2[i]
		_, π := Open(pp, i, m)

		err := Verify(pp, m2[i], π, C, i)
		assert.NoError(t, err)
	}
}

func TestAggregation(t *testing.T) {
	N := 8
	pp := NewPublicParams(N)

	var m1 Vec
	var m2 Vec
	for i := 0; i < N; i++ {
		m1 = append(m1, c.NewRandomZr(rand.Reader))
		m2 = append(m2, c.NewRandomZr(rand.Reader))
	}

	C1 := Commit(pp, m1)
	C2 := Commit(pp, m2)

	for i := 0; i < 1; i++ {
		m1, π1 := Open(pp, i, m1)
		err := Verify(pp, m1, π1, C1, i)
		assert.NoError(t, err)

		m2, π2 := Open(pp, i, m2)
		err = Verify(pp, m2, π2, C2, i)
		assert.NoError(t, err)

		commitments := G1v{C1, C2}
		π := Aggregate(pp, commitments, []*G1{π1, π2}, RO)

		Σ := Vec{m1, m2}.InnerProd(Vec{RO(pp, commitments, 0), RO(pp, commitments, 1)})

		err = VerifyAggregation(pp, []int{i, i}, commitments, π, Σ, RO)
		assert.NoError(t, err)
	}
}
