package rsec16

import (
	"testing"

	"github.com/akalin/gopar/gf2p16"
	"github.com/stretchr/testify/require"
)

func byteToUint16LEArray(bs []byte) []uint16 {
	u16s := make([]uint16, len(bs)/2)
	for i := 0; i < len(u16s); i++ {
		u16s[i] = uint16(bs[2*i]) + uint16(bs[2*i+1])<<8
	}
	return u16s
}

func uint16LEToByteArray(u16s []uint16) []byte {
	bs := make([]byte, 2*len(u16s))
	for i := 0; i < len(u16s); i++ {
		bs[2*i] = byte(u16s[i])
		bs[2*i+1] = byte(u16s[i] >> 8)
	}
	return bs
}

func applyMatrixNaive(m gf2p16.Matrix, in, out [][]byte) {
	if len(in[0]) != len(out[0]) {
		panic("mismatched lengths")
	}

	inU16 := make([][]uint16, len(in))
	for i := range inU16 {
		inU16[i] = byteToUint16LEArray(in[i])
	}
	n := gf2p16.NewMatrixFromFunction(len(inU16), len(inU16[0]), func(i, j int) gf2p16.T {
		return gf2p16.T(inU16[i][j])
	})
	prod := m.Times(n)
	for i := 0; i < len(out); i++ {
		outIU16 := make([]uint16, len(out[0])/2)
		for j := range outIU16 {
			outIU16[j] = uint16(prod.At(i, j))
		}
		copy(out[i], uint16LEToByteArray(outIU16))
	}
}

func makeIn(inputCount, dataByteCount int) [][]byte {
	in := make([][]byte, inputCount)
	for i := 0; i < len(in); i++ {
		in[i] = make([]byte, dataByteCount)
		for j := 0; j < dataByteCount; j++ {
			in[i][j] = byte(i + j)
		}
	}
	return in
}

func makeOut(outputCount, dataByteCount int) [][]byte {
	out := make([][]byte, outputCount)
	for i := 0; i < len(out); i++ {
		out[i] = make([]byte, dataByteCount)
	}
	return out
}

func TestApplyMatrixIdentity(t *testing.T) {
	count := 4
	dataByteCount := 10
	m := gf2p16.NewIdentityMatrix(count)

	in := makeIn(count, dataByteCount)

	out := makeOut(count, dataByteCount)
	applyMatrix(m, in, out)

	require.Equal(t, in, out)
}

func TestApplyMatrixVandermonde(t *testing.T) {
	inputCount := 4
	outputCount := 3
	dataByteCount := 10
	m := newTestVandermondeMatrix(outputCount, inputCount)

	in := makeIn(inputCount, dataByteCount)

	out := makeOut(outputCount, dataByteCount)
	applyMatrix(m, in, out)

	expectedOut := makeOut(outputCount, dataByteCount)
	applyMatrixNaive(m, in, expectedOut)

	require.Equal(t, expectedOut, out)
}

func benchmarkApplyMatrix(b *testing.B, inputCount, outputCount, dataByteCount int) {
	b.SetBytes(int64(dataByteCount))

	m := newTestVandermondeMatrix(outputCount, inputCount)

	in := makeIn(inputCount, dataByteCount)

	out := makeOut(outputCount, dataByteCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		applyMatrix(m, in, out)
	}
}

func BenchmarkApplyMatrix(b *testing.B) {
	b.Run("ic=3,oc=4,db=1K", func(b *testing.B) {
		benchmarkApplyMatrix(b, 3, 4, 1024)
	})
	b.Run("ic=3,oc=4,db=1M", func(b *testing.B) {
		benchmarkApplyMatrix(b, 3, 4, 1024*1024)
	})
	b.Run("ic=3,oc=4,db=10M", func(b *testing.B) {
		benchmarkApplyMatrix(b, 3, 4, 10*1024*1024)
	})
}
