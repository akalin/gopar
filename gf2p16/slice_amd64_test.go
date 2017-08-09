package gf2p16

import (
	"math/rand"
	"testing"

	"github.com/klauspost/cpuid"
	"github.com/stretchr/testify/require"
)

func skipNonSSSE3(t *testing.T) {
	if !cpuid.CPU.SSSE3() {
		t.Skip("SSSE3 not supported; skipping")
	}
}

func fill(bs []byte, b byte) {
	for i := range bs {
		bs[i] = b
	}
}

func fill16(b byte) [16]byte {
	var bs [16]byte
	fill(bs[:], b)
	return bs
}

func TestStandardToAltMapSSSE3Unsafe(t *testing.T) {
	skipNonSSSE3(t)

	in0 := [16]byte{
		0x20, 0x21, 0x30, 0x31,
		0x40, 0x41, 0x50, 0x51,
		0x60, 0x61, 0x70, 0x71,
		0x80, 0x81, 0x90, 0x91,
	}

	in1 := [16]byte{
		0xa0, 0xa1, 0xb0, 0xb1,
		0xc0, 0xc1, 0xd0, 0xd1,
		0xe0, 0xe1, 0xf0, 0xf1,
		0x00, 0x01, 0x10, 0x11,
	}

	filler := fill16(0xfe)
	outLow := [2][16]byte{filler, filler}
	outHigh := [2][16]byte{filler, filler}

	expectedOutLow := [2][16]byte{{
		0x20, 0x30, 0x40, 0x50,
		0x60, 0x70, 0x80, 0x90,
		0xa0, 0xb0, 0xc0, 0xd0,
		0xe0, 0xf0, 0x00, 0x10,
	}, filler}

	expectedOutHigh := [2][16]byte{{
		0x21, 0x31, 0x41, 0x51,
		0x61, 0x71, 0x81, 0x91,
		0xa1, 0xb1, 0xc1, 0xd1,
		0xe1, 0xf1, 0x01, 0x11,
	}, filler}

	standardToAltMapSSSE3Unsafe(&in0, &in1, &outLow[0], &outHigh[0])

	require.Equal(t, expectedOutLow, outLow)
	require.Equal(t, expectedOutHigh, outHigh)
}

func TestStandardToAltMapSliceSSSE3Unsafe(t *testing.T) {
	skipNonSSSE3(t)

	rand := rand.New(rand.NewSource(1))

	in := makeBytes(t, rand, 32*10)
	out := make([]byte, len(in)+31)
	expectedOut := make([]byte, len(in)+31)
	fill(out, 0xfd)
	fill(expectedOut, 0xfd)

	for i := 0; i < 10; i++ {
		var in0, in1, outLow, outHigh [16]byte
		copy(in0[:], in[i*32+16:(i+1)*32])
		copy(in1[:], in[i*32:i*32+16])
		standardToAltMapSSSE3Unsafe(&in0, &in1, &outLow, &outHigh)
		copy(expectedOut[i*32+16:(i+1)*32], outLow[:])
		copy(expectedOut[i*32:i*32+16], outHigh[:])
	}

	standardToAltMapSliceSSSE3Unsafe(in, out)

	require.Equal(t, expectedOut, out)
}

func TestAltToStandardMapSSSE3Unsafe(t *testing.T) {
	skipNonSSSE3(t)

	inLow := [16]byte{
		0x20, 0x30, 0x40, 0x50,
		0x60, 0x70, 0x80, 0x90,
		0xa0, 0xb0, 0xc0, 0xd0,
		0xe0, 0xf0, 0x00, 0x10,
	}

	inHigh := [16]byte{
		0x21, 0x31, 0x41, 0x51,
		0x61, 0x71, 0x81, 0x91,
		0xa1, 0xb1, 0xc1, 0xd1,
		0xe1, 0xf1, 0x01, 0x11,
	}

	filler := fill16(0xfc)
	out0 := [2][16]byte{filler, filler}
	out1 := [2][16]byte{filler, filler}

	expectedOut0 := [2][16]byte{{
		0x20, 0x21, 0x30, 0x31,
		0x40, 0x41, 0x50, 0x51,
		0x60, 0x61, 0x70, 0x71,
		0x80, 0x81, 0x90, 0x91,
	}, filler}

	expectedOut1 := [2][16]byte{{
		0xa0, 0xa1, 0xb0, 0xb1,
		0xc0, 0xc1, 0xd0, 0xd1,
		0xe0, 0xe1, 0xf0, 0xf1,
		0x00, 0x01, 0x10, 0x11,
	}, filler}

	altToStandardMapSSSE3Unsafe(&inLow, &inHigh, &out0[0], &out1[0])

	require.Equal(t, expectedOut0, out0)
	require.Equal(t, expectedOut1, out1)
}

func TestAltToMapSliceSSSE3Unsafe(t *testing.T) {
	skipNonSSSE3(t)

	rand := rand.New(rand.NewSource(1))

	expectedOut := makeBytes(t, rand, 32*10+31)
	out := make([]byte, len(expectedOut))
	in := make([]byte, len(out)-31)
	fill(out, 0xdd)
	fill(expectedOut, 0xdd)

	standardToAltMapSliceSSSE3Unsafe(expectedOut[:len(in)], in)

	altToStandardMapSliceSSSE3Unsafe(in, out)

	require.Equal(t, expectedOut, out)
}

func mulAltMap(c T, inLow, inHigh, outLow, outHigh *[16]byte) {
	var in0, in1 [16]byte
	altToStandardMapSSSE3Unsafe(inLow, inHigh, &in0, &in1)
	var out [32]byte
	mulByteSliceLEGeneric(c, append(in0[:], in1[:]...), out[:])
	var out0, out1 [16]byte
	copy(out0[:], out[:16])
	copy(out1[:], out[16:])
	standardToAltMapSSSE3Unsafe(&out0, &out1, outLow, outHigh)
}

func TestMulAltMapSSSE3Unsafe(t *testing.T) {
	skipNonSSSE3(t)

	rand := rand.New(rand.NewSource(1))

	in := makeBytes(t, rand, 32)
	var inLow, inHigh [16]byte
	copy(inLow[:], in[:16])
	copy(inHigh[:], in[16:])

	c := T(rand.Int())

	filler := fill16(0xfb)
	expectedOutLow := [2][16]byte{filler, filler}
	expectedOutHigh := [2][16]byte{filler, filler}
	mulAltMap(c, &inLow, &inHigh, &expectedOutLow[0], &expectedOutHigh[0])

	outLow := [2][16]byte{filler, filler}
	outHigh := [2][16]byte{filler, filler}
	mulAltMapSSSE3Unsafe(&mulTable64[c], &inLow, &inHigh, &outLow[0], &outHigh[0])

	require.Equal(t, expectedOutLow, outLow)
	require.Equal(t, expectedOutHigh, outHigh)
}
