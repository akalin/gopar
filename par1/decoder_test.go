package par1

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"testing"

	"github.com/klauspost/reedsolomon"
	"github.com/stretchr/testify/require"
)

type testFileIO struct {
	t        *testing.T
	fileData map[string][]byte
}

func (io testFileIO) ReadFile(path string) (data []byte, err error) {
	defer func() {
		io.t.Logf("ReadFile(%s) => (%d, %v)", path, len(data), err)
	}()
	if data, ok := io.fileData[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (io testFileIO) WriteFile(path string, data []byte) error {
	io.t.Logf("WriteFile(%s, %d bytes)", path, len(data))
	io.fileData[path] = data
	return nil
}

type testDecoderDelegate struct {
	t *testing.T
}

func (d testDecoderDelegate) OnHeaderLoad(headerInfo string) {
	d.t.Logf("OnHeaderLoad(%s)", headerInfo)
}

func (d testDecoderDelegate) OnFileEntryLoad(i, n int, filename, entryInfo string) {
	d.t.Logf("OnFileEntryLoad(%d, %d, %s, %s)", i, n, filename, entryInfo)
}

func (d testDecoderDelegate) OnCommentLoad(comment []byte) {
	d.t.Logf("OnCommentLoad(%q)", comment)
}

func (d testDecoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, corrupt bool, err error) {
	d.t.Logf("OnDataFileLoad(%d, %d, %s, byteCount=%d, corrupt=%t, %v)", i, n, path, byteCount, corrupt, err)
}

func (d testDecoderDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {
	d.t.Logf("OnDataFileWrite(%d, %d, %s, byteCount=%d, %v)", i, n, path, byteCount, err)
}

func (d testDecoderDelegate) OnVolumeFileLoad(i uint64, path string, storedSetHash, computedSetHash [16]byte, dataByteCount int, err error) {
	d.t.Logf("OnVolumeFileLoad(%d, %s, storedSetHash=%x, computedSetHash=%x, dataByteCount=%d, %v)", i, path, storedSetHash, computedSetHash, dataByteCount, err)
}

func buildPARData(t *testing.T, io testFileIO, parityShardCount int) {
	dataShardCount := len(io.fileData)
	rs, err := reedsolomon.New(dataShardCount, parityShardCount, reedsolomon.WithPAR1Matrix())
	require.NoError(t, err)

	var keys []string
	for k := range io.fileData {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	shardByteCount := 0
	for _, k := range keys {
		if len(io.fileData[k]) > shardByteCount {
			shardByteCount = len(io.fileData[k])
		}
	}
	var shards [][]byte
	for _, k := range keys {
		shards = append(shards, append(io.fileData[k], make([]byte, shardByteCount-len(io.fileData[k]))...))
	}
	for i := 0; i < parityShardCount; i++ {
		shards = append(shards, make([]byte, shardByteCount))
	}
	err = rs.Encode(shards)
	require.NoError(t, err)

	var entries []fileEntry
	var setHashInput []byte
	for _, k := range keys {
		data := io.fileData[k]
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
			filename: k,
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

	ext := path.Ext(keys[0])
	base := keys[0][:len(keys[0])-len(ext)]

	io.fileData[base+".par"] = indexVolumeBytes

	for i, parityShard := range shards[dataShardCount:] {
		vol := vTemplate
		vol.header.VolumeNumber = uint64(i + 1)
		vol.data = parityShard
		volBytes, err := writeVolume(vol)
		require.NoError(t, err)
		io.fileData[fmt.Sprintf("%s.p%02d", base, i+1)] = volBytes
	}
}

func TestVerify(t *testing.T) {
	io := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3, 0x4},
			"file.r01": {0x5, 0x6, 0x7},
			"file.r02": {0x8, 0x9, 0xa, 0xb, 0xc},
			"file.r03": nil,
			"file.r04": {0xd},
		},
	}

	buildPARData(t, io, 3)

	decoder, err := newDecoder(io, testDecoderDelegate{t}, "file.par")
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	ok, err := decoder.Verify()
	require.NoError(t, err)
	require.True(t, ok)

	fileData5 := io.fileData["file.r04"]
	fileData5[len(fileData5)-1]++
	err = decoder.LoadFileData()
	require.NoError(t, err)
	ok, err = decoder.Verify()
	require.NoError(t, err)
	require.False(t, ok)

	fileData5[len(fileData5)-1]--
	err = decoder.LoadFileData()
	require.NoError(t, err)
	ok, err = decoder.Verify()
	require.NoError(t, err)
	require.True(t, ok)

	p03Data := io.fileData["file.p03"]
	delete(io.fileData, "file.p03")
	err = decoder.LoadParityData()
	require.NoError(t, err)
	ok, err = decoder.Verify()
	require.NoError(t, err)
	require.True(t, ok)

	io.fileData["file.p03"] = p03Data
	delete(io.fileData, "file.p02")
	err = decoder.LoadParityData()
	require.NoError(t, err)
	ok, err = decoder.Verify()
	require.NoError(t, err)
	require.False(t, ok)
}

func TestSetHashMismatch(t *testing.T) {
	io1 := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3, 0x4},
			"file.r01": {0x5, 0x6, 0x7},
			"file.r02": {0x8, 0x9, 0xa, 0xb, 0xc},
			"file.r03": nil,
			"file.r04": {0xd},
		},
	}

	io2 := testFileIO{
		t:        t,
		fileData: make(map[string][]byte),
	}
	for k, v := range io1.fileData {
		io2.fileData[k] = make([]byte, len(v))
		copy(io2.fileData[k], v)
	}
	io2.fileData["file.rar"][0]++

	buildPARData(t, io1, 3)
	buildPARData(t, io2, 3)
	// Insert a parity volume that has a different set hash.
	io1.fileData["file.p02"] = io2.fileData["file.p02"]

	decoder, err := newDecoder(io1, testDecoderDelegate{t}, "file.par")
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.Equal(t, errors.New("unexpected set hash for parity volume"), err)
}

func TestRepair(t *testing.T) {
	io := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3, 0x4},
			"file.r01": {0x5, 0x6, 0x7},
			"file.r02": {0x8, 0x9, 0xa, 0xb, 0xc},
			"file.r03": nil,
			"file.r04": {0xd},
		},
	}

	buildPARData(t, io, 3)

	decoder, err := newDecoder(io, testDecoderDelegate{t}, "file.par")
	require.NoError(t, err)

	r02Data := io.fileData["file.r02"]
	r02DataCopy := make([]byte, len(r02Data))
	copy(r02DataCopy, r02Data)
	r02Data[len(r02Data)-1]++
	delete(io.fileData, "file.r03")
	r04Data := io.fileData["file.r04"]
	delete(io.fileData, "file.r04")

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repaired, err := decoder.Repair()
	require.NoError(t, err)

	require.Equal(t, []string{"file.r02", "file.r03", "file.r04"}, repaired)
	require.Equal(t, r02DataCopy, io.fileData["file.r02"])
	require.Equal(t, 0, len(io.fileData["file.r03"]))
	require.Equal(t, r04Data, io.fileData["file.r04"])
}
