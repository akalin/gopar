package rsec16

import (
	"testing"

	"github.com/akalin/gopar/gf2p16"
	"github.com/stretchr/testify/require"
)

func TestVandermondeMatrix(t *testing.T) {
	alphaFunc := func(i int) gf2p16.T {
		return gf2p16.T(i)
	}
	for i := 1; i < 100; i++ {
		m := newVandermondeMatrix(i, i, alphaFunc)
		_, err := m.Inverse()
		require.NoError(t, err)
	}
}
