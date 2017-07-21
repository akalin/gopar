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

func newCauchyParityMatrix(dataShards, parityShards int) gf2p16.Matrix {
	return newCauchyMatrix(parityShards, dataShards, func(i int) gf2p16.T {
		return gf2p16.T(dataShards + i)
	}, func(i int) gf2p16.T {
		return gf2p16.T(i)
	})
}

// NewCoderCauchy returns a Coder that works with the given number of
// data and parity shards, using a Cauchy matrix.
func NewCoderCauchy(dataShards, parityShards int) (Coder, error) {
	if dataShards <= 0 {
		panic("invalid data shard count")
	}
	if parityShards <= 0 {
		panic("invalid parity shard count")
	}

	if dataShards+parityShards > math.MaxUint16 {
		return Coder{}, errors.New("too many shards")
	}

	parityMatrix := newCauchyParityMatrix(dataShards, parityShards)
	return Coder{dataShards, parityShards, parityMatrix}, nil
}

var generators []gf2p16.T

func init() {
	// TODO: Generate this table at compile time.
	for i := 0; i < (1 << 16); i++ {
		if i%3 == 0 || i%5 == 0 || i%17 == 0 || i%257 == 0 {
			continue
		}
		g := gf2p16.T(2).Pow(uint32(i))
		generators = append(generators, g)
	}
}

func newVandermondeParityMatrix(dataShards, parityShards int) gf2p16.Matrix {
	return newVandermondeMatrix(parityShards, dataShards, func(i int) gf2p16.T {
		return generators[i]
	})
}

// NewCoderPAR2Vandermonde returns a Coder that works with the given
// number of data and parity shards, using a Vandermonde matrix as
// specified in the PAR2 spec. Note that this matrix is flawed, so
// ReconstructData may fail.
func NewCoderPAR2Vandermonde(dataShards, parityShards int) (Coder, error) {
	// The PAR2 encoding matrix looks like:
	//
	// 1       0       0        ... 0             0             0
	// 0       1       0        ... 0             0             0
	// 0       0       1        ... 0             0             0
	// ...
	// 0       0       0        ... 1             0             0
	// 0       0       0        ... 0             1             0
	// 0       0       0        ... 0             0             1
	//
	// 1       1       1        ... 1             1             1
	// 2       4       16       ... g_{m-2}       g_{m-1}       g_m
	// 2^2     4^2     16^2     ... g_{m-2}^2     g_{m-1}^2     g_m^2
	// ...
	// 2^(n-2) 4^(n-2) 16^(n-2) ... g_{m-2}^(n-2) g_{m-1}^(n-2) g_m^(n-2)
	// 2^(n-1) 4^(n-1) 16^(n-1) ... g_{m-2}^(n-1) g_{m-1}^(n-1) g_m^(n-1)
	//
	// where m is dataShards, n is parityShards, and g_m is the
	// mth smallest generator (element of order 65535) of
	// GF(65536). The top matrix is the m x m identity matrix, and
	// the bottom matrix is a n x m Vandermonde matrix. The rows
	// of the Vandermonde matrix repeat exactly when n > 65535,
	// and the columns repeat exactly when m > 32768. This means
	// that we can have at most 32768 data shards and 65535 parity
	// shards, since the PAR2 spec doesn't specify any lower
	// limits.
	//
	// Note that submatrices of the PAR2 encoding matrix may be
	// singular even when m <= 32768 and n <= 65535, due to a flaw
	// in the above construction.
	if dataShards <= 0 {
		panic("invalid data shard count")
	}
	if parityShards <= 0 {
		panic("invalid parity shard count")
	}

	if dataShards > len(generators) {
		return Coder{}, errors.New("too many data shards")
	}

	if parityShards > (1<<16)-1 {
		return Coder{}, errors.New("too many parity shards")
	}

	parityMatrix := newVandermondeParityMatrix(dataShards, parityShards)
	return Coder{dataShards, parityShards, parityMatrix}, nil
}

// GenerateParity takes a list of data shards, which must have length
// matching the dataShards value passed into NewCoder, and which must
// have equal-sized byte slices with even length, and returns a list
// of parityShards parity shards.
func (c Coder) GenerateParity(data [][]byte) [][]byte {
	parity := make([][]byte, c.parityShards)
	for i := range parity {
		parity[i] = make([]byte, len(data[0]))
	}
	applyMatrixSingle(c.parityMatrix, data, parity)
	return parity
}

func makeReconstructionMatrix(dataShards int, availableRows, missingRows, usedParityRows []int, parityMatrix gf2p16.Matrix) (gf2p16.Matrix, error) {
	m := gf2p16.NewMatrixFromFunction(len(usedParityRows), len(usedParityRows), func(i, j int) gf2p16.T {
		k := usedParityRows[i]
		l := missingRows[j]
		return parityMatrix.At(k, l)
	})
	n := gf2p16.NewMatrixFromFunction(len(usedParityRows), dataShards, func(i, j int) gf2p16.T {
		if j < len(availableRows) {
			k := usedParityRows[i]
			l := availableRows[j]
			return parityMatrix.At(k, l)
		}
		if i == j-len(availableRows) {
			return 1
		}
		return 0
	})
	return m.RowReduceForInverse(n)
}

// ReconstructData takes a list of data shards and parity shards, some
// of which may be nil, and tries to reconstruct the missing data
// shards. If successful, the nil rows of data are filled in and a nil
// error is returned. Otherwise, an error is returned.
func (c Coder) ReconstructData(data, parity [][]byte) error {
	var availableRows, missingRows []int
	var input [][]byte
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

	reconstructionMatrix, err := makeReconstructionMatrix(c.dataShards, availableRows, missingRows, usedParityRows, c.parityMatrix)
	if err != nil {
		return err
	}

	reconstructedData := make([][]byte, len(missingRows))
	for i := range reconstructedData {
		reconstructedData[i] = make([]byte, len(input[0]))
	}
	applyMatrixSingle(reconstructionMatrix, input, reconstructedData)
	for i, r := range missingRows {
		data[r] = reconstructedData[i]
	}
	return nil
}
