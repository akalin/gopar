package rsec16

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/akalin/gopar/gf2p16"
	"github.com/stretchr/testify/require"
)

func TestCalculateParallelParams(t *testing.T) {
	for totalLength := 1; totalLength < 256; totalLength++ {
		for numGoroutines := 1; numGoroutines < 17; numGoroutines++ {
			for minPerGoroutineLength := 1; minPerGoroutineLength < 4; minPerGoroutineLength++ {
				for perGoroutineLengthDivisor := 1; perGoroutineLengthDivisor < 8; perGoroutineLengthDivisor++ {
					perGoroutineLength, newNumGoroutines := calculateParallelParams(totalLength, numGoroutines, minPerGoroutineLength, perGoroutineLengthDivisor)
					require.True(t, perGoroutineLength >= minPerGoroutineLength)
					require.Equal(t, 0, perGoroutineLength%perGoroutineLengthDivisor)
					require.True(t, perGoroutineLength*(newNumGoroutines-1) < totalLength)
					require.True(t, perGoroutineLength*newNumGoroutines >= totalLength)
				}
			}
		}
	}
}

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

type applyMatrixFunc func(gf2p16.Matrix, [][]byte, [][]byte)

func applyNumGoroutines(fn func(gf2p16.Matrix, [][]byte, [][]byte, int), numGoroutines int) applyMatrixFunc {
	return func(m gf2p16.Matrix, in, out [][]byte) {
		fn(m, in, out, numGoroutines)
	}
}

func runApplyMatrixTest(t *testing.T, fn func(*testing.T, applyMatrixFunc)) {
	t.Run("Single", func(t *testing.T) { fn(t, applyMatrixSingle) })
	var testNumGoroutines = []int{2, 4, 8}
	for _, numGoroutines := range testNumGoroutines {
		// Capture range variable.
		numGoroutines := numGoroutines
		t.Run(fmt.Sprintf("OutParallel-%d", numGoroutines), func(t *testing.T) {
			fn(t, applyNumGoroutines(applyMatrixParallelOut, numGoroutines))
		})
	}
	for _, numGoroutines := range testNumGoroutines {
		// Capture range variable.
		numGoroutines := numGoroutines
		t.Run(fmt.Sprintf("DataParallel-%d", numGoroutines), func(t *testing.T) {
			fn(t, applyNumGoroutines(applyMatrixParallelData, numGoroutines))
		})
	}
}

func testApplyMatrixIdentity(t *testing.T, applyMatrixFn applyMatrixFunc) {
	count := 4
	dataByteCount := 10
	m := gf2p16.NewIdentityMatrix(count)

	in := makeIn(count, dataByteCount)

	out := makeOut(count, dataByteCount)
	applyMatrixFn(m, in, out)

	require.Equal(t, in, out)
}

func TestApplyMatrixIdentity(t *testing.T) {
	runApplyMatrixTest(t, testApplyMatrixIdentity)
}

func testApplyMatrixVandermonde(t *testing.T, applyMatrixFn applyMatrixFunc) {
	inputCount := 4
	outputCount := 3
	dataByteCount := 10
	m := newTestVandermondeMatrix(outputCount, inputCount)

	in := makeIn(inputCount, dataByteCount)

	out := makeOut(outputCount, dataByteCount)
	applyMatrixFn(m, in, out)

	expectedOut := makeOut(outputCount, dataByteCount)
	applyMatrixNaive(m, in, expectedOut)

	require.Equal(t, expectedOut, out)
}

func TestApplyMatrixVandermonde(t *testing.T) {
	runApplyMatrixTest(t, testApplyMatrixVandermonde)
}

func runApplyMatrixBenchmark(b *testing.B, fn func(*testing.B, applyMatrixFunc)) {
	b.Run("Single", func(b *testing.B) { fn(b, applyMatrixSingle) })
	var benchmarkNumGoroutines []int
	for i := 2; i <= runtime.GOMAXPROCS(0); i *= 2 {
		benchmarkNumGoroutines = append(benchmarkNumGoroutines, i)
	}
	for _, numGoroutines := range benchmarkNumGoroutines {
		// Capture range variable.
		numGoroutines := numGoroutines
		b.Run(fmt.Sprintf("OutParallel-%d", numGoroutines), func(b *testing.B) {
			fn(b, applyNumGoroutines(applyMatrixParallelOut, numGoroutines))
		})
	}
	for _, numGoroutines := range benchmarkNumGoroutines {
		// Capture range variable.
		numGoroutines := numGoroutines
		b.Run(fmt.Sprintf("DataParallel-%d", numGoroutines), func(b *testing.B) {
			fn(b, applyNumGoroutines(applyMatrixParallelData, numGoroutines))
		})
	}
}

func sizeString(size int) string {
	if size%(1024*1024) == 0 {
		return fmt.Sprintf("%dM", size/(1024*1024))
	} else if size%1024 == 0 {
		return fmt.Sprintf("%dK", size/1024)
	} else {
		return fmt.Sprintf("%d", size)
	}
}

type applyMatrixBenchmarkConfig struct {
	inputCount    int
	outputCount   int
	dataByteCount int
}

func (config applyMatrixBenchmarkConfig) String() string {
	return fmt.Sprintf("ic=%d,oc=%d,db=%s", config.inputCount, config.outputCount, sizeString(config.dataByteCount))
}

func benchmarkApplyMatrix(b *testing.B, config applyMatrixBenchmarkConfig, applyMatrixFn applyMatrixFunc) {
	b.SetBytes(int64(config.dataByteCount))

	m := newVandermondeMatrix(config.outputCount, config.inputCount, func(i int) gf2p16.T { return gf2p16.T(i) })

	in := makeIn(config.inputCount, config.dataByteCount)

	out := makeOut(config.outputCount, config.dataByteCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		applyMatrixFn(m, in, out)
	}
}

func BenchmarkApplyMatrix(b *testing.B) {
	gf2p16.LoadCaches()
	configs := []applyMatrixBenchmarkConfig{
		{3, 4, 1024},
		{3, 4, 1024 * 1024},
		{3, 4, 10 * 1024 * 1024},
		{3, 16, 1024},
		{3, 16, 1024 * 1024},
		{3, 16, 10 * 1024 * 1024},
		{3, 64, 1024},
		{3, 64, 1024 * 1024},
		{3, 64, 10 * 1024 * 1024},
	}
	for _, config := range configs {
		b.Run(config.String(), func(b *testing.B) {
			runApplyMatrixBenchmark(b, func(b *testing.B, applyMatrixFn applyMatrixFunc) {
				benchmarkApplyMatrix(b, config, applyMatrixFn)
			})
		})
	}
}
