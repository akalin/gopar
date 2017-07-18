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

func writeRecoveryPacket(exp exponent, packet recoveryPacket) ([]byte, error) {
	// Remember that packet.data is []uint16, so its size must be
	// a multiple of 2 in order for the byte count to be a
	// multiple of 4.
	if len(packet.data) == 0 || len(packet.data)%2 != 0 {
		return nil, errors.New("invalid recovery data byte count")
	}

	var expBytes [4]byte
	binary.LittleEndian.PutUint32(expBytes[:], uint32(exp))
	return append(expBytes[:], uint16LEToByteArray(packet.data)...), nil
}
