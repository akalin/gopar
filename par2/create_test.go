package par2

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/akalin/gopar/memfs"
	"github.com/stretchr/testify/require"
)

func testCreate(t *testing.T, workingDir string) {
	fs := makeEncoderMemFS(workingDir)

	paths := fs.Paths()

	parPath := filepath.Join(workingDir, "parity.par2")

	err := create(testFileIO{t, fs}, parPath, paths, CreateOptions{
		SliceByteCount:  4,
		NumParityShards: 100,
		NumGoroutines:   NumGoroutinesDefault(),
		CreateDelegate:  testEncoderDelegate{t},
	})
	require.NoError(t, err)

	decoder, err := newDecoderForTest(t, fs, parPath)
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
	root := memfs.RootDir()
	dir1 := filepath.Join(root, "dir1")
	dir2 := filepath.Join(dir1, "dir2")
	dir3 := filepath.Join(root, "dir3")
	dirs := []string{root, dir1, dir2, dir3}
	for _, workingDir := range dirs {
		workingDir := workingDir
		t.Run(fmt.Sprintf("workingDir=%s", workingDir), func(t *testing.T) {
			testCreate(t, workingDir)
		})
	}
}
