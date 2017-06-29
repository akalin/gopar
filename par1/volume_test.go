package par1

import (
	"crypto/md5"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVolumeRoundTrip(t *testing.T) {
	v := volume{
		header: header{
			ID:            expectedID,
			VersionNumber: expectedVersion,
			SetHash:       [16]byte{0x3, 0x4},
			VolumeNumber:  5,
		},
		entries: []fileEntry{
			fileEntry{
				header: fileEntryHeader{
					Status:       10,
					FileBytes:    10,
					Hash:         [16]byte{0x1, 0x2},
					SixteenKHash: [16]byte{0x3, 0x4},
				},
				filename: "filename世界.r01",
			},
			fileEntry{
				header: fileEntryHeader{
					Status:       10,
					FileBytes:    10,
					Hash:         [16]byte{0x1, 0x2},
					SixteenKHash: [16]byte{0x3, 0x4},
				},
				filename: "filename世界.r02",
			},
		},
		data: []byte{0x1, 0x2},
	}

	volumeBytes, err := writeVolume(v)
	require.NoError(t, err)

	roundTripVolume, err := readVolume(volumeBytes)
	require.NoError(t, err)

	v.header.ControlHash = md5.Sum(volumeBytes[controlHashOffset:])
	v.header.FileCount = uint64(len(v.entries))
	v.header.FileListOffset = expectedFileListOffset
	v.header.FileListBytes = uint64(len(volumeBytes)) - expectedFileListOffset - uint64(len(v.data))
	v.header.DataOffset = v.header.FileListOffset + v.header.FileListBytes
	v.header.DataBytes = uint64(len(v.data))
	for i, entry := range v.entries {
		entryBytes, err := writeFileEntry(entry)
		require.NoError(t, err)
		v.entries[i].header.EntryBytes = uint64(len(entryBytes))
	}
	require.Equal(t, v, roundTripVolume)
}
