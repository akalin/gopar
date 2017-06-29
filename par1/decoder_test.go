package par1

import (
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

func (d testDecoderDelegate) OnDataFileLoad(path string, err error) {
	d.t.Logf("OnDataFileLoad(%s, %v)", path, err)
}

func (d testDecoderDelegate) OnVolumeFileLoad(path string, err error) {
	d.t.Logf("OnVolumeFileLoad(%s, %v)", path, err)
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
	for _, k := range keys {
		entry := fileEntry{
			header: fileEntryHeader{
				FileBytes: uint64(len(io.fileData[k])),
			},
			filename: k,
		}
		entries = append(entries, entry)
	}

	vTemplate := volume{
		header: header{
			ID:            expectedID,
			VersionNumber: expectedVersion,
			SetHash:       [16]byte{0x3, 0x4},
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

	delete(io.fileData, "file.r03")
	r04Data := io.fileData["file.r04"]
	delete(io.fileData, "file.r04")

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repaired, err := decoder.Repair()
	require.NoError(t, err)

	require.Equal(t, []string{"file.r03", "file.r04"}, repaired)
	require.Equal(t, 0, len(io.fileData["file.r03"]))
	require.Equal(t, r04Data, io.fileData["file.r04"])
}
