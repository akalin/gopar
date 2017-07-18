package par2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMainPacketRoundTrip(t *testing.T) {
	packet := mainPacket{
		sliceByteCount: 16,
		recoverySet:    []fileID{{0x1}, {0x2}},
		nonRecoverySet: []fileID{{0x3}, {0x4}},
	}
	packetBytes, err := writeMainPacket(packet)
	require.NoError(t, err)
	roundTripPacket, err := readMainPacket(packetBytes)
	require.NoError(t, err)
	require.Equal(t, packet, roundTripPacket)
}
