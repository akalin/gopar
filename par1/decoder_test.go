package par1

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"testing"

	"github.com/akalin/gopar/fs"
	"github.com/akalin/gopar/hashutil"
	"github.com/akalin/gopar/memfs"
	"github.com/akalin/gopar/testfs"
	"github.com/klauspost/reedsolomon"
	"github.com/stretchr/testify/require"
)

type testDecoderDelegate struct {
	t *testing.T
}

func (d testDecoderDelegate) OnHeaderLoad(headerInfo string) {
	d.t.Helper()
	d.t.Logf("OnHeaderLoad(%s)", headerInfo)
}

func (d testDecoderDelegate) OnFileEntryLoad(i, n int, filename, entryInfo string) {
	d.t.Helper()
	d.t.Logf("OnFileEntryLoad(%d, %d, %s, %s)", i, n, filename, entryInfo)
}

func (d testDecoderDelegate) OnCommentLoad(comment []byte) {
	d.t.Helper()
	d.t.Logf("OnCommentLoad(%q)", comment)
}

func (d testDecoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, corrupt bool, err error) {
	d.t.Helper()
	d.t.Logf("OnDataFileLoad(%d, %d, %s, byteCount=%d, corrupt=%t, %v)", i, n, path, byteCount, corrupt, err)
}

