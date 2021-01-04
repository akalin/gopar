package par2

import (
	"encoding/binary"
	"errors"
	"math"
)

var recoveryPacketType = packetType{'P', 'A', 'R', ' ', '2', '.', '0', '\x00', 'R', 'e', 'c', 'v', 'S', 'l', 'i', 'c'}

type exponent uint16

type recoveryPacket struct {
	data []byte
}

func readRecoveryPacket(body []byte) (exponent, recoveryPacket, error) {
	if len(body) == 0 || len(body)%4 != 0 {
		return 0, recoveryPacket{}, errors.New("invalid recovery data byte count")
	}

	exp := binary.LittleEndian.Uint32(body)
	if exp > math.MaxUint16 {
		return 0, recoveryPacket{}, errors.New("exponent out of range")
	}

	return exponent(exp), recoveryPacket{body[4:]}, nil
}

func writeRecoveryPacket(exp exponent, packet recoveryPacket) ([]byte, error) {
	if len(packet.data) == 0 || len(packet.data)%4 != 0 {
		return nil, errors.New("invalid recovery data byte count")
	}

	var expBytes [4]byte
	binary.LittleEndian.PutUint32(expBytes[:], uint32(exp))
	return append(expBytes[:], packet.data...), nil
}
