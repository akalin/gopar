package par2

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/akalin/gopar/memfs"
	"github.com/akalin/gopar/testfs"
	"github.com/stretchr/testify/require"
)

func testRepair(t *testing.T, workingDir string, options RepairOptions) {
	fs := makeDecoderMemFS(workingDir)
	rarPath := "file.rar"
	r01Path := filepath.Join("dir1", "file.r01")
	r04Path := filepath.Join("dir4", "dir5", "file.r04")

	parityShardCount := 2

	buildPAR2Data(t, fs, workingDir, 4, parityShardCount)

	parPath := filepath.Join(workingDir, "file.par2")

	result, err := repair(testfs.MakeTestFS(t, fs), parPath, options)
	require.NoError(t, err)
	require.Equal(t, RepairResult{}, result)

	perturbFile(t, fs, r04Path)
	result, err = repair(testfs.MakeTestFS(t, fs), parPath, options)
	require.NoError(t, err)
	require.Equal(t, RepairResult{
		RepairedPaths: []string{filepath.Join(workingDir, r04Path)},
	}, result)

	perturbFile(t, fs, rarPath)
	perturbFile(t, fs, r01Path)
	perturbFile(t, fs, r04Path)
	result, err = repair(testfs.MakeTestFS(t, fs), parPath, options)
	require.True(t, RepairErrorMeansRepairNecessaryButNotPossible(err))
	require.Equal(t, RepairResult{}, result)
}

func TestRepair(t *testing.T) {
	root := memfs.RootDir()
	dir1 := filepath.Join(root, "dir1")
	dir2 := filepath.Join(dir1, "dir2")
	dir3 := filepath.Join(root, "dir3")
	dirs := []string{root, dir1, dir2, dir3}
	for _, workingDir := range dirs {
		workingDir := workingDir
		t.Run(fmt.Sprintf("workingDir=%s", workingDir), func(t *testing.T) {
			testRepair(t, workingDir, RepairOptions{
				NumGoroutines: NumGoroutinesDefault(),
			})
		})
	}
}

func TestRepairDefaults(t *testing.T) {
	root := memfs.RootDir()
	dir1 := filepath.Join(root, "dir1")
	dir2 := filepath.Join(dir1, "dir2")
	dir3 := filepath.Join(root, "dir3")
	dirs := []string{root, dir1, dir2, dir3}
	for _, workingDir := range dirs {
		workingDir := workingDir
		t.Run(fmt.Sprintf("workingDir=%s", workingDir), func(t *testing.T) {
			testRepair(t, workingDir, RepairOptions{})
		})
	}
}
