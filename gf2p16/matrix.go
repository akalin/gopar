package gf2p16

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
