package rsec16

import (
	"math"

	"github.com/akalin/gopar/gf2p16"
)

// A Coder is an object that can generate parity shards, verify parity
// shards, and reconstruct data shards from parity shards.
type Coder struct {
	dataShards, parityShards int
	parityMatrix             gf2p16.Matrix
}

// NewCoder returns a Coder that works with the given number of data
// and parity shards.
func NewCoder(dataShards, parityShards int) Coder {
	if dataShards <= 0 {
		panic("invalid data shard count")
	}
	if parityShards <= 0 {
		panic("invalid parity shard count")
	}

	// TODO: Return an error instead.
	if dataShards+parityShards > math.MaxUint16 {
		panic("too many shards")
	}

	parityMatrix := newCauchyMatrix(parityShards, dataShards, func(i int) gf2p16.T {
		return gf2p16.T(dataShards + i)
	}, func(i int) gf2p16.T {
		return gf2p16.T(i)
	})
	return Coder{dataShards, parityShards, parityMatrix}
}

func applyMatrix(m gf2p16.Matrix, in, out [][]uint16) {
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

// GenerateParity takes a list of data shards, which must have length
// matching the dataShards value passed into NewCoder, and which must
// have equal-sized uint16 slices, and returns a list of parityShards
// parity shards.
func (c Coder) GenerateParity(data [][]uint16) [][]uint16 {
	parity := make([][]uint16, c.parityShards)
	for i := range parity {
		parity[i] = make([]uint16, len(data[0]))
	}
	applyMatrix(c.parityMatrix, data, parity)
	return parity
}
