package gf2

import (
	"testing"

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
