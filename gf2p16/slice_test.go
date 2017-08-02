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

func tToUint16Array(ts []T) []uint16 {
	u16s := make([]uint16, len(ts))
	for i := range u16s {
		u16s[i] = uint16(ts[i])
	}
	return u16s
}

func uint16ToTArray(u16s []uint16) []T {
	ts := make([]T, len(u16s))
	for i := range ts {
		ts[i] = T(u16s[i])
	}
	return ts
}

func mulAndAddUint16Slice(c T, in, out []uint16) {
	for i, x := range in {
		out[i] ^= uint16(c.Times(T(x)))
	}
}

func testMulByteSliceLE(t *testing.T, mulFn func(T, []byte, []byte)) {
	in := []byte{0xff, 0xfe, 0xaa, 0xab}
	out := []byte{0x3, 0x4, 0x5, 0x6}
	c := T(0x3)

	expectedOutU16 := make([]uint16, 2)
	mulAndAddUint16Slice(c, byteToUint16LEArray(in), expectedOutU16)
	expectedOut := uint16LEToByteArray(expectedOutU16)

	mulFn(c, in, out)

	require.Equal(t, expectedOut, out)
}

func TestMulByteSliceLE(t *testing.T) {
	t.Run("generic", func(t *testing.T) {
		testMulByteSliceLE(t, mulByteSliceLEGeneric)
	})
	t.Run("exported", func(t *testing.T) {
		testMulByteSliceLE(t, MulByteSliceLE)
	})
	if platformLittleEndian {
		t.Run("platformLE", func(t *testing.T) {
			testMulByteSliceLE(t, mulByteSliceLEPlatformLE)
		})
	}
}

func testMulAndAddByteSliceLE(t *testing.T, mulAndAddFn func(T, []byte, []byte)) {
	in := []byte{0xff, 0xfe, 0xaa, 0xab}
	out := []byte{0x3, 0x4, 0x5, 0x6}
	c := T(0x3)

	expectedOutU16 := byteToUint16LEArray(out)
	mulAndAddUint16Slice(c, byteToUint16LEArray(in), expectedOutU16)
	expectedOut := uint16LEToByteArray(expectedOutU16)

	mulAndAddFn(c, in, out)

	require.Equal(t, expectedOut, out)
}

func TestMulAndAddByteSliceLE(t *testing.T) {
	t.Run("generic", func(t *testing.T) {
		testMulAndAddByteSliceLE(t, mulAndAddByteSliceLEGeneric)
	})
	t.Run("exported", func(t *testing.T) {
		testMulAndAddByteSliceLE(t, MulAndAddByteSliceLE)
	})
	if platformLittleEndian {
		t.Run("platformLE", func(t *testing.T) {
			testMulAndAddByteSliceLE(t, mulAndAddByteSliceLEPlatformLE)
		})
	}
}

func makeBytes(tb testing.TB, rand *rand.Rand, byteCount int) []byte {
	bs := make([]byte, byteCount)
	n, err := rand.Read(bs)
	require.NoError(tb, err)
	require.Equal(tb, byteCount, n)
	return bs
}

func testMulSlice(t *testing.T, byteCount int, mulFn func(T, []T, []T)) {
	rand := rand.New(rand.NewSource(1))

	in := byteToTLEArray(makeBytes(t, rand, byteCount))
	out := make([]T, len(in))
	c := T(0x3)

	expectedOutU16 := make([]uint16, len(in))
	mulAndAddUint16Slice(c, tToUint16Array(in), expectedOutU16)
	expectedOut := uint16ToTArray(expectedOutU16)

	mulFn(c, in, out)

	require.Equal(t, expectedOut, out)
}

func TestMulSlice(t *testing.T) {
	for _, byteCount := range []int{10, 100, 1000} {
		t.Run(fmt.Sprintf("generic-%d", byteCount), func(t *testing.T) {
			testMulSlice(t, byteCount, mulSliceGeneric)
		})
		t.Run(fmt.Sprintf("nongeneric-%d", byteCount), func(t *testing.T) {
			testMulSlice(t, byteCount, mulSlice)
		})
	}
}

func testMulAndAddSlice(t *testing.T, byteCount int, mulAndAddFn func(T, []T, []T)) {
	rand := rand.New(rand.NewSource(1))

	in := byteToTLEArray(makeBytes(t, rand, byteCount))
	out := byteToTLEArray(makeBytes(t, rand, byteCount))
	c := T(0x3)

	expectedOutU16 := tToUint16Array(out)
	mulAndAddUint16Slice(c, tToUint16Array(in), expectedOutU16)
	expectedOut := uint16ToTArray(expectedOutU16)

	mulAndAddFn(c, in, out)

	require.Equal(t, expectedOut, out)
}

func TestMulAndAddSlice(t *testing.T) {
	for _, byteCount := range []int{10, 100, 1000} {
		t.Run(fmt.Sprintf("generic-%d", byteCount), func(t *testing.T) {
			testMulAndAddSlice(t, byteCount, mulAndAddSliceGeneric)
		})
		t.Run(fmt.Sprintf("nongeneric-%d", byteCount), func(t *testing.T) {
			testMulAndAddSlice(t, byteCount, mulAndAddSlice)
		})
	}
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

func byteToTLEArray(bs []byte) []T {
	ts := make([]T, len(bs)/2)
	for i := range ts {
		ts[i] = T(binary.LittleEndian.Uint16(bs[2*i:]))
	}
	return ts
}

func benchMulSlice(b *testing.B, byteCount int) {
	b.SetBytes(int64(byteCount))

	rand := rand.New(rand.NewSource(1))

	in := byteToTLEArray(makeBytes(b, rand, byteCount))
	out := make([]T, byteCount/2)
	c := T(5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mulSlice(c, in, out)
	}
}

func BenchmarkMulSlice(b *testing.B) {
	runMulBenchmark(b, benchMulSlice)
}

func benchMulAndAddSlice(b *testing.B, byteCount int) {
	b.SetBytes(int64(byteCount))

	rand := rand.New(rand.NewSource(1))

	in := byteToTLEArray(makeBytes(b, rand, byteCount))
	out := make([]T, byteCount/2)
	c := T(5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mulAndAddSlice(c, in, out)
	}
}

func BenchmarkMulAndAddSlice(b *testing.B) {
	runMulBenchmark(b, benchMulAndAddSlice)
}

func benchMulAndAddUint16Slice(b *testing.B, byteCount int) {
	b.SetBytes(int64(byteCount))

	rand := rand.New(rand.NewSource(1))

	in := byteToUint16LEArray(makeBytes(b, rand, byteCount))
	out := make([]uint16, byteCount/2)
	c := T(5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mulAndAddUint16Slice(c, in, out)
	}
}

func BenchmarkMulAndAddUint16Slice(b *testing.B) {
	runMulBenchmark(b, benchMulAndAddUint16Slice)
}
