package rsec16

import (
	"testing"

	"github.com/akalin/gopar/gf2p16"
	"github.com/stretchr/testify/require"
)

func applyMatrixNaive(m gf2p16.Matrix, in, out [][]uint16) {
	if len(in[0]) != len(out[0]) {
		panic("mismatched lengths")
	}

	n := gf2p16.NewMatrixFromFunction(len(in), len(in[0]), func(i, j int) gf2p16.T {
		return gf2p16.T(in[i][j])
	})
	prod := m.Times(n)
	for i := 0; i < len(out); i++ {
		for j := 0; j < len(out[0]); j++ {
			out[i][j] = uint16(prod.At(i, j))
		}
	}
}

func makeOut(outputCount, dataByteCount int) [][]uint16 {
	out := make([][]uint16, outputCount)
	for i := 0; i < len(out); i++ {
		out[i] = make([]uint16, dataByteCount/2)
	}
	return out
}

func TestApplyMatrix(t *testing.T) {
	inputCount := 4
	outputCount := 3
	dataByteCount := 10
	m := newTestVandermondeMatrix(outputCount, inputCount)

	in := make([][]uint16, inputCount)
	for i := 0; i < len(in); i++ {
		in[i] = make([]uint16, dataByteCount/2)
		for j := 0; j < dataByteCount/2; j++ {
			in[i][j] = uint16(i + j)
		}
	}

	out := makeOut(outputCount, dataByteCount)
	applyMatrix(m, in, out)

	expectedOut := makeOut(outputCount, dataByteCount)
	applyMatrixNaive(m, in, expectedOut)

	require.Equal(t, expectedOut, out)
}
