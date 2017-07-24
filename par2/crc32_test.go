package par2

import (
	"fmt"
	"hash/crc32"
	"testing"

	"github.com/akalin/gopar/gf2"
	"github.com/stretchr/testify/require"
)

// TODO: Use math.bits when it becomes available in Go 1.9.

const m0 = 0x5555555555555555 // 01010101 ...
const m1 = 0x3333333333333333 // 00110011 ...
const m2 = 0x0f0f0f0f0f0f0f0f // 00001111 ...
const m3 = 0x00ff00ff00ff00ff // etc.

func reverse32(x uint32) uint32 {
	const m = 1<<32 - 1
	x = x>>1&(m0&m) | x&(m0&m)<<1
	x = x>>2&(m1&m) | x&(m1&m)<<2
	x = x>>4&(m2&m) | x&(m2&m)<<4
	x = x>>8&(m3&m) | x&(m3&m)<<8
	return x>>16 | x<<16
}

// ieeeModulus is 0x100000000 | reverse32(crc32.IEEE).
const ieeeModulus = 0x104c11db7

// crcz32 is the function such that
//
//   crc32.ChecksumIEEE(a) = ffffffff ^ crcz(ffffffff.. ^ a).
func crcz32(p []byte) uint32 {
	return 0xffffffff ^ crc32.Update(0xffffffff, crc32.IEEETable, p)
}

// crcz32ReverseBytesShiftLeft returns crcz32(reverseBytes(a) << n),
// where reverseBytes() maps math.bits.Reverse8 over all bytes of a,
// given crcz32(a) and n. n must be less than 32.
func crcz32ReverseBytesShiftLeft(crcz uint32, n uint) uint32 {
	t := gf2.Poly64(reverse32(crcz))
	t <<= n
	_, t = t.Div(ieeeModulus)
	return reverse32(uint32(t))
}

func TestCRCZ32ReverseBytesShiftLeft(t *testing.T) {
	a := []byte{0x80, 0x40, 0, 0, 0}
	aSL1 := []byte{0x40, 0x20, 0, 0, 0}
	aSL2 := []byte{0x20, 0x10, 0, 0, 0}
	aSL7 := []byte{0x81, 0, 0, 0, 0}
	aSL8 := []byte{0x80, 0x40, 0, 0, 0, 0}
	aSL31 := []byte{0x81, 0, 0, 0, 0, 0, 0, 0}

	crc := crcz32(a)

	require.Equal(t, crcz32(aSL1), crcz32ReverseBytesShiftLeft(crc, 1))
	require.Equal(t, crcz32(aSL2), crcz32ReverseBytesShiftLeft(crc, 2))
	require.Equal(t, crcz32(aSL7), crcz32ReverseBytesShiftLeft(crc, 7))
	require.Equal(t, crcz32(aSL8), crcz32ReverseBytesShiftLeft(crc, 8))
	require.Equal(t, crcz32(aSL31), crcz32ReverseBytesShiftLeft(crc, 31))
}

// crc32ReverseBytesShiftLeft returns crc32(reverseBytes(a) << n),
// given crc32(a), crc32(len(a) copies of 0), and
// crc32(len(reverseBytes(a) << n) copies of 0). n must be less than
// 32.
func crc32ReverseBytesShiftLeft(crc, crc0in, crc0out uint32, n uint) uint32 {
	return crcz32ReverseBytesShiftLeft(crc^crc0in, n) ^ crc0out
}

func TestCRCShiftLeftReversed(t *testing.T) {
	a := []byte{0x80, 0x40, 0, 0, 0}
	aSL1 := []byte{0x40, 0x20, 0, 0, 0}
	aSL2 := []byte{0x20, 0x10, 0, 0, 0}
	aSL7 := []byte{0x81, 0, 0, 0, 0}
	aSL8 := []byte{0x80, 0x40, 0, 0, 0, 0}
	aSL31 := []byte{0x81, 0, 0, 0, 0, 0, 0, 0}

	crc32 := crc32.ChecksumIEEE

	crc := crc32(a)
	crc05 := crc32(make([]byte, 5))
	crc06 := crc32(make([]byte, 6))
	crc08 := crc32(make([]byte, 8))

	require.Equal(t, crc32(aSL1), crc32ReverseBytesShiftLeft(crc, crc05, crc05, 1))
	require.Equal(t, crc32(aSL2), crc32ReverseBytesShiftLeft(crc, crc05, crc05, 2))
	require.Equal(t, crc32(aSL7), crc32ReverseBytesShiftLeft(crc, crc05, crc05, 7))
	require.Equal(t, crc32(aSL8), crc32ReverseBytesShiftLeft(crc, crc05, crc06, 8))
	require.Equal(t, crc32(aSL31), crc32ReverseBytesShiftLeft(crc, crc05, crc08, 31))
}