func (d testDecoderDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {
	d.t.Helper()
	d.t.Logf("OnDataFileWrite(%d, %d, %s, byteCount=%d, %v)", i, n, path, byteCount, err)
}

func (d testDecoderDelegate) OnVolumeFileLoad(i uint64, path string, storedSetHash, computedSetHash [16]byte, dataByteCount int, err error) {
	d.t.Helper()
	d.t.Logf("OnVolumeFileLoad(%d, %s, storedSetHash=%x, computedSetHash=%x, dataByteCount=%d, %v)", i, path, storedSetHash, computedSetHash, dataByteCount, err)
}

func toSortedStrings(arr []string) []string {
	arrCopy := make([]string, len(arr))
	copy(arrCopy, arr)
	sort.Strings(arrCopy)
	return arrCopy
}

func buildVTemplate(t *testing.T, memFS memfs.MemFS, sortedPaths []string) volume {
	var entries []fileEntry
	for _, path := range sortedPaths {
		hasher := hashutil.MakeMD5HasherWith16k()
		readStream, err := memFS.GetReadStream(path)
		require.NoError(t, err)
		data, err := fs.ReadAndClose(hashutil.TeeReadStream(readStream, hasher))
		require.NoError(t, err)
		var status fileEntryStatus
		status.setSavedInVolumeSet(true)
		hash, hash16k := hasher.Hashes()
		entry := fileEntry{
			header: fileEntryHeader{
				Status:    status,
				FileBytes: uint64(len(data)),
				Hash:      hash,
				Hash16k:   hash16k,
			},
			filename: filepath.Base(path),
		}
		entries = append(entries, entry)
	}

	return volume{
		header: header{
			ID:            expectedID,
			VersionNumber: makeVersionNumber(expectedVersion),
			SetHash:       computeSetHash(entries),
		},
		entries: entries,
	}
}

func buildPARData(t *testing.T, fs memfs.MemFS, parityShardCount int) {
	dataShardCount := fs.FileCount()
	rs, err := reedsolomon.New(dataShardCount, parityShardCount, reedsolomon.WithPAR1Matrix())
	require.NoError(t, err)

	paths := fs.Paths()
	sortedPaths := toSortedStrings(paths)

	shardByteCount := 0
	for _, path := range paths {
		data, err := fs.ReadFile(path)
		require.NoError(t, err)
		dataLength := len(data)
		if dataLength > shardByteCount {
			shardByteCount = dataLength
		}
	}
	var shards [][]byte
	for _, path := range sortedPaths {
		data, err := fs.ReadFile(path)
		require.NoError(t, err)
		shards = append(shards, append(data, make([]byte, shardByteCount-len(data))...))
	}
	for i := 0; i < parityShardCount; i++ {
		shards = append(shards, make([]byte, shardByteCount))
	}
	err = rs.Encode(shards)
	require.NoError(t, err)

	vTemplate := buildVTemplate(t, fs, sortedPaths)

	indexVolume := vTemplate
	indexVolume.header.VolumeNumber = 0
	indexVolume.data = []byte{0x1, 0x2}
	indexVolumeBytes, err := writeVolume(indexVolume)
	require.NoError(t, err)

	firstPath := sortedPaths[0]
	ext := path.Ext(firstPath)
	base := firstPath[:len(firstPath)-len(ext)]

	require.NoError(t, fs.WriteFile(base+".par", indexVolumeBytes))

	for i, parityShard := range shards[dataShardCount:] {
		vol := vTemplate
		vol.header.VolumeNumber = uint64(i + 1)
		vol.data = parityShard
		volBytes, err := writeVolume(vol)
		require.NoError(t, err)
		require.NoError(t, fs.WriteFile(fmt.Sprintf("%s.p%02d", base, i+1), volBytes))
	}
}

func makeDecoderMemFS(workingDir string) memfs.MemFS {
	return memfs.MakeMemFS(workingDir, map[string][]byte{
		"file.rar": {0x1, 0x2, 0x3, 0x4},
		"file.r01": {0x5, 0x6, 0x7},
		"file.r02": {0x8, 0x9, 0xa, 0xb, 0xc},
		"file.r03": nil,
		"file.r04": {0xd},
	})
}

func newDecoderForTest(t *testing.T, fs memfs.MemFS, indexFile string) (*Decoder, error) {
	return newDecoder(testfs.MakeTestFS(t, fs), testDecoderDelegate{t}, indexFile)
}

func perturbFile(t *testing.T, fs memfs.MemFS, path string) {
	data, err := fs.ReadFile(path)
	require.NoError(t, err)
	data[len(data)-1]++
}

func unperturbFile(t *testing.T, fs memfs.MemFS, path string) {
	data, err := fs.ReadFile(path)
	require.NoError(t, err)
	data[len(data)-1]--
}

func testFileCounts(t *testing.T, workingDir string, useAbsPath bool) {
	fs := makeDecoderMemFS(workingDir)
	dataFileCount := fs.FileCount()
	parityFileCount := 3

	buildPARData(t, fs, parityFileCount)

	parPath := "file.par"
	if useAbsPath {
		parPath = filepath.Join(workingDir, parPath)
	}
	decoder, err := newDecoderForTest(t, fs, parPath)
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	require.Equal(t, FileCounts{
		UsableDataFileCount:   dataFileCount,
		UsableParityFileCount: parityFileCount,
	}, decoder.FileCounts())

	perturbFile(t, fs, "file.r04")
	err = decoder.LoadFileData()
	require.NoError(t, err)
	require.Equal(t, FileCounts{
		UsableDataFileCount:   dataFileCount - 1,
		UnusableDataFileCount: 1,
		UsableParityFileCount: parityFileCount,
	}, decoder.FileCounts())

	unperturbFile(t, fs, "file.r04")
	err = decoder.LoadFileData()
	require.NoError(t, err)
	require.Equal(t, FileCounts{
		UsableDataFileCount:   dataFileCount,
		UsableParityFileCount: parityFileCount,
	}, decoder.FileCounts())

	p03Data, err := fs.RemoveFile("file.p03")
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)
	require.Equal(t, FileCounts{
		UsableDataFileCount:     dataFileCount,
		UsableParityFileCount:   parityFileCount - 1,
		UnusableParityFileCount: 0,
	}, decoder.FileCounts())

	require.NoError(t, fs.WriteFile("file.p03", p03Data))
	_, err = fs.RemoveFile("file.p02")
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)
	require.Equal(t, FileCounts{
		UsableDataFileCount:     dataFileCount,
		UsableParityFileCount:   parityFileCount - 1,
		UnusableParityFileCount: 1,
	}, decoder.FileCounts())
}

func runOnExampleWorkingDirs(t *testing.T, testFn func(*testing.T, string, bool)) {
	workingDirs := []string{
		memfs.RootDir(),
		filepath.Join(memfs.RootDir(), "dir"),
		filepath.Join(memfs.RootDir(), "dir1", "dir2"),
	}
	for _, workingDir := range workingDirs {
		workingDir := workingDir
		for _, useAbsPath := range []bool{false, true} {
			useAbsPath := useAbsPath
			t.Run(fmt.Sprintf("workingDir=%s,useAbsPath=%t", workingDir, useAbsPath), func(t *testing.T) {
				testFn(t, workingDir, useAbsPath)
			})
		}
	}
}

func TestFileCounts(t *testing.T) {
	runOnExampleWorkingDirs(t, testFileCounts)
}

