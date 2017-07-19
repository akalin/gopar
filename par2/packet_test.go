package par2

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPacketHeaderRoundTrip(t *testing.T) {
	h := packetHeader{
		Magic:         expectedMagic,
		Length:        100,
		Hash:          [16]byte{0x1},
		RecoverySetID: [16]byte{0x2},
		Type:          [16]byte{0x3},
	}

	buf := bytes.NewBuffer(nil)
	err := writePacketHeader(buf, h)
	require.NoError(t, err)
	roundTripHeader, err := readPacketHeader(buf)
	require.NoError(t, err)
	require.Equal(t, h, roundTripHeader)
	require.Equal(t, 0, buf.Len())
}

type typedPacket struct {
	packetType packetType
	body       []byte
}

func TestPacketsRoundTrip(t *testing.T) {
	packets := []typedPacket{
		{packetType{0x1}, []byte{0x2, 0x3, 0x0, 0x1}},
		{packetType{0x4}, []byte{0x5, 0x6, 0x1, 0x0}},
	}

	buf := bytes.NewBuffer(nil)
	setID := recoverySetID{0x5}
	for _, p := range packets {
		err := writeNextPacket(buf, setID, p.packetType, p.body)
		require.NoError(t, err)
	}

	var roundTripPackets []typedPacket
	for {
		roundTripSetID, packetType, body, err := readNextPacket(buf)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		require.Equal(t, setID, roundTripSetID)
		roundTripPackets = append(roundTripPackets, typedPacket{packetType, body})
	}

	require.Equal(t, packets, roundTripPackets)
	require.Equal(t, 0, buf.Len())
}
