package par2

import (
	"runtime"
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

func (d testEncoderDelegate) OnIndexFileWrite(path string, byteCount int, err error) {
	d.t.Logf("OnIndexFileWrite(%s, %d, %v)", path, byteCount, err)
}

func (d testEncoderDelegate) OnRecoveryFileWrite(start, count, total int, path string, dataByteCount, byteCount int, err error) {
	d.t.Logf("OnRecoveryFileWrite(start=%d, count=%d, total=%d, %s, dataByteCount=%d, byteCount=%d, %v)", start, count, total, path, dataByteCount, byteCount, err)
}

func newEncoderForTest(t *testing.T, io testFileIO, paths []string, sliceByteCount, parityShardCount int) (*Encoder, error) {
	return newEncoder(io, testEncoderDelegate{t}, paths, sliceByteCount, parityShardCount, runtime.GOMAXPROCS(0))
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
	encoder, err := newEncoderForTest(t, io, paths, sliceByteCount, parityShardCount)
	require.NoError(t, err)

	err = encoder.LoadFileData()
	require.NoError(t, err)

	err = encoder.ComputeParityData()
	require.NoError(t, err)

	var recoverySet []fileID
	dataShardsByID := make(map[fileID][][]byte)
	for filename, data := range io.fileData {
		fileID, _, _, fileDataShards := computeDataFileInfo(sliceByteCount, filename, data)
		recoverySet = append(recoverySet, fileID)
		dataShardsByID[fileID] = fileDataShards
	}

	sort.Slice(recoverySet, func(i, j int) bool {
		return fileIDLess(recoverySet[i], recoverySet[j])
	})

	var dataShards [][]byte
	for _, fileID := range recoverySet {
		dataShards = append(dataShards, dataShardsByID[fileID]...)
	}

	coder, err := rsec16.NewCoderPAR2Vandermonde(len(dataShards), parityShardCount, runtime.GOMAXPROCS(0))
	require.NoError(t, err)

	computedParityShards := coder.GenerateParity(dataShards)
	require.Equal(t, computedParityShards, encoder.parityShards)
}

func TestWriteParity(t *testing.T) {
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
	parityShardCount := 100
	encoder, err := newEncoderForTest(t, io, paths, sliceByteCount, parityShardCount)
	require.NoError(t, err)

	err = encoder.LoadFileData()
	require.NoError(t, err)

	err = encoder.ComputeParityData()
	require.NoError(t, err)

	err = encoder.Write("parity.par2")
	require.NoError(t, err)

	decoder, err := newDecoderForTest(t, io, "parity.par2")
	require.NoError(t, err)

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	ok, err := decoder.Verify(true)
	require.NoError(t, err)
	require.True(t, ok)
}
