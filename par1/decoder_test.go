package par1

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"testing"

	"github.com/klauspost/reedsolomon"
	"github.com/stretchr/testify/require"
)

func rootDir() string {
	// This complexity is only for Windows, the only platform
	// which has the concept of a VolumeName, e.g. C:. We don't
	// care which drive the current working directory is on. On
	// all other platforms, volName is empty.
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	volName := filepath.VolumeName(filepath.Clean(wd))
	return volName + string(filepath.Separator)
}

func toAbsPath(workingDir, path string) string {
	if !filepath.IsAbs(workingDir) {
		panic("workingDir must be an absolute path")
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(workingDir, path)
}

func fileDataToAbsPaths(workingDir string, fileData map[string][]byte) map[string][]byte {
	newFileData := make(map[string][]byte)
	for path, data := range fileData {
		newFileData[toAbsPath(workingDir, path)] = data
	}
	return newFileData
}

type testFileIO struct {
	t          *testing.T
	workingDir string
	fileData   map[string][]byte
}

func makeTestFileIO(t *testing.T, workingDir string, fileData map[string][]byte) testFileIO {
	return testFileIO{t, workingDir, fileDataToAbsPaths(workingDir, fileData)}
}

func (io testFileIO) fileCount() int {
	return len(io.fileData)
}

func (io testFileIO) paths() []string {
	var paths []string
	for path := range io.fileData {
		paths = append(paths, path)
	}
	return paths
}

func (io testFileIO) getData(path string) []byte {
	absPath := toAbsPath(io.workingDir, path)
	data, ok := io.fileData[absPath]
	if !ok {
		io.t.Fatalf("no file at path %s", absPath)
	}
	return data
}

func (io testFileIO) setData(path string, data []byte) {
	absPath := toAbsPath(io.workingDir, path)
	io.fileData[absPath] = data
}

func (io testFileIO) removeData(path string) []byte {
	absPath := toAbsPath(io.workingDir, path)
	data := io.getData(absPath)
	delete(io.fileData, absPath)
	return data
}

func (io testFileIO) ReadFile(path string) (data []byte, err error) {
	io.t.Helper()
	defer func() {
		io.t.Helper()
		io.t.Logf("ReadFile(%s) => (%d, %v)", path, len(data), err)
	}()
	absPath := toAbsPath(io.workingDir, path)
	if data, ok := io.fileData[absPath]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (io testFileIO) WriteFile(path string, data []byte) error {
	io.t.Helper()
	io.t.Logf("WriteFile(%s, %d bytes)", path, len(data))
	absPath := toAbsPath(io.workingDir, path)
	io.fileData[absPath] = data
	return nil
}

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

func buildPARData(t *testing.T, io testFileIO, parityShardCount int) {
	dataShardCount := io.fileCount()
	rs, err := reedsolomon.New(dataShardCount, parityShardCount, reedsolomon.WithPAR1Matrix())
	require.NoError(t, err)

	paths := io.paths()
	sortedPaths := toSortedStrings(paths)

	shardByteCount := 0
	for _, path := range paths {
		dataLength := len(io.getData(path))
		if dataLength > shardByteCount {
			shardByteCount = dataLength
		}
	}
	var shards [][]byte
	for _, path := range sortedPaths {
		data := io.getData(path)
		shards = append(shards, append(data, make([]byte, shardByteCount-len(data))...))
	}
	for i := 0; i < parityShardCount; i++ {
		shards = append(shards, make([]byte, shardByteCount))
	}
	err = rs.Encode(shards)
	require.NoError(t, err)

	var entries []fileEntry
	var setHashInput []byte
	for _, path := range sortedPaths {
		data := io.getData(path)
		var status fileEntryStatus
		status.setSavedInVolumeSet(true)
		hash := md5.Sum(data)
		entry := fileEntry{
			header: fileEntryHeader{
				Status:       status,
				FileBytes:    uint64(len(data)),
				Hash:         hash,
				SixteenKHash: sixteenKHash(data),
			},
			filename: filepath.Base(path),
		}
		entries = append(entries, entry)
		setHashInput = append(setHashInput, hash[:]...)
	}

	vTemplate := volume{
		header: header{
			ID:            expectedID,
			VersionNumber: makeVersionNumber(expectedVersion),
			SetHash:       md5.Sum(setHashInput),
		},
		entries: entries,
	}

	indexVolume := vTemplate
	indexVolume.header.VolumeNumber = 0
	indexVolume.data = []byte{0x1, 0x2}
	indexVolumeBytes, err := writeVolume(indexVolume)
	require.NoError(t, err)

	firstPath := sortedPaths[0]
	ext := path.Ext(firstPath)
	base := firstPath[:len(firstPath)-len(ext)]

	io.setData(base+".par", indexVolumeBytes)

	for i, parityShard := range shards[dataShardCount:] {
		vol := vTemplate
		vol.header.VolumeNumber = uint64(i + 1)
		vol.data = parityShard
		volBytes, err := writeVolume(vol)
		require.NoError(t, err)
		io.setData(fmt.Sprintf("%s.p%02d", base, i+1), volBytes)
	}
}

func makeDecoderTestFileIO(t *testing.T, workingDir string) testFileIO {
	return makeTestFileIO(t, workingDir, map[string][]byte{
		"file.rar": {0x1, 0x2, 0x3, 0x4},
		"file.r01": {0x5, 0x6, 0x7},
		"file.r02": {0x8, 0x9, 0xa, 0xb, 0xc},
		"file.r03": nil,
		"file.r04": {0xd},
	})
}

func testVerify(t *testing.T, workingDir string, useAbsPath bool) {
	io := makeDecoderTestFileIO(t, workingDir)

	buildPARData(t, io, 3)

	parPath := "file.par"
	if useAbsPath {
		parPath = filepath.Join(workingDir, parPath)
	}
	decoder, err := newDecoder(io, testDecoderDelegate{t}, parPath)
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	needsRepair, err := decoder.Verify()
	require.NoError(t, err)
	require.False(t, needsRepair)

	fileData5 := io.getData("file.r04")
	fileData5[len(fileData5)-1]++
	err = decoder.LoadFileData()
	require.NoError(t, err)
	needsRepair, err = decoder.Verify()
	expectedErr := errors.New("shard sizes do not match")
	require.Equal(t, expectedErr, err)

	fileData5[len(fileData5)-1]--
	err = decoder.LoadFileData()
	require.NoError(t, err)
	needsRepair, err = decoder.Verify()
	require.NoError(t, err)
	require.False(t, needsRepair)

	p03Data := io.removeData("file.p03")
	err = decoder.LoadParityData()
	require.NoError(t, err)
	needsRepair, err = decoder.Verify()
	require.NoError(t, err)
	require.False(t, needsRepair)

	io.setData("file.p03", p03Data)
	io.removeData("file.p02")
	err = decoder.LoadParityData()
	require.NoError(t, err)
	needsRepair, err = decoder.Verify()
	expectedErr = errors.New("shard sizes do not match")
	require.Equal(t, expectedErr, err)
}

func TestVerify(t *testing.T) {
	workingDirs := []string{
		rootDir(),
		filepath.Join(rootDir(), "dir"),
		filepath.Join(rootDir(), "dir1", "dir2"),
	}
	for _, workingDir := range workingDirs {
		workingDir := workingDir
		for _, useAbsPath := range []bool{false, true} {
			useAbsPath := useAbsPath
			t.Run(fmt.Sprintf("workingDir=%s,useAbsPath=%t", workingDir, useAbsPath), func(t *testing.T) {
				testVerify(t, workingDir, useAbsPath)
			})
		}
	}
}

func TestSetHashMismatch(t *testing.T) {
	io1 := makeDecoderTestFileIO(t, rootDir())
	io2 := makeDecoderTestFileIO(t, rootDir())
	io2.getData("file.rar")[0]++

	buildPARData(t, io1, 3)
	buildPARData(t, io2, 3)
	// Insert a parity volume that has a different set hash.
	io1.setData("file.p02", io2.getData("file.p02"))

	decoder, err := newDecoder(io1, testDecoderDelegate{t}, "file.par")
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.Equal(t, errors.New("unexpected set hash for parity volume"), err)
}

func TestRepair(t *testing.T) {
	io := makeDecoderTestFileIO(t, rootDir())

	buildPARData(t, io, 3)

	decoder, err := newDecoder(io, testDecoderDelegate{t}, "file.par")
	require.NoError(t, err)

	r02Data := io.getData("file.r02")
	r02DataCopy := make([]byte, len(r02Data))
	copy(r02DataCopy, r02Data)
	r02Data[len(r02Data)-1]++
	io.removeData("file.r03")
	r04Data := io.removeData("file.r04")

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repaired, err := decoder.Repair(true)
	require.NoError(t, err)

	// removeData returns nil for "file.r03", but Repair writes a
	// zero-length array instead.
	expectedR03Data := []byte{}
	require.Equal(t, []string{"file.r02", "file.r03", "file.r04"}, repaired)
	require.Equal(t, r02DataCopy, io.getData("file.r02"))
	require.Equal(t, expectedR03Data, io.getData("file.r03"))
	require.Equal(t, r04Data, io.getData("file.r04"))
}
