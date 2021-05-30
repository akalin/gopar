package par2

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/akalin/gopar/memfs"
	"github.com/akalin/gopar/testfs"
	"github.com/stretchr/testify/require"
)

func testVerify(t *testing.T, workingDir string, options VerifyOptions) {
	fs := makeDecoderMemFS(workingDir)
	r04Path := filepath.Join("dir4", "dir5", "file.r04")
	parityShardCount := 3

	dataShardCount := buildPAR2Data(t, fs, workingDir, 4, parityShardCount)

	parPath := filepath.Join(workingDir, "file.par2")

	result, err := verify(testfs.TestFS{T: t, FS: fs}, parPath, options)
	require.NoError(t, err)
	require.Equal(t, VerifyResult{
		ShardCounts: ShardCounts{
			UsableDataShardCount:   dataShardCount,
			UsableParityShardCount: parityShardCount,
		},
	}, result)

	fileData5, err := fs.ReadFile(r04Path)
	require.NoError(t, err)
	fileData5[len(fileData5)-1]++
	result, err = verify(testfs.TestFS{T: t, FS: fs}, parPath, options)
	require.NoError(t, err)
	require.Equal(t, VerifyResult{
		ShardCounts: ShardCounts{
			UsableDataShardCount:   dataShardCount - 1,
			UnusableDataShardCount: 1,
			UsableParityShardCount: parityShardCount,
		},
	}, result)
}

func TestVerify(t *testing.T) {
	root := memfs.RootDir()
	dir1 := filepath.Join(root, "dir1")
	dir2 := filepath.Join(dir1, "dir2")
	dir3 := filepath.Join(root, "dir3")
	dirs := []string{root, dir1, dir2, dir3}
	for _, workingDir := range dirs {
		workingDir := workingDir
		t.Run(fmt.Sprintf("workingDir=%s", workingDir), func(t *testing.T) {
			testVerify(t, workingDir, VerifyOptions{
				NumGoroutines: NumGoroutinesDefault(),
			})
		})
	}
}

func TestVerifyDefaults(t *testing.T) {
	root := memfs.RootDir()
	dir1 := filepath.Join(root, "dir1")
	dir2 := filepath.Join(dir1, "dir2")
	dir3 := filepath.Join(root, "dir3")
	dirs := []string{root, dir1, dir2, dir3}
	for _, workingDir := range dirs {
		workingDir := workingDir
		t.Run(fmt.Sprintf("workingDir=%s", workingDir), func(t *testing.T) {
			testVerify(t, workingDir, VerifyOptions{})
		})
	}
}
