package par2

import (
	"crypto/md5"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileDescriptionPacketRoundTrip(t *testing.T) {
	packet := fileDescriptionPacket{
		hash:         [md5.Size]byte{0x1, 0x2},
		sixteenKHash: [md5.Size]byte{0x3, 0x4},
		byteCount:    5,
		filename:     "subdir/file.txt",
	}
	fileID := computeFileID(packet.sixteenKHash, uint64(packet.byteCount), []byte(packet.filename))
	packetBytes, err := writeFileDescriptionPacket(fileID, packet)
	require.NoError(t, err)
	roundTripFileID, roundTripPacket, err := readFileDescriptionPacket(packetBytes)
	require.NoError(t, err)
	require.Equal(t, fileID, roundTripFileID)
	require.Equal(t, packet, roundTripPacket)
}