func testCRC32Window(t *testing.T, windowSize, updateCount int) {
	w := newCRC32Window(windowSize)

	bs := make([]byte, windowSize)
	for i := range bs {
		bs[i] = ^byte(i)
	}
	crc := crc32.ChecksumIEEE(bs)

	for i := 0; i < updateCount; i++ {
		oldLeader := bs[0]
		newTrailer := bs[len(bs)-1] - 1
		crc = w.update(crc, oldLeader, newTrailer)
		bs = append(bs, newTrailer)[1:]
		require.Equal(t, crc32.ChecksumIEEE(bs), crc)
	}

	for i := range bs {
		bs[i] = byte(i)
	}
	crc = crc32.ChecksumIEEE(bs)

	for i := 0; i < updateCount; i++ {
		oldLeader := bs[0]
		newTrailer := bs[len(bs)-1] - 1
		crc = w.update(crc, oldLeader, newTrailer)
		bs = append(bs, newTrailer)[1:]
		require.Equal(t, crc32.ChecksumIEEE(bs), crc)
	}
}

func TestCRC32Window(t *testing.T) {
	t.Run("ws=4,uc=100", func(t *testing.T) {
		testCRC32Window(t, 4, 100)
	})
	t.Run("ws=10,uc=50", func(t *testing.T) {
		testCRC32Window(t, 10, 50)
	})
	t.Run("ws=128,uc=20", func(t *testing.T) {
		testCRC32Window(t, 128, 20)
	})
}

func benchmarkCRC32(b *testing.B, windowSize int) {
	b.SetBytes(int64(windowSize))

	bs := make([]byte, windowSize+b.N+1)
	for i := range bs {
		bs[i] = ^byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		crc32.ChecksumIEEE(bs[i+1 : i+1+windowSize])
	}
}

// 16 and 64 are cutoff points for various implementations of
// crc32.ChecksumIEEE.
var benchWindowSizes = []int{4, 8, 15, 16, 63, 64, 128, 256, 512, 1024}

func BenchmarkCRC32(b *testing.B) {
	for _, windowSize := range append([]int{1}, benchWindowSizes...) {
		b.Run(fmt.Sprintf("ws=%d", windowSize), func(b *testing.B) {
			benchmarkCRC32(b, windowSize)
		})
	}
}

func benchmarkCRC32Window(b *testing.B, windowSize int) {
	b.SetBytes(1)

	w := newCRC32Window(windowSize)

	bs := make([]byte, windowSize+b.N+1)
	for i := range bs {
		bs[i] = ^byte(i)
	}
	crc := crc32.ChecksumIEEE(bs[:windowSize])

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		crc = w.update(crc, bs[i+1], bs[i+1+windowSize])
	}
}

func BenchmarkCRC32Window(b *testing.B) {
	for _, windowSize := range benchWindowSizes {
		b.Run(fmt.Sprintf("ws=%d", windowSize), func(b *testing.B) {
			benchmarkCRC32Window(b, windowSize)
		})
	}
}

func benchmarkNewCRC32Window(b *testing.B, windowSize int) {
	for i := 0; i < b.N; i++ {
		newCRC32Window(windowSize)
	}
}

func BenchmarkNewCRC32Window(b *testing.B) {
	for _, windowSize := range benchWindowSizes {
		b.Run(fmt.Sprintf("ws=%d", windowSize), func(b *testing.B) {
			benchmarkNewCRC32Window(b, windowSize)
		})
	}
}
