package par1

import (
	"fmt"
	"os"
	"reflect"
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

func TestVerify(t *testing.T) {
	io := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3, 0x4},
			"file.r01": {0x5, 0x6, 0x7, 0x8},
			"file.r02": {0x9, 0xa, 0xb, 0xc},
			"file.r03": {0xd, 0xe, 0xf, 0x10},
			"file.r04": {0x11, 0x12, 0x13, 0x14},
		},
	}

	dataShardCount := len(io.fileData)
	rs, err := reedsolomon.New(dataShardCount, 3, reedsolomon.WithPAR1Matrix())
	require.NoError(t, err)
	shards := [][]byte{
		io.fileData["file.rar"],
		io.fileData["file.r01"],
		io.fileData["file.r02"],
		io.fileData["file.r03"],
		io.fileData["file.r04"],
		make([]byte, 4),
		make([]byte, 4),
		make([]byte, 4),
	}
	err = rs.Encode(shards)
	require.NoError(t, err)

	filenameByteCount := uint64(len(encodeUTF16LEString("file.rar")))
	entryByteCount := uint64(reflect.TypeOf(fileEntryHeader{}).Size()) + filenameByteCount

	vTemplate := volume{
		header: header{
			ID:             expectedID,
			VersionNumber:  expectedVersion,
			ControlHash:    [16]uint8{},
			SetHash:        [16]byte{0x3, 0x4},
			FileCount:      5,
			FileListOffset: expectedFileListOffset,
			FileListBytes:  0,
			DataOffset:     expectedFileListOffset,
			DataBytes:      0,
		},
		entries: []fileEntry{
			fileEntry{
				header: fileEntryHeader{
					EntryBytes: entryByteCount,
				},
				filename: "file.rar",
			},
			fileEntry{
				header: fileEntryHeader{
					EntryBytes: entryByteCount,
				},
				filename: "file.r01",
			},
			fileEntry{
				header: fileEntryHeader{
					EntryBytes: entryByteCount,
				},
				filename: "file.r02",
			},
			fileEntry{
				header: fileEntryHeader{
					EntryBytes: entryByteCount,
				},
				filename: "file.r03",
			},
			fileEntry{
				header: fileEntryHeader{
					EntryBytes: entryByteCount,
				},
				filename: "file.r04",
			},
		},
	}

	indexVolume := vTemplate
	indexVolume.header.VolumeNumber = 0
	indexVolumeBytes, err := writeVolume(indexVolume)
	require.NoError(t, err)
	io.fileData["file.par"] = indexVolumeBytes

	for i, parityShard := range shards[dataShardCount:] {
		vol := vTemplate
		vol.header.VolumeNumber = uint64(i + 1)
		vol.data = parityShard
		volBytes, err := writeVolume(vol)
		require.NoError(t, err)
		io.fileData[fmt.Sprintf("file.p%02d", i+1)] = volBytes
	}

	decoder, err := newDecoder(io, "file.par")
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
