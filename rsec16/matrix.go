package rsec16

import "github.com/akalin/gopar/gf2p16"

func applyMatrix(m gf2p16.Matrix, in, out [][]uint16) {
	if len(in[0]) != len(out[0]) {
		panic("mismatched lengths")
	}

	// TODO: Maybe iterate over input slices first.
	for i, outSlice := range out {
		c := m.At(i, 0)
		gf2p16.MulSlice(c, in[0], outSlice)
		for j := 1; j < len(in); j++ {
			c := m.At(i, j)
			gf2p16.MulAndAddSlice(c, in[j], outSlice)
		}
	}
}