func testVerifyAllData(t *testing.T, workingDir string, useAbsPath bool) {
	fs := makeDecoderMemFS(workingDir)

	buildPARData(t, fs, 3)

	parPath := "file.par"
	if useAbsPath {
		parPath = filepath.Join(workingDir, parPath)
	}
	decoder, err := newDecoderForTest(t, fs, parPath)
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	ok, err := decoder.VerifyAllData()
	require.NoError(t, err)
	require.True(t, ok)

	perturbFile(t, fs, "file.r04")
	err = decoder.LoadFileData()
	require.NoError(t, err)
	_, err = decoder.VerifyAllData()
	expectedErr := errors.New("shard sizes do not match")
	require.Equal(t, expectedErr, err)

	unperturbFile(t, fs, "file.r04")
	err = decoder.LoadFileData()
	require.NoError(t, err)
	ok, err = decoder.VerifyAllData()
	require.NoError(t, err)
	require.True(t, ok)

	p03Data, err := fs.RemoveFile("file.p03")
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)
	ok, err = decoder.VerifyAllData()
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, fs.WriteFile("file.p03", p03Data))
	_, err = fs.RemoveFile("file.p02")
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)
	_, err = decoder.VerifyAllData()
	expectedErr = errors.New("shard sizes do not match")
	require.Equal(t, expectedErr, err)
}

func TestVerifyAllData(t *testing.T) {
	runOnExampleWorkingDirs(t, testVerifyAllData)
}

func TestBadFilename(t *testing.T) {
	workingDir := filepath.Join(memfs.RootDir(), "dir")
	fs := memfs.MakeMemFS(workingDir, map[string][]byte{
		"file.rar": {0x1, 0x2, 0x3, 0x4},
	})

	indexVolume := buildVTemplate(t, fs, []string{"file.rar"})
	indexVolume.header.VolumeNumber = 0
	indexVolume.data = []byte{0x1, 0x2}
	indexVolume.entries[0].filename = filepath.Join("dir", "file.rar")
	indexVolumeBytes, err := writeVolume(indexVolume)
	require.NoError(t, err)

	require.NoError(t, fs.WriteFile("file.par", indexVolumeBytes))

	decoder, err := newDecoderForTest(t, fs, "file.par")
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.Equal(t, errors.New("bad filename"), err)
}

func TestSetHashMismatch(t *testing.T) {
	fs1 := makeDecoderMemFS(memfs.RootDir())
	fs2 := makeDecoderMemFS(memfs.RootDir())
	rarData, err := fs2.ReadFile("file.rar")
	require.NoError(t, err)
	rarData[0]++

	buildPARData(t, fs1, 3)
	buildPARData(t, fs2, 3)
	// Insert a parity volume that has a different set hash.
	p02Data, err := fs2.ReadFile("file.p02")
	require.NoError(t, err)
	require.NoError(t, fs1.WriteFile("file.p02", p02Data))

	decoder, err := newDecoderForTest(t, fs1, "file.par")
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.Equal(t, errors.New("unexpected set hash for parity volume"), err)
}

func testDecoderRepair(t *testing.T, workingDir string, useAbsPath bool) {
	fs := makeDecoderMemFS(workingDir)

	buildPARData(t, fs, 3)

	parPath := "file.par"
	if useAbsPath {
		parPath = filepath.Join(workingDir, parPath)
	}
	decoder, err := newDecoderForTest(t, fs, parPath)
	require.NoError(t, err)

	r02Data, err := fs.ReadFile("file.r02")
	require.NoError(t, err)
	r02DataCopy := make([]byte, len(r02Data))
	copy(r02DataCopy, r02Data)
	r02Data[len(r02Data)-1]++
	_, err = fs.RemoveFile("file.r03")
	require.NoError(t, err)
	r04Data, err := fs.RemoveFile("file.r04")
	require.NoError(t, err)

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repairedPaths, err := decoder.Repair(true)
	require.NoError(t, err)

	// removeData returns nil for "file.r03", but Repair writes a
	// zero-length array instead.
	expectedR03Data := []byte{}
	expectedRepairedPaths := []string{"file.r02", "file.r03", "file.r04"}
	if useAbsPath {
		for i, path := range expectedRepairedPaths {
			expectedRepairedPaths[i] = filepath.Join(workingDir, path)
		}
	}
	require.Equal(t, toSortedStrings(expectedRepairedPaths), toSortedStrings(repairedPaths))
	repairedR02Data, err := fs.ReadFile("file.r02")
	require.NoError(t, err)
	require.Equal(t, r02DataCopy, repairedR02Data)
	repairedR03Data, err := fs.ReadFile("file.r03")
	require.NoError(t, err)
	require.Equal(t, expectedR03Data, repairedR03Data)
	repairedR04Data, err := fs.ReadFile("file.r04")
	require.NoError(t, err)
	require.Equal(t, r04Data, repairedR04Data)
}

func TestDecoderRepair(t *testing.T) {
	runOnExampleWorkingDirs(t, testDecoderRepair)
}
