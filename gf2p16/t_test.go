package gf2p16

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInverse(t *testing.T) {
	var invTable [order - 1]T
	for i := 1; i < (1 << 16); i++ {
		x := T(i)
		xInv := x.Inverse()
		require.NotEqual(t, T(0), xInv, "x=%d", x)
		if x != 1 {
			require.NotEqual(t, x, xInv, "x=%d", x)
		}
		require.Equal(t, T(0), invTable[x-1])
		invTable[x-1] = xInv
		require.Equal(t, T(1), x.Times(xInv), "x=%d", x)
		require.Equal(t, T(1), xInv.Times(x), "x=%d", x)
	}

	for i, xInv := range invTable {
		require.NotEqual(t, T(0), xInv, "i=%d", i)
	}
}

func TestOneDivInverse(t *testing.T) {
	for i := 1; i < (1 << 16); i++ {
		x := T(i)
		require.Equal(t, x.Inverse(), T(1).Div(x))
	}
}

func TestTimesDiv(t *testing.T) {
	for i := 0; i < (1 << 16); i += (1 << 7) {
		for j := 0; j < (1 << 16); j += (1 << 7) {
			x := T(i)
			y := T(j)
			p := x.Times(y)
			if y != 0 {
				require.Equal(t, x, p.Div(y), "x=%d, y=%d", x, y)
			}
			if x != 0 {
				require.Equal(t, y, p.Div(x), "x=%d, y=%d", x, y)
			}
		}
	}
}
