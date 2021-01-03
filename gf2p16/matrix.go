package gf2p16

import (
	"github.com/akalin/gopar/errorcode"
)

// Matrix is an immutable rectangular array of elements of
// GF(2^16). It has just enough methods to support Reed-Solomon
// erasure codes.
type Matrix struct {
	rows, columns int
	// Elements are stored in row-major order.
	elements []T
}

func checkRowColumnCount(rows, columns int) {
	if rows <= 0 {
		panic("invalid row count")
	}
	if columns <= 0 {
		panic("invalid column count")
	}
}

// NewZeroMatrix returns a rows x columns matrix with every element
// being zero.
func NewZeroMatrix(rows, columns int) Matrix {
	checkRowColumnCount(rows, columns)
	return Matrix{rows, columns, make([]T, rows*columns)}
}

// NewMatrixFromSlice returns a rows x columns matrix with elements
// taken from the given array in row-major order.
func NewMatrixFromSlice(rows, columns int, elements []T) Matrix {
	checkRowColumnCount(rows, columns)
	if len(elements) != rows*columns {
		panic("element count is not rows*columns")
	}
	elementsCopy := make([]T, len(elements))
	copy(elementsCopy, elements)
	return Matrix{rows, columns, elementsCopy}
}

// NewMatrixFromFunction returns a rows x columns matrix with elements
// filled in from the given function, which is passed the row index
// and the column index, and shouldn't rely on any particular call
// ordering.
func NewMatrixFromFunction(rows, columns int, fn func(int, int) T) Matrix {
	checkRowColumnCount(rows, columns)
	elements := make([]T, rows*columns)
	for i := 0; i < rows; i++ {
		for j := 0; j < columns; j++ {
			elements[i*columns+j] = fn(i, j)
		}
	}
	return NewMatrixFromSlice(rows, columns, elements)
}

// NewIdentityMatrix returns an n x n identity matrix.
func NewIdentityMatrix(n int) Matrix {
	return NewMatrixFromFunction(n, n, func(i, j int) T {
		if i == j {
			return 1
		}
		return 0
	})
}

func (m Matrix) checkRowIndex(i int) {
	if i < 0 || i >= m.rows {
		panic("row index out of bounds")
	}
}

func (m Matrix) checkColumnIndex(i int) {
	if i < 0 || i >= m.columns {
		panic("column index out of bounds")
	}
}

// At returns the element at row index i and column index j.
func (m Matrix) At(i, j int) T {
	m.checkRowIndex(i)
	m.checkColumnIndex(j)
	return m.elements[i*m.columns+j]
}

// Times returns the matrix product of m with n, which must have
// compatible dimensions.
func (m Matrix) Times(n Matrix) Matrix {
	if m.columns != n.rows {
		panic("mismatched dimensions")
	}

	return NewMatrixFromFunction(m.rows, n.columns, func(i, j int) T {
		var t T
		for k := 0; k < m.columns; k++ {
			mIK := m.At(i, k)
			nKJ := n.At(k, j)
			t ^= mIK.Times(nKJ)
		}
		return t
	})
}

// row returns a slice into m.elements, so caller must not mutate
// except for local temporary arrays.
func (m Matrix) row(i int) []T {
	m.checkRowIndex(i)
	return m.elements[i*m.columns : (i+1)*m.columns]
}

func (m Matrix) clone() Matrix {
	return NewMatrixFromSlice(m.rows, m.columns, m.elements)
}

// The mutating functions below must not be called except on local
// temporary arrays.

func (m Matrix) swapRows(i, j int) {
	m.checkRowIndex(i)
	m.checkRowIndex(j)

	if i == j {
		return
	}

	rowI := m.row(i)
	rowJ := m.row(j)
	for k := 0; k < m.columns; k++ {
		rowI[k], rowJ[k] = rowJ[k], rowI[k]
	}
}

func (m Matrix) scaleRow(i int, c T) {
	row := m.row(i)
	mulSlice(c, row, row)
}

func (m Matrix) addScaledRow(dest, src int, c T) {
	rowSrc := m.row(src)
	rowDest := m.row(dest)
	mulAndAddSlice(c, rowSrc, rowDest)
}

func (m Matrix) rowReduceForInverse(n Matrix) error {
	// Convert to row echelon form.
	for i := 0; i < m.rows; i++ {
		// Swap the ith row with the first row with a non-zero
		// ith column.
		var pivot T
		for j := i; j < m.rows; j++ {
			if m.At(j, i) != 0 {
				m.swapRows(i, j)
				n.swapRows(i, j)
				pivot = m.At(i, i)
				break
			}
		}
		if pivot == 0 {
			return errorcode.SingularMatrix
		}

		// Scale the ith row to have 1 as the pivot.
		pivotInv := pivot.Inverse()
		m.scaleRow(i, pivotInv)
		n.scaleRow(i, pivotInv)

		// Zero out all elements below m_ii.
		for j := i + 1; j < m.rows; j++ {
			t := m.At(j, i)
			if t != 0 {
				m.addScaledRow(j, i, t)
				n.addScaledRow(j, i, t)
			}
		}
	}

	// Then convert to reduced row echelon form.
	for i := 0; i < m.rows; i++ {
		// Zero out all elements above m_ii.
		for j := 0; j < i; j++ {
			t := m.At(j, i)
			if t != 0 {
				m.addScaledRow(j, i, t)
				n.addScaledRow(j, i, t)
			}
		}
	}

	return nil
}

// Inverse returns the matrix inverse of m, which must be square, or
// an error if it is singular.
func (m Matrix) Inverse() (Matrix, error) {
	if m.rows != m.columns {
		panic("cannot invert non-square matrix")
	}
	mInv := NewIdentityMatrix(m.columns)
	err := m.clone().rowReduceForInverse(mInv)
	if err != nil {
		return Matrix{}, err
	}

	return mInv, nil
}

// RowReduceForInverse runs row reduction on copies of m, which must
// be square, and n, which must have the same number of rows as m. The
// row-reduced copy of n is returned if m is non-singular; otherwise,
// an empty matrix and an error is returned.
//
// Note that if n is the identity matrix, then the inverse of m is
// returned. This is the special case of the following fact: if n is
// equal to some matrix ( n_L | I ), where I is the
// appropriately-sized identity matrix and | denotes horizontal
// augmentation, then if the returned matrix is n', the matrix
// ( I / n' ) is the inverse of ( I / n_L | 0 / m ), where / denotes
// vertical augmentation.
func (m Matrix) RowReduceForInverse(n Matrix) (Matrix, error) {
	if m.rows != m.columns {
		panic("cannot row-reduce non-square matrix")
	}
	if n.rows != m.rows {
		panic("n must have the same number of rows as m")
	}
	nReduced := n.clone()
	err := m.clone().rowReduceForInverse(nReduced)
	if err != nil {
		return Matrix{}, err
	}

	return nReduced, nil
}
