package par2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreatorPacketRoundTrip(t *testing.T) {
	clientID := "some par program"
	packetBytes, err := writeCreatorPacket(clientID)
	require.NoError(t, err)
	roundTripClientID := readCreatorPacket(packetBytes)
	require.Equal(t, clientID, roundTripClientID)
}
