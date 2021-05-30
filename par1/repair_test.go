package par1

import (
	"path/filepath"
	"testing"

	"github.com/akalin/gopar/testfs"
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
	result, err := repair(testfs.TestFS{T: t, FS: fs}, parPath, options)
	require.NoError(t, err)
	require.Equal(t, RepairResult{}, result)

	perturbFile(t, fs, "file.r04")
	result, err = repair(testfs.TestFS{T: t, FS: fs}, parPath, options)
	require.NoError(t, err)
	require.Equal(t, RepairResult{
		RepairedPaths: []string{r04Path},
	}, result)

	perturbFile(t, fs, "file.r04")
	perturbFile(t, fs, "file.r02")
	perturbFile(t, fs, "file.r01")
	perturbFile(t, fs, "file.rar")
	result, err = repair(testfs.TestFS{T: t, FS: fs}, parPath, options)
	require.True(t, RepairErrorMeansRepairNecessaryButNotPossible(err))
	require.Equal(t, RepairResult{}, result)
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
