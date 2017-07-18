package par2

import (
	"crypto/md5"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIFSCPacketRoundTrip(t *testing.T) {
	packet := ifscPacket{
		checksumPairs: []checksumPair{
			{
				[md5.Size]byte{0xa, 0xb},
				[4]byte{0x1, 0x2, 0x3, 0x4},
			},
			{
				[md5.Size]byte{0x1a, 0x1b},
				[4]byte{0x5, 0x6, 0x7, 0x8},
			},
		},
	}
	fileID := fileID{0x1, 0x2}
	packetBytes, err := writeIFSCPacket(fileID, packet)
	require.NoError(t, err)
	roundTripFileID, roundTripPacket, err := readIFSCPacket(packetBytes)
	require.NoError(t, err)
	require.Equal(t, fileID, roundTripFileID)
	require.Equal(t, packet, roundTripPacket)
}
