package rsec16

import (
	"sync"

	"github.com/akalin/gopar/gf2p16"
)

func applyMatrixSlice(m gf2p16.Matrix, in, out [][]byte, outStart, outEnd, dataStart, dataEnd int) {
	for i := outStart; i < outEnd; i++ {
		outSlice := out[i][dataStart:dataEnd]
		c := m.At(i, 0)
		inSlice := in[0][dataStart:dataEnd]
		gf2p16.MulByteSliceLE(c, inSlice, outSlice)
		for j := 1; j < len(in); j++ {
			c := m.At(i, j)
			inSlice := in[j][dataStart:dataEnd]
			gf2p16.MulAndAddByteSliceLE(c, inSlice, outSlice)
		}
	}
}

func applyMatrixSingle(m gf2p16.Matrix, in, out [][]byte) {
	if len(in[0]) != len(out[0]) {
		panic("mismatched lengths")
	}

	applyMatrixSlice(m, in, out, 0, len(out), 0, len(in[0]))
}

func calculateParallelParams(totalLength, numGoroutines, minPerGoroutineLength, perGoroutineLengthDivisor int) (perGoroutineLength, newNumGoroutines int) {
	perGoroutineLength = (totalLength + numGoroutines - 1) / numGoroutines
	if perGoroutineLength < minPerGoroutineLength {
		perGoroutineLength = minPerGoroutineLength
	}

	rem := perGoroutineLength % perGoroutineLengthDivisor
	if rem != 0 {
		perGoroutineLength += (perGoroutineLengthDivisor - rem)
	}

	newNumGoroutines = (totalLength + perGoroutineLength - 1) / perGoroutineLength
	return perGoroutineLength, newNumGoroutines
}

func applyMatrixParallelOut(m gf2p16.Matrix, in, out [][]byte, numGoroutines int) {
	if len(in[0]) != len(out[0]) {
		panic("mismatched lengths")
	}

	if numGoroutines < 1 {
		panic("invalid numGoroutines value")
	}

	outLength := len(out)
	perGoroutineOutLength, numGoroutines := calculateParallelParams(outLength, numGoroutines, 1, 1)
	if numGoroutines < 2 {
		applyMatrixSingle(m, in, out)
		return
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			start := i * perGoroutineOutLength
			end := start + perGoroutineOutLength
			if end > outLength {
				end = outLength
			}
			applyMatrixSlice(m, in, out, start, end, 0, len(in[0]))
		}(i)
	}

	wg.Wait()
}

func applyMatrixParallelData(m gf2p16.Matrix, in, out [][]byte, numGoroutines int) {
	if len(in[0]) != len(out[0]) {
		panic("mismatched lengths")
	}

	if numGoroutines < 1 {
		panic("invalid numGoroutines value")
	}

	dataLength := len(out[0])
	perGoroutineDataLength, numGoroutines := calculateParallelParams(dataLength, numGoroutines, 16, 16)
	if numGoroutines < 2 {
		applyMatrixSingle(m, in, out)
		return
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			start := i * perGoroutineDataLength
			end := start + perGoroutineDataLength
			if end > dataLength {
				end = dataLength
			}
			applyMatrixSlice(m, in, out, 0, len(out), start, end)
		}(i)
	}

	wg.Wait()
}
