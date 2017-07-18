package par2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestByteToUint16LEArray(t *testing.T) {
	bs := []byte{0x1, 0x2, 0x3, 0x4, 0x5}
	expectedU16s := []uint16{0x201, 0x403}
	require.Equal(t, expectedU16s, byteToUint16LEArray(bs))
}

func TestUint16LEToByteArray(t *testing.T) {
	u16s := []uint16{0x201, 0x403}
	expectedBs := []byte{0x1, 0x2, 0x3, 0x4}
	require.Equal(t, expectedBs, uint16LEToByteArray(u16s))
}
