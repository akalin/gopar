package par1

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeaderRoundTrip(t *testing.T) {
	h := header{
		ID:             expectedID,
		VersionNumber:  expectedVersion,
		ControlHash:    [16]byte{0x1, 0x2},
		SetHash:        [16]byte{0x3, 0x4},
		VolumeNumber:   5,
		FileCount:      6,
		FileListOffset: expectedFileListOffset,
		FileListBytes:  100,
		DataOffset:     expectedFileListOffset + 100,
		DataBytes:      200,
	}

	headerBytes, err := writeHeader(h)
	require.NoError(t, err)
	roundTripHeader, err := readHeader(bytes.NewBuffer(headerBytes))
	require.NoError(t, err)
	require.Equal(t, h, roundTripHeader)
}
