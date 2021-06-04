package par1

import (
	"crypto/md5"
	"io/ioutil"
	"testing"

	"github.com/akalin/gopar/memfs"
	"github.com/stretchr/testify/require"
)

var fileEntries = []fileEntry{
	{
		header: fileEntryHeader{
			// Not included in volume set.
			Status:    10,
			FileBytes: 10,
			Hash:      [16]byte{0x1, 0x2},
			Hash16k:   [16]byte{0x5, 0x6},
		},
		filename: "filename世界.r01",
	},
	{
		header: fileEntryHeader{
			// Included in volume set.
			Status:    11,
			FileBytes: 10,
			Hash:      [16]byte{0x3, 0x4},
			Hash16k:   [16]byte{0x7, 0x8},
		},
		filename: "filename世界.r02",
	},
}

func TestComputeSetHash(t *testing.T) {
	// Only the second file is included in the volume set.
	expectedSetHash := md5.Sum(fileEntries[1].header.Hash[:])
	require.Equal(t, expectedSetHash, computeSetHash(fileEntries))
}

func TestVolumeRoundTrip(t *testing.T) {
	setHash := computeSetHash(fileEntries)
	expectedData := []byte{0x1, 0x2}
	v := volume{
		header: header{
			ID:            expectedID,
			VersionNumber: makeVersionNumber(expectedVersion),
			SetHash:       setHash,
			VolumeNumber:  5,
		},
		entries: fileEntries,
	}

	volumeBytes, err := writeVolume(v, expectedData)
	require.NoError(t, err)

	readStream := memfs.MakeReadStream(volumeBytes)
	roundTripVolume, err := readVolume(readStream)
	require.NoError(t, err)
	data, err := ioutil.ReadAll(readStream)
	require.NoError(t, err)

	v.header.ControlHash = md5.Sum(volumeBytes[controlHashOffset:])
	v.header.FileCount = uint64(len(v.entries))
	v.header.FileListOffset = expectedFileListOffset
	v.header.FileListBytes = uint64(len(volumeBytes)) - expectedFileListOffset - uint64(len(data))
	v.header.DataOffset = v.header.FileListOffset + v.header.FileListBytes
	v.header.DataBytes = uint64(len(data))
	for i, entry := range v.entries {
		entryBytes, err := writeFileEntry(entry)
		require.NoError(t, err)
		v.entries[i].header.EntryBytes = uint64(len(entryBytes))
	}
	require.Equal(t, v, roundTripVolume)
}
