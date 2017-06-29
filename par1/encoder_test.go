package par1

import (
	"testing"

	"github.com/klauspost/reedsolomon"
	"github.com/stretchr/testify/require"
)

type testEncoderDelegate struct {
	t *testing.T
}

func (d testEncoderDelegate) OnDataFileLoad(path string, err error) {
	d.t.Logf("OnDataFileLoad(%s, %v)", path, err)
}

func (d testEncoderDelegate) OnVolumeFileWrite(path string, err error) {
	d.t.Logf("OnVolumeFileWrite(%s, %v)", path, err)
}

func TestEncodeParity(t *testing.T) {
	io := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3},
			"file.r01": {0x5, 0x6, 0x7, 0x8},
			"file.r02": {0x9, 0xa, 0xb, 0xc},
			"file.r03": {0xd, 0xe},
			"file.r04": nil,
		},
	}

	paths := []string{"file.rar", "file.r01", "file.r02", "file.r03", "file.r04"}

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
		shards = append(shards, append(io.fileData[path], make([]byte, 4-len(io.fileData[path]))...))
	}

	shards = append(shards, encoder.parityData...)

	ok, err := rs.Verify(shards)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestWriteParity(t *testing.T) {
	io := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3},
			"file.r01": {0x5, 0x6, 0x7, 0x8},
			"file.r02": {0x9, 0xa, 0xb, 0xc},
			"file.r03": {0xd, 0xe},
			"file.r04": nil,
		},
	}

	paths := []string{"file.rar", "file.r01", "file.r02", "file.r03", "file.r04"}

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

	ok, err := decoder.Verify()
	require.NoError(t, err)
	require.True(t, ok)
}
