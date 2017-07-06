package rsec16

import (
	"testing"

	"github.com/akalin/gopar/gf2p16"
	"github.com/stretchr/testify/require"
)

func TestCauchyMatrix(t *testing.T) {
	xFunc := func(i int) gf2p16.T {
		return gf2p16.T(2 * i)
	}
	yFunc := func(i int) gf2p16.T {
		return gf2p16.T(2*i + 1)
	}
	for i := 1; i < 100; i++ {
		m := newCauchyMatrix(i, i, xFunc, yFunc)
		_, err := m.Inverse()
		require.NoError(t, err)
	}
}
