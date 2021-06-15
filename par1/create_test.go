package par1

import (
	"path/filepath"
	"testing"

	"github.com/akalin/gopar/testfs"
	"github.com/stretchr/testify/require"
)

type testCreateDelegate struct {
	testEncoderDelegate
}

func (d testCreateDelegate) OnFilesNotAllInSameDir() {
	d.t.Helper()
	d.t.Log("OnFilesNotAllInSameDir()")
}

func testCreate(t *testing.T, workingDir string, useAbsPath bool, options CreateOptions) {
	fs := makeEncoderMemFS(workingDir)

	paths := fs.Paths()

	parPath := "file.par"
	err := create(testfs.MakeTestFS(t, fs), parPath, paths, options)
	require.NoError(t, err)

	for _, path := range paths {
		require.NoError(t, fs.MoveFile(path, filepath.Base(path)))
	}

	decoder := newDecoderForTest(t, fs, parPath)
	defer closeCloser(t, decoder)

	err = decoder.LoadIndexFile()
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	ok, err := decoder.VerifyAllData()
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCreate(t *testing.T) {
	runOnExampleWorkingDirs(t, func(t *testing.T, workingDir string, useAbsPath bool) {
		testCreate(t, workingDir, useAbsPath, CreateOptions{
			NumParityFiles: NumParityFilesDefault,
			CreateDelegate: testCreateDelegate{testEncoderDelegate{t}},
		})
	})
}

func TestCreateDefaults(t *testing.T) {
	runOnExampleWorkingDirs(t, func(t *testing.T, workingDir string, useAbsPath bool) {
		testCreate(t, workingDir, useAbsPath, CreateOptions{})
	})
}
