package par2

import (
	"encoding/binary"
	"errors"
	"math"
)

var recoveryPacketType = packetType{'P', 'A', 'R', ' ', '2', '.', '0', '\x00', 'R', 'e', 'c', 'v', 'S', 'l', 'i', 'c'}

type exponent uint16

type recoveryPacket struct {
	data []uint16
}

func readRecoveryPacket(body []byte) (exponent, recoveryPacket, error) {
	if len(body) == 0 || len(body)%4 != 0 {
		return 0, recoveryPacket{}, errors.New("invalid recovery data byte count")
	}

	exp := binary.LittleEndian.Uint32(body)
	if exp > math.MaxUint16 {
		return 0, recoveryPacket{}, errors.New("exponent out of range")
	}

	return exponent(exp), recoveryPacket{byteToUint16LEArray(body[4:])}, nil
}
