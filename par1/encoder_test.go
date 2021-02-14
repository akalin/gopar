package par1

import (
	"testing"

	"github.com/klauspost/reedsolomon"
	"github.com/stretchr/testify/require"
)

type testEncoderDelegate struct {
	t *testing.T
}

func (d testEncoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {
	d.t.Helper()
	d.t.Logf("OnDataFileLoad(%d, %d, byteCount=%d, %s, %v)", i, n, byteCount, path, err)
}

func (d testEncoderDelegate) OnVolumeFileWrite(i, n int, path string, dataByteCount, byteCount int, err error) {
	d.t.Helper()
	d.t.Logf("OnVolumeFileWrite(%d, %d, %s, dataByteCount=%d, byteCount=%d, %v)", i, n, path, dataByteCount, byteCount, err)
}

func makeEncoderTestFileIO(t *testing.T) testFileIO {
	return testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3},
			"file.r01": {0x5, 0x6, 0x7, 0x8},
			"file.r02": {0x9, 0xa, 0xb, 0xc},
			"file.r03": {0xd, 0xe},
			"file.r04": nil,
		},
	}
}

func TestEncodeParity(t *testing.T) {
	io := makeEncoderTestFileIO(t)

	paths := io.paths()

	encoder, err := newEncoder(io, testEncoderDelegate{t}, paths, 3)
	require.NoError(t, err)

	err = encoder.LoadFileData()
	require.NoError(t, err)

	err = encoder.ComputeParityData()
	require.NoError(t, err)

	rs, err := reedsolomon.New(len(encoder.fileData), encoder.volumeCount, reedsolomon.WithPAR1Matrix())
	require.NoError(t, err)

	var shards [][]byte
	for _, path := range paths {
		data := io.getData(path)
		shards = append(shards, append(data, make([]byte, 4-len(data))...))
	}

	shards = append(shards, encoder.parityData...)

	ok, err := rs.Verify(shards)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestWriteParity(t *testing.T) {
	io := makeEncoderTestFileIO(t)

	paths := io.paths()

	encoder, err := newEncoder(io, testEncoderDelegate{t}, paths, 3)
	require.NoError(t, err)

	err = encoder.LoadFileData()
	require.NoError(t, err)

	err = encoder.ComputeParityData()
	require.NoError(t, err)

	err = encoder.Write("parity.par")
	require.NoError(t, err)

	decoder, err := newDecoder(io, testDecoderDelegate{t}, "parity.par")
	require.NoError(t, err)

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	needsRepair, err := decoder.Verify()
	require.NoError(t, err)
	require.False(t, needsRepair)
}
