package par2

import (
	"bytes"
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
