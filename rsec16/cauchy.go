package rsec16

import "github.com/akalin/gopar/gf2p16"

func newCauchyMatrix(rows, columns int, xFunc, yFunc func(int) gf2p16.T) gf2p16.Matrix {
	return gf2p16.NewMatrixFromFunction(rows, columns, func(i, j int) gf2p16.T {
		return xFunc(i).Plus(yFunc(j)).Inverse()
	})
}
