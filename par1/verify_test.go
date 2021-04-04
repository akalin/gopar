package par1

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type testVerifyDelegate struct {
	testDecoderDelegate
}

func testVerify(t *testing.T, workingDir string, useAbsPath bool, options VerifyOptions) {
	fs := makeDecoderMemFS(workingDir)

	buildPARData(t, fs, 3)

	parPath := "file.par"
	if useAbsPath {
		parPath = filepath.Join(workingDir, parPath)
	}
	result, err := verify(testFileIO{t, fs}, parPath, options)
	require.NoError(t, err)
	require.Equal(t, VerifyResult{NeedsRepair: false}, result)

	fileData5, err := fs.ReadFile("file.r04")
	require.NoError(t, err)
	fileData5[len(fileData5)-1]++
	result, err = verify(testFileIO{t, fs}, parPath, options)
	expectedErr := errors.New("shard sizes do not match")
	require.Equal(t, expectedErr, err)
	require.Equal(t, VerifyResult{NeedsRepair: true}, result)
}

func TestVerify(t *testing.T) {
	runOnExampleWorkingDirs(t, func(t *testing.T, workingDir string, useAbsPath bool) {
		testVerify(t, workingDir, useAbsPath, VerifyOptions{
			VerifyDelegate: testVerifyDelegate{testDecoderDelegate{t}},
		})
	})
}

func TestVerifyDefaults(t *testing.T) {
	runOnExampleWorkingDirs(t, func(t *testing.T, workingDir string, useAbsPath bool) {
		testVerify(t, workingDir, useAbsPath, VerifyOptions{})
	})
}
