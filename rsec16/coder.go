package rsec16

import (
	"errors"
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

// ReconstructData takes a list of data shards and parity shards, some
// of which may be nil, and tries to reconstruct the missing data
// shards. If successful, the nil rows of data are filled in and a nil
// error is returned. Otherwise, an error is returned.
func (c Coder) ReconstructData(data, parity [][]uint16) error {
	var availableRows, missingRows []int
	var input [][]uint16
	for i, dataShard := range data {
		if dataShard != nil {
			availableRows = append(availableRows, i)
			input = append(input, dataShard)
		} else {
			missingRows = append(missingRows, i)
		}
	}

	if len(missingRows) == 0 {
		// Nothing to reconstruct.
		return nil
	}

	var usedParityRows []int
	for i := 0; i < len(parity) && len(input) < c.dataShards; i++ {
		if parity[i] != nil {
			usedParityRows = append(usedParityRows, i)
			input = append(input, parity[i])
		}
	}

	if len(input) < c.dataShards {
		return errors.New("not enough parity shards")
	}

	m := gf2p16.NewMatrixFromFunction(c.dataShards, c.dataShards, func(i, j int) gf2p16.T {
		if i < len(availableRows) {
			k := availableRows[i]
			// Take the kth row of the c.dataShards x
			// c.dataShards identity matrix.
			if j == k {
				return 1
			}
			return 0
		}

		// Take the rest of the rows from the parity matrix
		// corresponding to the used parity shards.
		k := usedParityRows[i-len(availableRows)]
		return c.parityMatrix.At(k, j)
	})
	mInv, err := m.Inverse()
	if err != nil {
		return err
	}

	n := gf2p16.NewMatrixFromFunction(len(missingRows), c.dataShards, func(i, j int) gf2p16.T {
		return mInv.At(missingRows[i], j)
	})

	reconstructedData := make([][]uint16, len(missingRows))
	for i := range reconstructedData {
		reconstructedData[i] = make([]uint16, len(input[0]))
	}
	applyMatrix(n, input, reconstructedData)
	for i, r := range missingRows {
		data[r] = reconstructedData[i]
	}
	return nil
}
