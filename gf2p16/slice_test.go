package gf2p16

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func byteToUint16LEArray(bs []byte) []uint16 {
	u16s := make([]uint16, len(bs)/2)
	for i := range u16s {
		u16s[i] = binary.LittleEndian.Uint16(bs[2*i:])
	}
	return u16s
}

func uint16LEToByteArray(u16s []uint16) []byte {
	bs := make([]byte, 2*len(u16s))
	for i, x := range u16s {
		binary.LittleEndian.PutUint16(bs[2*i:], x)
	}
	return bs
}

func mulAndAddSlice(c T, in, out []uint16) {
	for i, x := range in {
		out[i] ^= uint16(c.Times(T(x)))
	}
}

func TestMulByteSliceLE(t *testing.T) {
	in := []byte{0xff, 0xfe, 0xaa, 0xab}
	out := []byte{0x3, 0x4, 0x5, 0x6}
	c := T(0x3)

	expectedOutU16 := make([]uint16, 2)
	mulAndAddSlice(c, byteToUint16LEArray(in), expectedOutU16)
	expectedOut := uint16LEToByteArray(expectedOutU16)

	MulByteSliceLE(c, in, out)

	require.Equal(t, expectedOut, out)
}

func TestMulAndAddByteSliceLE(t *testing.T) {
	in := []byte{0xff, 0xfe, 0xaa, 0xab}
	out := []byte{0x3, 0x4, 0x5, 0x6}
	c := T(0x3)

	expectedOutU16 := byteToUint16LEArray(out)
	mulAndAddSlice(c, byteToUint16LEArray(in), expectedOutU16)
	expectedOut := uint16LEToByteArray(expectedOutU16)

	MulAndAddByteSliceLE(c, in, out)

	require.Equal(t, expectedOut, out)
}
