package gf2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoly64TimesBasic(t *testing.T) {
	// (x + 1)(x + 1) = x^2 + 2x + 1 = x^2 + 1.
	require.Equal(t, Poly64(5), Poly64(3).Times(3))
	// (x^2 + 1)(x + 1) = x^3 + x^2 + x + 1.
	require.Equal(t, Poly64(15), Poly64(5).Times(3))
}

func TestPoly64TimesCommutative(t *testing.T) {
	for i := Poly64(0); i < Poly64(1<<8); i++ {
		for j := Poly64(0); j < Poly64(1<<8); j++ {
			require.Equal(t, i.Times(j), j.Times(i), "i=%d, j=%d", i, j)
		}
	}
}

func TestPoly64Div(t *testing.T) {
	for i := Poly64(0); i < Poly64(1<<8); i++ {
		for j := Poly64(1); j < Poly64(1<<8); j++ {
			q, r := i.Div(j)
			require.Equal(t, i, q.Times(j).Plus(r), "i=%d, j=%d, q=%d, r=%d", i, j, q, r)
			require.True(t, r == 0 || (ilog2(uint64(r)) < ilog2(uint64(j))), "i=%d, j=%d, q=%d, r=%d", i, j, q, r)
		}
	}
}

func irreducible(n Poly64) bool {
	for i := Poly64(2); i < n; i++ {
		if _, r := n.Div(i); r == 0 {
			return false
		}
	}
	return true
}

func TestIrreducible(t *testing.T) {
	expectedIrreducibles := []Poly64{
		// x, x + 1
		2, 3,
		// x^2 + x + 1
		7,
		// x^3 + x + 1, x^3 + x^2 + 1
		11, 13,
		// x^4 + x + 1, x^4 + x^3 + 1, x^4 + x^3 + x^2 + x + 1
		19, 25,
		// x^4 + x^3 + x^2 + x + 1
		31,
		// x^5 + x^2 + 1, x^5 + x^3 + 1, x^5 + x^3 + x^2 + x + 1
		37, 41, 47,
		// x^5 + x^4 + x^2 + x + 1, x^5 + x^4 + x^3 + x + 1
		55, 59,
		// x^5 + x^4 + x^3 + x^2 + 1
		61,
	}

	var irreducibles []Poly64
	for i := Poly64(2); i < 64; i++ {
		if irreducible(i) {
			irreducibles = append(irreducibles, i)
		}
	}

	require.Equal(t, expectedIrreducibles, irreducibles)
}

func TestMod11(t *testing.T) {
	for i := Poly64(1); i < 8; i++ {
		foundInvMod11 := false
		for j := Poly64(1); j < 8; j++ {
			_, prodMod11 := i.Times(j).Div(11)
			require.NotEqual(t, Poly64(0), prodMod11, "i=%d, j=%d", i, j)
			if prodMod11 == 1 {
				require.False(t, foundInvMod11, "i=%d, j=%d", i, j)
				foundInvMod11 = true
			}
		}
		assert.True(t, foundInvMod11, "i=%d", i)
	}
}
