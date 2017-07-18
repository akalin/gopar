package par2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecoveryPacketRoundTrip(t *testing.T) {
	var exp exponent = 0xff00
	packet := recoveryPacket{
		data: []uint16{0xffff, 0x0000, 0xabcd, 0x0001},
	}
	packetBytes, err := writeRecoveryPacket(exp, packet)
	require.NoError(t, err)
	roundTripExp, roundTripPacket, err := readRecoveryPacket(packetBytes)
	require.NoError(t, err)
	require.Equal(t, exp, roundTripExp)
	require.Equal(t, packet, roundTripPacket)
}
