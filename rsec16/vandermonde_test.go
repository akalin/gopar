package rsec16

import (
	"testing"

	"github.com/akalin/gopar/gf2p16"
	"github.com/stretchr/testify/require"
)

func newTestVandermondeMatrix(rows, columns int) gf2p16.Matrix {
	return newVandermondeMatrix(rows, columns, func(i int) gf2p16.T {
		return gf2p16.T(i)
	})
}

func TestVandermondeMatrix(t *testing.T) {
	for i := 1; i < 100; i++ {
		m := newTestVandermondeMatrix(i, i)
		_, err := m.Inverse()
		require.NoError(t, err)
	}
}
