package par2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecoveryPacketRoundTrip(t *testing.T) {
	var exp exponent = 0xff00
	packet := recoveryPacket{
		data: []byte{0xff, 0xff, 0x00, 0x00, 0xcd, 0xab, 0x01, 0x00},
	}
	packetBytes, err := writeRecoveryPacket(exp, packet)
	require.NoError(t, err)
	roundTripExp, roundTripPacket, err := readRecoveryPacket(packetBytes)
	require.NoError(t, err)
	require.Equal(t, exp, roundTripExp)
	require.Equal(t, packet, roundTripPacket)
}
