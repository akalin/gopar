package par1

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func testCreate(t *testing.T, workingDir string, useAbsPath bool) {
	fs := makeEncoderMemFS(workingDir)

	paths := fs.Paths()

	parPath := "file.par"
	err := create(testFileIO{t, fs}, parPath, paths, CreateOptions{
		NumParityFiles:  NumParityFilesDefault,
		EncoderDelegate: testEncoderDelegate{t},
	})
	require.NoError(t, err)

	for _, path := range paths {
		require.NoError(t, fs.MoveFile(path, filepath.Base(path)))
	}

	decoder, err := newDecoder(testFileIO{t, fs}, testDecoderDelegate{t}, parPath)
	require.NoError(t, err)

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	needsRepair, err := decoder.Verify()
	require.NoError(t, err)
	require.False(t, needsRepair)
}

func TestCreate(t *testing.T) {
	runOnExampleWorkingDirs(t, testCreate)
}
