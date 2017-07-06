package rsec16

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoderGenerateParity(t *testing.T) {
	data := [][]uint16{
		{0x1, 0x2},
		{0x3, 0x4},
		{0x5, 0x6},
		{0x7, 0x8},
		{0x9, 0xa},
	}
	c := NewCoder(5, 3)
	parity := c.GenerateParity(data)
	require.Equal(t, 3, len(parity))
	for _, row := range parity {
		require.Equal(t, 2, len(row))
	}
}
