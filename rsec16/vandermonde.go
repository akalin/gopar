package rsec16

import "github.com/akalin/gopar/gf2p16"

// newVandermondeMatrix returns a matrix where a[i, j] =
// alpha(j)^i. (Note that this is the transpose of the matrix given by
// the Wikipedia article.)
func newVandermondeMatrix(rows, columns int, alphaColumnFunc func(int) gf2p16.T) gf2p16.Matrix {
	return gf2p16.NewMatrixFromFunction(rows, columns, func(i, j int) gf2p16.T {
		return alphaColumnFunc(j).Pow(uint32(i))
	})
}
