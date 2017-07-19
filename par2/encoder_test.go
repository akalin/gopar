package par2

import (
	"sort"
	"testing"

	"github.com/akalin/gopar/rsec16"
	"github.com/stretchr/testify/require"
)

type testEncoderDelegate struct {
	t *testing.T
}

func (d testEncoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {
	d.t.Logf("OnDataFileLoad(%d, %d, byteCount=%d, %s, %v)", i, n, byteCount, path, err)
}

func (d testEncoderDelegate) OnParityFileWrite(i, n int, path string, dataByteCount, byteCount int, err error) {
	d.t.Logf("OnParityFileWrite(%d, %d, %s, %v, dataByteCount=%d, byteCount=%d)", i, n, path, dataByteCount, byteCount, err)
}

func TestEncodeParity(t *testing.T) {
	io := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3},
			"file.r01": {0x5, 0x6, 0x7, 0x8},
			"file.r02": {0x9, 0xa, 0xb, 0xc},
			"file.r03": {0xd, 0xe},
			"file.r04": {0xf},
		},
	}

	paths := []string{"file.rar", "file.r01", "file.r02", "file.r03", "file.r04"}

	sliceByteCount := 4
	parityShardCount := 3
	encoder, err := newEncoder(io, testEncoderDelegate{t}, paths, sliceByteCount, parityShardCount)
	require.NoError(t, err)

	err = encoder.LoadFileData()
	require.NoError(t, err)

	err = encoder.ComputeParityData()
	require.NoError(t, err)

	var recoverySet []fileID
	dataShardsByID := make(map[fileID][][]uint16)
	for filename, data := range io.fileData {
		fileID, _, _, fileDataShards := computeDataFileInfo(sliceByteCount, filename, data)
		recoverySet = append(recoverySet, fileID)
		dataShardsByID[fileID] = fileDataShards
	}

	sort.Slice(recoverySet, func(i, j int) bool {
		return fileIDLess(recoverySet[i], recoverySet[j])
	})

	var dataShards [][]uint16
	for _, fileID := range recoverySet {
		dataShards = append(dataShards, dataShardsByID[fileID]...)
	}

	coder, err := rsec16.NewCoderPAR2Vandermonde(len(dataShards), parityShardCount)
	require.NoError(t, err)

	computedParityShards := coder.GenerateParity(dataShards)
	require.Equal(t, computedParityShards, encoder.parityShards)
}
