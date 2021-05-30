package par2

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/akalin/gopar/memfs"
	"github.com/akalin/gopar/rsec16"
	"github.com/akalin/gopar/testfs"
	"github.com/stretchr/testify/require"
)

type testEncoderDelegate struct {
	t *testing.T
}

func (d testEncoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {
	d.t.Helper()
	d.t.Logf("OnDataFileLoad(%d, %d, byteCount=%d, %s, %v)", i, n, byteCount, path, err)
}

func (d testEncoderDelegate) OnIndexFileWrite(path string, byteCount int, err error) {
	d.t.Helper()
	d.t.Logf("OnIndexFileWrite(%s, %d, %v)", path, byteCount, err)
}

func (d testEncoderDelegate) OnRecoveryFileWrite(start, count, total int, path string, dataByteCount, byteCount int, err error) {
	d.t.Helper()
	d.t.Logf("OnRecoveryFileWrite(start=%d, count=%d, total=%d, %s, dataByteCount=%d, byteCount=%d, %v)", start, count, total, path, dataByteCount, byteCount, err)
}

func newEncoderForTest(t *testing.T, fs memfs.MemFS, basePath string, paths []string, sliceByteCount, parityShardCount int) (*Encoder, error) {
	return newEncoder(testfs.TestFS{T: t, FS: fs}, testEncoderDelegate{t}, basePath, paths, sliceByteCount, parityShardCount, rsec16.DefaultNumGoroutines())
}

func makeEncoderMemFS(workingDir string) memfs.MemFS {
	return memfs.MakeMemFS(workingDir, map[string][]byte{
		"file.rar":                                {0x1, 0x2, 0x3},
		filepath.Join("dir1", "file.r01"):         {0x5, 0x6, 0x7, 0x8},
		filepath.Join("dir1", "file.r02"):         {0x9, 0xa, 0xb, 0xc},
		filepath.Join("dir2", "dir3", "file.r03"): {0xd, 0xe},
		filepath.Join("dir4", "dir5", "file.r04"): {0xf},
	})
}

func TestEncodeParity(t *testing.T) {
	workingDir := memfs.RootDir()
	fs := makeEncoderMemFS(workingDir)

	paths := fs.Paths()

	sliceByteCount := 4
	parityShardCount := 3
	encoder, err := newEncoderForTest(t, fs, workingDir, paths, sliceByteCount, parityShardCount)
	require.NoError(t, err)

	err = encoder.LoadFileData()
	require.NoError(t, err)

	err = encoder.ComputeParityData()
	require.NoError(t, err)

	var recoverySet []fileID
	dataShardsByID := make(map[fileID][][]byte)
	for _, path := range paths {
		data, err := fs.ReadFile(path)
		require.NoError(t, err)
		relPath, err := filepath.Rel(workingDir, path)
		require.NoError(t, err)
		fileID, _, _, fileDataShards := computeDataFileInfo(sliceByteCount, relPath, data)
		recoverySet = append(recoverySet, fileID)
		dataShardsByID[fileID] = fileDataShards
	}

	sort.Slice(recoverySet, func(i, j int) bool {
		return fileIDLess(recoverySet[i], recoverySet[j])
	})

	var dataShards [][]byte
	for _, fileID := range recoverySet {
		dataShards = append(dataShards, dataShardsByID[fileID]...)
	}

	coder, err := rsec16.NewCoderPAR2Vandermonde(len(dataShards), parityShardCount, rsec16.DefaultNumGoroutines())
	require.NoError(t, err)

	computedParityShards := coder.GenerateParity(dataShards)
	require.Equal(t, computedParityShards, encoder.parityShards)
}

func testWriteParity(t *testing.T, workingDir, outputPath string) {
	fs := makeEncoderMemFS(workingDir)

	paths := fs.Paths()

	sliceByteCount := 4
	parityShardCount := 100
	encoder, err := newEncoderForTest(t, fs, workingDir, paths, sliceByteCount, parityShardCount)
	require.NoError(t, err)

	err = encoder.LoadFileData()
	require.NoError(t, err)

	err = encoder.ComputeParityData()
	require.NoError(t, err)

	parPath := filepath.Join(outputPath, "parity.par2")
	err = encoder.Write(parPath)
	require.NoError(t, err)

	var movedParPath string
	if workingDir == outputPath {
		movedParPath = parPath
	} else {
		decoder, err := newDecoderForTest(t, fs, parPath)
		require.NoError(t, err)

		err = decoder.LoadFileData()
		require.NoError(t, err)
		err = decoder.LoadParityData()
		require.NoError(t, err)

		require.True(t, decoder.ShardCounts().RepairNeeded())

		// Call fs.Paths() again to pick up newly-written files.
		for _, path := range fs.Paths() {
			if strings.HasPrefix(path, filepath.Join(outputPath, "parity")) {
				relPath, err := filepath.Rel(outputPath, path)
				require.NoError(t, err)
				movedPath := filepath.Join(workingDir, relPath)
				require.NoError(t, fs.MoveFile(path, movedPath))
			}
		}

		movedParPath = filepath.Join(workingDir, "parity.par2")
	}

	decoder, err := newDecoderForTest(t, fs, movedParPath)
	require.NoError(t, err)

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	require.False(t, decoder.ShardCounts().RepairNeeded())
}

func TestWriteParity(t *testing.T) {
	root := memfs.RootDir()
	dir1 := filepath.Join(root, "dir1")
	dir2 := filepath.Join(dir1, "dir2")
	dir3 := filepath.Join(root, "dir3")
	dirs := []string{root, dir1, dir2, dir3}
	for _, workingDir := range dirs {
		workingDir := workingDir
		for _, outputDir := range dirs {
			outputDir := outputDir
			t.Run(fmt.Sprintf("workingDir=%s,outputDir=%s", workingDir, outputDir), func(t *testing.T) {
				testWriteParity(t, workingDir, outputDir)
			})
		}
	}
}

func TestWriteParityFilesOutOfBasePath(t *testing.T) {
	dir := memfs.RootDir()
	fs := makeEncoderMemFS(dir)

	paths := fs.Paths()

	sliceByteCount := 4
	parityShardCount := 100
	_, err := newEncoderForTest(t, fs, filepath.Join(dir, "somedir"), paths, sliceByteCount, parityShardCount)
	require.Equal(t, errors.New("data files must lie in basePath"), err)
}
