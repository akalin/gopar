package par2

import (
	"encoding/binary"
	"math"

	"github.com/akalin/gopar/errorcode"
)

var recoveryPacketType = packetType{'P', 'A', 'R', ' ', '2', '.', '0', '\x00', 'R', 'e', 'c', 'v', 'S', 'l', 'i', 'c'}

type exponent uint16

type recoveryPacket struct {
	data []byte
}

func readRecoveryPacket(body []byte) (exponent, recoveryPacket, error) {
	if len(body) == 0 || len(body)%4 != 0 {
		return 0, recoveryPacket{}, errorcode.InvalidRecoveryDataByteCount
	}

	exp := binary.LittleEndian.Uint32(body)
	if exp > math.MaxUint16 {
		return 0, recoveryPacket{}, errorcode.ExponentOutOfRange
	}

	return exponent(exp), recoveryPacket{body[4:]}, nil
}

func writeRecoveryPacket(exp exponent, packet recoveryPacket) ([]byte, error) {
	if len(packet.data) == 0 || len(packet.data)%4 != 0 {
		return nil, errorcode.InvalidRecoveryDataByteCount
	}

	var expBytes [4]byte
	binary.LittleEndian.PutUint32(expBytes[:], uint32(exp))
	return append(expBytes[:], packet.data...), nil
}
