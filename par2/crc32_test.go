package par2

import (
	"fmt"
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/require"
)

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
