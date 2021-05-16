package par1

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type testRepairDelegate struct {
	testDecoderDelegate
}

func testRepair(t *testing.T, workingDir string, useAbsPath bool, options RepairOptions) {
	fs := makeDecoderMemFS(workingDir)

	buildPARData(t, fs, 3)

	parPath := "file.par"
	r04Path := "file.r04"
	if useAbsPath {
		parPath = filepath.Join(workingDir, parPath)
		r04Path = filepath.Join(workingDir, r04Path)
	}
	result, err := repair(testFileIO{t, fs}, parPath, options)
	require.NoError(t, err)
	require.Equal(t, RepairResult{}, result)

	fileData5, err := fs.ReadFile("file.r04")
	require.NoError(t, err)
	fileData5[len(fileData5)-1]++
	result, err = repair(testFileIO{t, fs}, parPath, options)
	require.NoError(t, err)
	require.Equal(t, RepairResult{
		RepairedPaths: []string{r04Path},
	}, result)
}

func TestRepair(t *testing.T) {
	runOnExampleWorkingDirs(t, func(t *testing.T, workingDir string, useAbsPath bool) {
		testRepair(t, workingDir, useAbsPath, RepairOptions{
			RepairDelegate: testRepairDelegate{testDecoderDelegate{t}},
		})
	})
}

func TestRepairDefaults(t *testing.T) {
	runOnExampleWorkingDirs(t, func(t *testing.T, workingDir string, useAbsPath bool) {
		testRepair(t, workingDir, useAbsPath, RepairOptions{})
	})
}
