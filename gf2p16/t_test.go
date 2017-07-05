package gf2p16

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimesDiv(t *testing.T) {
	// TODO: Test with finer granularity once we use tables.
	for i := 0; i < (1 << 16); i += 1 << 12 {
		for j := 0; j < (1 << 16); j += 1 << 12 {
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
