package gf2p16

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMulSlice(t *testing.T) {
	in := []uint16{0x1, 0x2}
	out := []uint16{0x3, 0x4}
	MulSlice(0x3, in, out)
	require.Equal(t, []uint16{0x3, 0x6}, out)
}

func TestMulAndAddSlice(t *testing.T) {
	in := []uint16{0x1, 0x2}
	out := []uint16{0x3, 0x4}
	MulAndAddSlice(0x3, in, out)
	require.Equal(t, []uint16{0x0, 0x2}, out)
}
