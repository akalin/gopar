package par1

import (
	"errors"
	"path/filepath"
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

func makeEncoderTestFileIO(t *testing.T, workingDir string) testFileIO {
	return makeTestFileIO(t, workingDir, map[string][]byte{
		"file.rar":                                {0x1, 0x2, 0x3},
		filepath.Join("dir1", "file.r01"):         {0x5, 0x6, 0x7, 0x8},
		filepath.Join("dir2", "file.r02"):         {0x9, 0xa, 0xb, 0xc},
		filepath.Join("dir2", "dir3", "file.r03"): {0xd, 0xe},
		filepath.Join("dir4", "dir5", "file.r04"): nil,
	})
}

func TestEncodeParity(t *testing.T) {
	io := makeEncoderTestFileIO(t, rootDir())

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

func TestEncodeParityFilenameCollision(t *testing.T) {
	io := makeEncoderTestFileIO(t, rootDir())
	io.setData(filepath.Join("dir6", "file.rar"), []byte{0x5, 0x6})

	paths := io.paths()

	_, err := newEncoder(io, testEncoderDelegate{t}, paths, 3)
	require.Equal(t, errors.New("filename collision"), err)
}

func testWriteParity(t *testing.T, workingDir string, useAbsPath bool) {
	io := makeEncoderTestFileIO(t, workingDir)

	paths := io.paths()

	encoder, err := newEncoder(io, testEncoderDelegate{t}, paths, 3)
	require.NoError(t, err)

	err = encoder.LoadFileData()
	require.NoError(t, err)

	err = encoder.ComputeParityData()
	require.NoError(t, err)

	parPath := "file.par"
	if useAbsPath {
		parPath = filepath.Join(workingDir, parPath)
	}
	err = encoder.Write(parPath)
	require.NoError(t, err)

	for _, path := range paths {
		io.moveData(path, filepath.Base(path))
	}

	decoder, err := newDecoder(io, testDecoderDelegate{t}, parPath)
	require.NoError(t, err)

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	needsRepair, err := decoder.Verify()
	require.NoError(t, err)
	require.False(t, needsRepair)
}

func TestWriteParity(t *testing.T) {
	runOnExampleWorkingDirs(t, testWriteParity)
}
