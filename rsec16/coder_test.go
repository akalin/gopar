package rsec16

import (
	"errors"
	"testing"

	"github.com/akalin/gopar/gf2p16"
	"github.com/stretchr/testify/require"
)

func TestCoderCauchyNewCoderError(t *testing.T) {
	// Ideally, we'd test that NewCoder(32768, 32767) succeeds,
	// but doing so takes 15 seconds!
	_, err := NewCoderCauchy(32768, 32768)
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
	_, err := NewCoderPAR2Vandermonde(32769, 65535)
	require.Equal(t, errors.New("too many data shards"), err)

	_, err = NewCoderPAR2Vandermonde(32768, 65536)
	require.Equal(t, errors.New("too many parity shards"), err)
}

func testCoder(t *testing.T, testFn func(*testing.T, func(int, int) (Coder, error))) {
	t.Run("Cauchy", func(t *testing.T) {
		testFn(t, NewCoderCauchy)
	})
	t.Run("PAR2Vandermonde", func(t *testing.T) {
		testFn(t, NewCoderPAR2Vandermonde)
	})
}

func testCoderGenerateParity(t *testing.T, newCoder func(int, int) (Coder, error)) {
	data := [][]uint16{
		{0x1, 0x2},
		{0x3, 0x4},
		{0x5, 0x6},
		{0x7, 0x8},
		{0x9, 0xa},
	}
	c, err := newCoder(5, 3)
	require.NoError(t, err)
	parity := c.GenerateParity(data)
	require.Equal(t, 3, len(parity))
	for _, row := range parity {
		require.Equal(t, 2, len(row))
	}
}

func TestCoderGenerateParity(t *testing.T) {
	testCoder(t, testCoderGenerateParity)
}

func testCoderReconstructData(t *testing.T, newCoder func(int, int) (Coder, error)) {
	data := [][]uint16{
		{0x1, 0x2},
		{0x3, 0x4},
		{0x5, 0x6},
		{0x7, 0x8},
		{0x9, 0xa},
	}
	c, err := newCoder(5, 3)
	require.NoError(t, err)
	parity := c.GenerateParity(data)

	corruptedData := [][]uint16{
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
	data := [][]uint16{
		{0x1, 0x2},
		{0x3, 0x4},
		{0x5, 0x6},
		{0x7, 0x8},
		{0x9, 0xa},
	}
	c, err := newCoder(5, 3)
	require.NoError(t, err)
	parity := c.GenerateParity(data)

	corruptedData := [][]uint16{
		data[0],
		nil,
		data[2],
		data[3],
		nil,
	}
	corruptedParity := [][]uint16{
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
	data := [][]uint16{
		{0x1, 0x2},
		{0x3, 0x4},
		{0x5, 0x6},
		{0x7, 0x8},
		{0x9, 0xa},
	}
	c, err := newCoder(5, 3)
	require.NoError(t, err)
	parity := c.GenerateParity(data)

	corruptedData := [][]uint16{
		data[0],
		nil,
		nil,
		nil,
		nil,
	}
	expectedErr := errors.New("not enough parity shards")
	err = c.ReconstructData(corruptedData, parity)
	require.Equal(t, expectedErr, err)

	corruptedData = [][]uint16{
		data[0],
		data[1],
		nil,
		nil,
		nil,
	}
	corruptedParity := [][]uint16{
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

// TODO: Add tests demonstrating the flaws in the PAR2 Vandermonde matrix.
