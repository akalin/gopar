package par1

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVolumeRoundTrip(t *testing.T) {
	filename1 := "filename世界.r01"
	filename1ByteCount := uint64(len(encodeUTF16LEString(filename1)))
	filename2 := "filename世界.r02"
	filename2ByteCount := uint64(len(encodeUTF16LEString(filename2)))
	headerByteCount := uint64(reflect.TypeOf(fileEntryHeader{}).Size())
	entry1ByteCount := headerByteCount + filename1ByteCount
	entry2ByteCount := headerByteCount + filename2ByteCount

	v := volume{
		header: header{
			ID:             expectedID,
			VersionNumber:  expectedVersion,
			ControlHash:    [16]uint8{},
			SetHash:        [16]byte{0x3, 0x4},
			VolumeNumber:   5,
			FileCount:      2,
			FileListOffset: expectedFileListOffset,
			FileListBytes:  0,
			DataOffset:     expectedFileListOffset,
			DataBytes:      0,
		},
		entries: []fileEntry{
			fileEntry{
				header: fileEntryHeader{
					EntryBytes:   entry1ByteCount,
					Status:       10,
					FileBytes:    10,
					Hash:         [16]byte{0x1, 0x2},
					SixteenKHash: [16]byte{0x3, 0x4},
				},
				filename: filename1,
			},
			fileEntry{
				header: fileEntryHeader{
					EntryBytes:   entry2ByteCount,
					Status:       10,
					FileBytes:    10,
					Hash:         [16]byte{0x1, 0x2},
					SixteenKHash: [16]byte{0x3, 0x4},
				},
				filename: filename2,
			},
		},
		data: []byte{0x1, 0x2},
	}

	volumeBytes, err := writeVolume(v)
	require.NoError(t, err)
	roundTripVolume, err := readVolume(volumeBytes)
	roundTripVolume.header.ControlHash = [16]byte{}
	require.NoError(t, err)
	require.Equal(t, v, roundTripVolume)
}
