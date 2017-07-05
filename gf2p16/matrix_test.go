package gf2p16

import (
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
