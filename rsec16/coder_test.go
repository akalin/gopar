package rsec16

import (
	"errors"
	"fmt"
	"testing"

	"github.com/akalin/gopar/gf2p16"
	"github.com/stretchr/testify/require"
)

func newCoderCauchy(dataShards, parityShards int) (Coder, error) {
	return NewCoderCauchy(dataShards, parityShards, DefaultNumGoroutines())
}

func newCoderPAR2Vandermonde(dataShards, parityShards int) (Coder, error) {
	return NewCoderPAR2Vandermonde(dataShards, parityShards, DefaultNumGoroutines())
}

func TestCoderCauchyNewCoderError(t *testing.T) {
	// Ideally, we'd test that NewCoder(32768, 32767) succeeds,
	// but doing so takes 15 seconds!
	_, err := newCoderCauchy(32768, 32768)
	require.Equal(t, errors.New("too many shards"), err)
}

func TestGenerators(t *testing.T) {
	require.Equal(t, 32768, len(generators))
	for i := 0; i < len(generators); i += 1000 {
		g := generators[i]
		for i := uint32(1); i < 65535; i++ {
			require.NotEqual(t, gf2p16.T(1), g.Pow(i))
		}
		require.Equal(t, gf2p16.T(1), g.Pow(65535))
	}
}

func TestCoderPAR2VandermondeNewCoderError(t *testing.T) {
	// Ideally, we'd test that NewCoder(32768, 65535) succeeds,
	// but doing so would probably take even longer than 15
	// seconds.
	_, err := newCoderPAR2Vandermonde(32769, 65535)
	require.Equal(t, errors.New("too many data shards"), err)

	_, err = newCoderPAR2Vandermonde(32768, 65536)
	require.Equal(t, errors.New("too many parity shards"), err)
}

func makeReconstructionMatrixNaive(dataShards int, availableRows, missingRows, usedParityRows []int, parityMatrix gf2p16.Matrix) (gf2p16.Matrix, error) {
	m := gf2p16.NewMatrixFromFunction(dataShards, dataShards, func(i, j int) gf2p16.T {
		if i < len(availableRows) {
			k := availableRows[i]
			// Take the kth row of the c.dataShards x
			// c.dataShards identity matrix.
			if j == k {
				return 1
			}
			return 0
		}

		// Take the rest of the rows from the parity matrix
		// corresponding to the used parity shards.
		k := usedParityRows[i-len(availableRows)]
		return parityMatrix.At(k, j)
	})
	mInv, err := m.Inverse()
	if err != nil {
		return gf2p16.Matrix{}, err
	}

	return gf2p16.NewMatrixFromFunction(len(missingRows), dataShards, func(i, j int) gf2p16.T {
		return mInv.At(missingRows[i], j)
	}), nil
}

func testMakeReconstructionMatrix(t *testing.T, newParityMatrixFn func(int, int) gf2p16.Matrix) {
	dataShards := 6
	parityShards := 11
	availableRows := []int{0, 1, 2, 4}
	missingRows := []int{3, 5}
	usedParityRows := []int{9, 10}
	parityMatrix := newParityMatrixFn(dataShards, parityShards)

	expectedReconstructionMatrix, err := makeReconstructionMatrixNaive(dataShards, availableRows, missingRows, usedParityRows, parityMatrix)
	require.NoError(t, err)

	reconstructionMatrix, err := makeReconstructionMatrix(dataShards, availableRows, missingRows, usedParityRows, parityMatrix)
	require.NoError(t, err)

	require.Equal(t, expectedReconstructionMatrix, reconstructionMatrix)
}

func TestMakeReconstructionMatrix(t *testing.T) {
	t.Run("Cauchy", func(t *testing.T) {
		testMakeReconstructionMatrix(t, newCauchyParityMatrix)
	})
	t.Run("PAR2Vandermonde", func(t *testing.T) {
		testMakeReconstructionMatrix(t, newVandermondeParityMatrix)
	})
}

func benchmarkMakeReconstructionMatrix(b *testing.B, newParityMatrixFn func(int, int) gf2p16.Matrix, dataShards, parityShards int) {
	missingRows := make([]int, parityShards)
	usedParityRows := make([]int, parityShards)
	for i := range usedParityRows {
		missingRows[i] = i
		usedParityRows[i] = i
	}
	availableRows := make([]int, dataShards-parityShards)
	for i := range availableRows {
		availableRows[i] = parityShards + i
	}
	parityMatrix := newVandermondeParityMatrix(dataShards, parityShards)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := makeReconstructionMatrix(dataShards, availableRows, missingRows, usedParityRows, parityMatrix)
		require.NoError(b, err)
	}
}

func benchmarkParityMatrix(b *testing.B, benchmarkFn func(*testing.B, func(int, int) gf2p16.Matrix)) {
	b.Run("Cauchy", func(b *testing.B) {
		benchmarkFn(b, newCauchyParityMatrix)
	})
	// Shorten name so that the benchmark results line up.
	b.Run("PAR2Vm", func(b *testing.B) {
		benchmarkFn(b, newVandermondeParityMatrix)
	})
}

