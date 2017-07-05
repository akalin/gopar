package gf2p16

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMatrix(t *testing.T) {
	m := NewZeroMatrix(2, 3)
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			require.Equal(t, T(0), m.At(i, j))
		}
	}

	m = NewMatrixFromSlice(2, 3, []T{0, 1, 2, 1, 2, 3})
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			require.Equal(t, T(i+j), m.At(i, j))
		}
	}

	m = NewMatrixFromFunction(2, 3, func(i, j int) T {
		return T(i + j)
	})
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			require.Equal(t, T(i+j), m.At(i, j))
		}
	}
}

func TestMatrixTimes(t *testing.T) {
	m := NewMatrixFromSlice(1, 2, []T{
		1,
		2,
	})
	n := NewMatrixFromSlice(2, 3, []T{
		1, 2, 3,
		2, 3, 4,
	})

	expectedProd := NewMatrixFromSlice(1, 3, []T{
		5, 4, 11,
	})

	prod := m.Times(n)
	require.Equal(t, expectedProd, prod)
}

func TestMatrixSwapRows(t *testing.T) {
	m := NewMatrixFromSlice(3, 2, []T{
		1, 2,
		2, 3,
		3, 4,
	})

	expectedM := m.clone()

	for i := 0; i < 3; i++ {
		m.swapRows(i, i)
		require.Equal(t, expectedM, m)
	}

	expectedM = NewMatrixFromSlice(3, 2, []T{
		1, 2,
		3, 4,
		2, 3,
	})

	m.swapRows(1, 2)
	require.Equal(t, expectedM, m)
}

func TestMatrixScaleRow(t *testing.T) {
	m := NewMatrixFromSlice(3, 2, []T{
		1, 2,
		2, 3,
		3, 4,
	})

	expectedM := NewMatrixFromSlice(3, 2, []T{
		1, 2,
		4, 6,
		3, 4,
	})

	m.scaleRow(1, T(2))
	require.Equal(t, expectedM, m)
}

func TestMatrixAddScaledRow(t *testing.T) {
	m := NewMatrixFromSlice(3, 2, []T{
		1, 2,
		2, 3,
		3, 4,
	})

	expectedM := NewMatrixFromSlice(3, 2, []T{
		1, 2,
		4, 11,
		3, 4,
	})

	m.addScaledRow(1, 2, T(2))
	require.Equal(t, expectedM, m)
}

func TestMatrixAugmentRight(t *testing.T) {
	m := NewMatrixFromSlice(3, 2, []T{
		1, 2,
		2, 3,
		3, 4,
	})

	expectedN := NewMatrixFromSlice(3, 5, []T{
		1, 2, 1, 0, 0,
		2, 3, 0, 1, 0,
		3, 4, 0, 0, 1,
	})

	n := m.augmentRight(NewIdentityMatrix(3))
	require.Equal(t, expectedN, n)
}

func TestMatrixColumnSlice(t *testing.T) {
	m := NewMatrixFromSlice(3, 5, []T{
		1, 2, 3, 4, 5,
		2, 3, 4, 5, 6,
		3, 4, 5, 6, 7,
	})

	expectedN := NewMatrixFromSlice(3, 2, []T{
		2, 3,
		3, 4,
		4, 5,
	})

	n := m.columnSlice(1, 3)
	require.Equal(t, expectedN, n)
}

func TestMatrixInverseIdentity(t *testing.T) {
	for i := 1; i < 100; i++ {
		m := NewIdentityMatrix(i)
		mInv, err := m.Inverse()
		require.NoError(t, err)
		require.Equal(t, m, mInv)
	}
}

func TestMatrixInverse(t *testing.T) {
	m := NewMatrixFromSlice(3, 3, []T{
		1, 2, 3,
		4, 5, 6,
		7, 8, 9,
	})
	n, err := m.Inverse()
	require.NoError(t, err)
	I := NewIdentityMatrix(3)
	require.Equal(t, I, m.Times(n))
	require.Equal(t, I, n.Times(m))
}

func TestMatrixInverseSingular(t *testing.T) {
	m := NewMatrixFromSlice(3, 3, []T{
		1, 2, 3,
		4, 5, 6,
		5, 7, 5,
	})
	_, err := m.Inverse()
	require.Equal(t, errors.New("singular matrix"), err)
}
