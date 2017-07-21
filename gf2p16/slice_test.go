package gf2p16

import (
	"encoding/binary"
	"fmt"
	"math/rand"
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

func runMulBenchmark(b *testing.B, fn func(*testing.B, int)) {
	for _, i := range []uint{2, 4, 8, 12, 14, 16, 20, 24, 26, 28} {
		var name string
		byteCount := 1 << i
		if byteCount >= 1024*1024 {
			name = fmt.Sprintf("%dM", byteCount/(1024*1024))
		} else if byteCount >= 1024 {
			name = fmt.Sprintf("%dK", byteCount/1024)
		} else {
			name = fmt.Sprintf("%d", byteCount)
		}
		b.Run(name, func(b *testing.B) {
			fn(b, byteCount)
		})
	}
}

func makeBytes(b *testing.B, rand *rand.Rand, byteCount int) []byte {
	bs := make([]byte, byteCount)
	n, err := rand.Read(bs)
	require.NoError(b, err)
	require.Equal(b, byteCount, n)
	return bs
}

func benchMulByteSliceLE(b *testing.B, byteCount int) {
	b.SetBytes(int64(byteCount))

	rand := rand.New(rand.NewSource(1))

	in := makeBytes(b, rand, byteCount)
	out := make([]byte, byteCount)
	c := T(5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MulByteSliceLE(c, in, out)
	}
}

func BenchmarkMulByteSliceLE(b *testing.B) {
	runMulBenchmark(b, benchMulByteSliceLE)
}

func benchMulAndAddByteSliceLE(b *testing.B, byteCount int) {
	b.SetBytes(int64(byteCount))

	rand := rand.New(rand.NewSource(1))

	in := makeBytes(b, rand, byteCount)
	out := make([]byte, byteCount)
	c := T(5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MulAndAddByteSliceLE(c, in, out)
	}
}

func BenchmarkMulAndAddByteSliceLE(b *testing.B) {
	runMulBenchmark(b, benchMulAndAddByteSliceLE)
}
