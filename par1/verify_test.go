package par1

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type testVerifyDelegate struct {
	testDecoderDelegate
}

func testVerify(t *testing.T, workingDir string, useAbsPath bool, options VerifyOptions) {
	fs := makeDecoderMemFS(workingDir)
	dataFileCount := fs.FileCount()
	parityFileCount := 3

	buildPARData(t, fs, parityFileCount)

	parPath := "file.par"
	if useAbsPath {
		parPath = filepath.Join(workingDir, parPath)
	}
	result, err := verify(testFileIO{t, fs}, parPath, options)
	require.NoError(t, err)
	require.Equal(t, VerifyResult{
		FileCounts: FileCounts{
			UsableDataFileCount:   dataFileCount,
			UsableParityFileCount: parityFileCount,
		},
		AllDataOk: options.VerifyAllData,
	}, result)

	fileData5, err := fs.ReadFile("file.r04")
	require.NoError(t, err)
	fileData5[len(fileData5)-1]++
	result, err = verify(testFileIO{t, fs}, parPath, options)
	require.NoError(t, err)
	require.Equal(t, VerifyResult{
		FileCounts: FileCounts{
			UsableDataFileCount:   dataFileCount - 1,
			UnusableDataFileCount: 1,
			UsableParityFileCount: parityFileCount,
		},
		AllDataOk: false,
	}, result)
}

func TestVerify(t *testing.T) {
	runOnExampleWorkingDirs(t, func(t *testing.T, workingDir string, useAbsPath bool) {
		testVerify(t, workingDir, useAbsPath, VerifyOptions{
			VerifyAllData:  true,
			VerifyDelegate: testVerifyDelegate{testDecoderDelegate{t}},
		})
	})
}

func TestVerifyDefaults(t *testing.T) {
	runOnExampleWorkingDirs(t, func(t *testing.T, workingDir string, useAbsPath bool) {
		testVerify(t, workingDir, useAbsPath, VerifyOptions{})
	})
}