func BenchmarkMakeReconstructionMatrix(b *testing.B) {
	for _, config := range []struct {
		dataShards   int
		parityShards int
	}{{100, 10}, {1000, 10}, {3000, 10}} {
		b.Run(fmt.Sprintf("%dx%d", config.dataShards, config.parityShards), func(b *testing.B) {
			benchmarkParityMatrix(b, func(b *testing.B, newParityMatrixFn func(int, int) gf2p16.Matrix) {
				benchmarkMakeReconstructionMatrix(b, newParityMatrixFn, config.dataShards, config.parityShards)
			})
		})
	}
}

func testCoder(t *testing.T, testFn func(*testing.T, func(int, int) (Coder, error))) {
	t.Run("Cauchy", func(t *testing.T) {
		testFn(t, newCoderCauchy)
	})
	t.Run("PAR2Vandermonde", func(t *testing.T) {
		testFn(t, newCoderPAR2Vandermonde)
	})
}

func makeTestData() [][]byte {
	return [][]byte{
		{0x0, 0x1, 0x0, 0x2},
		{0x0, 0x3, 0x0, 0x4},
		{0x0, 0x5, 0x0, 0x6},
		{0x0, 0x7, 0x0, 0x8},
		{0x0, 0x9, 0x0, 0xa},
	}
}

func testCoderGenerateParity(t *testing.T, newCoder func(int, int) (Coder, error)) {
	data := makeTestData()
	c, err := newCoder(5, 3)
	require.NoError(t, err)
	parity := c.GenerateParity(data)
	require.Equal(t, 3, len(parity))
	for _, row := range parity {
		require.Equal(t, 4, len(row))
	}
}

func TestCoderGenerateParity(t *testing.T) {
	testCoder(t, testCoderGenerateParity)
}

func testCoderReconstructData(t *testing.T, newCoder func(int, int) (Coder, error)) {
	data := makeTestData()
	c, err := newCoder(5, 3)
	require.NoError(t, err)
	parity := c.GenerateParity(data)

	corruptedData := [][]byte{
		nil,
		data[1],
		nil,
		data[3],
		nil,
	}
	err = c.ReconstructData(corruptedData, parity)
	require.NoError(t, err)
	require.Equal(t, data, corruptedData)
}

func TestCoderReconstructDataNotEnough(t *testing.T) {
	testCoder(t, testCoderReconstructData)
}

func testCoderReconstructDataMissingParity(t *testing.T, newCoder func(int, int) (Coder, error)) {
	data := makeTestData()
	c, err := newCoder(5, 3)
	require.NoError(t, err)
	parity := c.GenerateParity(data)

	corruptedData := [][]byte{
		data[0],
		nil,
		data[2],
		data[3],
		nil,
	}
	corruptedParity := [][]byte{
		nil,
		parity[1],
		parity[2],
	}

	err = c.ReconstructData(corruptedData, corruptedParity)
	require.NoError(t, err)
	require.Equal(t, data, corruptedData)
}

func TestCoderReconstructDataMissingParity(t *testing.T) {
	testCoder(t, testCoderReconstructDataMissingParity)
}

func testCoderReconstructDataNotEnough(t *testing.T, newCoder func(int, int) (Coder, error)) {
	data := makeTestData()
	c, err := newCoder(5, 3)
	require.NoError(t, err)
	parity := c.GenerateParity(data)

	corruptedData := [][]byte{
		data[0],
		nil,
		nil,
		nil,
		nil,
	}
	expectedErr := errors.New("not enough parity shards")
	err = c.ReconstructData(corruptedData, parity)
	require.Equal(t, expectedErr, err)

	corruptedData = [][]byte{
		data[0],
		data[1],
		nil,
		nil,
		nil,
	}
	corruptedParity := [][]byte{
		nil,
		parity[1],
		parity[2],
	}
	err = c.ReconstructData(corruptedData, corruptedParity)
	require.Equal(t, expectedErr, err)
}

func TestCoderReconstructData(t *testing.T) {
	testCoder(t, testCoderReconstructDataNotEnough)
}

func TestMatrixStats(t *testing.T) {
	dataShards := 10000
	parityShards := 10000
	missingRowCount := 100
	for k := 0; k < 100; k++ {
		dataRows := rand.Perm(dataShards)
		parityRows := rand.Perm(parityShards)
		availableRows := dataRows[:dataShards-missingRowCount]
		missingRows := dataRows[dataShards-missingRowCount:]
		usedParityRows := parityRows[:missingRowCount]
		parityMatrix := newVandermondeParityMatrix(dataShards, parityShards)
		r, err := makeReconstructionMatrix(dataShards, availableRows, missingRows, usedParityRows, parityMatrix)
		require.NoError(t, err)
		numZeros := 0
		numOnes := 0
		numOther := 0
		for i := 0; i < missingRowCount; i++ {
			for j := 0; j < missingRowCount; j++ {
				c := r.At(i, j)
				if c == 0 {
					numZeros++
				} else if c == 1 {
					numOnes++
				} else {
					numOther++
				}
			}
		}
		if numOther < missingRowCount*missingRowCount {
			t.Logf("k=%d #0s=%d, #1s=%d, #other=%d", k, numZeros, numOnes, numOther)
		}
	}
}

// TODO: Add tests demonstrating the flaws in the PAR2 Vandermonde matrix.
