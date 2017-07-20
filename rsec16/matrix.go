package rsec16

import "github.com/akalin/gopar/gf2p16"

func applyMatrix(m gf2p16.Matrix, in, out [][]uint16) {
	if len(in[0]) != len(out[0]) {
		panic("mismatched lengths")
	}

	// TODO: Optimize this.
	n := gf2p16.NewMatrixFromFunction(len(in), len(in[0]), func(i, j int) gf2p16.T {
		return gf2p16.T(in[i][j])
	})
	prod := m.Times(n)
	for i := 0; i < len(out); i++ {
		for j := 0; j < len(out[0]); j++ {
			out[i][j] = uint16(prod.At(i, j))
		}
	}
}
