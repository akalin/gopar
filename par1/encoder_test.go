package par1

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/akalin/gopar/memfs"
	"github.com/akalin/gopar/testfs"
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

func makeEncoderMemFS(workingDir string) memfs.MemFS {
	return memfs.MakeMemFS(workingDir, map[string][]byte{
		"file.rar":                                {0x1, 0x2, 0x3},
		filepath.Join("dir1", "file.r01"):         {0x5, 0x6, 0x7, 0x8},
		filepath.Join("dir2", "file.r02"):         {0x9, 0xa, 0xb, 0xc},
		filepath.Join("dir2", "dir3", "file.r03"): {0xd, 0xe},
		filepath.Join("dir4", "dir5", "file.r04"): nil,
	})
}

func newEncoderForTest(t *testing.T, fs memfs.MemFS, filePaths []string, volumeCount int) (*Encoder, error) {
	return newEncoder(testfs.MakeTestFS(t, fs), testEncoderDelegate{t}, filePaths, volumeCount)
}

func TestEncodeParity(t *testing.T) {
	fs := makeEncoderMemFS(memfs.RootDir())

	paths := fs.Paths()

	encoder, err := newEncoderForTest(t, fs, paths, 3)
	require.NoError(t, err)

	err = encoder.LoadFileData()
	require.NoError(t, err)

	err = encoder.ComputeParityData()
	require.NoError(t, err)

	rs, err := reedsolomon.New(len(encoder.fileData), encoder.volumeCount, reedsolomon.WithPAR1Matrix())
	require.NoError(t, err)

	var shards [][]byte
	for _, path := range paths {
		data, err := fs.ReadFile(path)
		require.NoError(t, err)
		shards = append(shards, append(data, make([]byte, 4-len(data))...))
	}

	shards = append(shards, encoder.parityData...)

	ok, err := rs.Verify(shards)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestEncodeParityFilenameCollision(t *testing.T) {
	fs := makeEncoderMemFS(memfs.RootDir())
	require.NoError(t, fs.WriteFile(filepath.Join("dir6", "file.rar"), []byte{0x5, 0x6}))

	paths := fs.Paths()

	_, err := newEncoderForTest(t, fs, paths, 3)
	require.Equal(t, errors.New("filename collision"), err)
}

func testWriteParity(t *testing.T, workingDir string, useAbsPath bool) {
	fs := makeEncoderMemFS(workingDir)

	paths := fs.Paths()

	encoder, err := newEncoderForTest(t, fs, paths, 3)
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
		require.NoError(t, fs.MoveFile(path, filepath.Base(path)))
	}

	decoder, err := newDecoder(testfs.MakeTestFS(t, fs), testDecoderDelegate{t}, parPath)
	require.NoError(t, err)

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	ok, err := decoder.VerifyAllData()
	require.NoError(t, err)
	require.True(t, ok)
}

func TestWriteParity(t *testing.T) {
	runOnExampleWorkingDirs(t, testWriteParity)
}
