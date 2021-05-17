package par2

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/akalin/gopar/memfs"
	"github.com/stretchr/testify/require"
)

func testRepair(t *testing.T, workingDir string, options RepairOptions) {
	fs := makeDecoderMemFS(workingDir)
	r04Path := filepath.Join("dir4", "dir5", "file.r04")
	parityShardCount := 3

	buildPAR2Data(t, fs, workingDir, 4, parityShardCount)

	parPath := filepath.Join(workingDir, "file.par2")

	result, err := repair(testFileIO{t, fs}, parPath, options)
	require.NoError(t, err)
	require.Equal(t, RepairResult{}, result)

	fileData5, err := fs.ReadFile(r04Path)
	require.NoError(t, err)
	fileData5[len(fileData5)-1]++
	result, err = repair(testFileIO{t, fs}, parPath, options)
	require.NoError(t, err)
	require.Equal(t, RepairResult{
		RepairedPaths: []string{filepath.Join(workingDir, r04Path)},
	}, result)
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
