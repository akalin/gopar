package par1

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUTF16LEStringRoundTrip(t *testing.T) {
	for _, s := range []string{
		"",
		"Hello, world",
		"Hello, 世界",
		"Hello\000world",
	} {
		encodedS := encodeUTF16LEString(s)
		roundTripS := decodeUTF16LEString(encodedS)
		require.Equal(t, s, roundTripS)
	}
}

func TestFileEntryRoundTrip(t *testing.T) {
	filename := "filename世界.r01"
	filenameByteCount := uint64(len(encodeUTF16LEString(filename)))
	entryByteCount := uint64(reflect.TypeOf(fileEntryHeader{}).Size()) + filenameByteCount
	entry := fileEntry{
		header: fileEntryHeader{
			EntryBytes:   entryByteCount,
			Status:       10,
			FileBytes:    10,
			Hash:         [16]byte{0x1, 0x2},
			SixteenKHash: [16]byte{0x3, 0x4},
		},
		filename: filename,
	}

	entryBytes, err := writeFileEntry(entry)
	require.NoError(t, err)
	roundTripEntry, err := readFileEntry(bytes.NewBuffer(entryBytes))
	require.NoError(t, err)
	require.Equal(t, entry, roundTripEntry)
}
