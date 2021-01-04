package par2

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"reflect"
)

type packetHeader struct {
	Magic         [8]byte
	Length        uint64
	Hash          [16]byte
	RecoverySetID [16]byte
	Type          [16]byte
}

var expectedMagic = [8]byte{'P', 'A', 'R', '2', '\x00', 'P', 'K', 'T'}

func sizeOfPacketHeader() uint64 {
	return uint64(reflect.TypeOf(packetHeader{}).Size())
}

func checkPacketHeader(h packetHeader) error {
	if h.Magic != expectedMagic {
		return errors.New("unexpected magic string")
	}

	if h.Length < sizeOfPacketHeader() || h.Length%4 != 0 {
		return errors.New("invalid length")
	}

	return nil
}

func readPacketHeader(buf *bytes.Buffer) (packetHeader, error) {
	var h packetHeader
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return packetHeader{}, err
	}

	err = checkPacketHeader(h)
	if err != nil {
		return packetHeader{}, err
	}

	return h, nil
}

func writePacketHeader(buf *bytes.Buffer, h packetHeader) error {
	err := checkPacketHeader(h)
	if err != nil {
		return err
	}

	return binary.Write(buf, binary.LittleEndian, h)
}

type recoverySetID [16]byte
type packetType [16]byte

func computePacketHash(setID recoverySetID, packetType packetType, body []byte) [md5.Size]byte {
	var hashInput []byte
	hashInput = append(hashInput, setID[:]...)
	hashInput = append(hashInput, packetType[:]...)
	hashInput = append(hashInput, body...)
	return md5.Sum(hashInput)
}

func readNextPacket(buf *bytes.Buffer) (recoverySetID, packetType, []byte, error) {
	h, err := readPacketHeader(buf)
	if err != nil {
		return [16]byte{}, packetType{}, nil, err
	}

	// TODO: Handle overflow.
	bodyLength := int(h.Length - sizeOfPacketHeader())
	body := buf.Next(bodyLength)
	if len(body) != bodyLength {
		return [16]byte{}, packetType{}, nil, errors.New("could not read body")
	}

	if computePacketHash(h.RecoverySetID, h.Type, body) != h.Hash {
		return [16]byte{}, packetType{}, nil, errors.New("hash mismatch")
	}

	bodyCopy := make([]byte, len(body))
	copy(bodyCopy, body)
	return h.RecoverySetID, h.Type, bodyCopy, nil
}

func writeNextPacket(buf *bytes.Buffer, setID recoverySetID, packetType packetType, body []byte) error {
	// TODO: Handle overflow.
	length := sizeOfPacketHeader() + uint64(len(body))
	h := packetHeader{
		Magic:         expectedMagic,
		Length:        length,
		Hash:          computePacketHash(setID, packetType, body),
		RecoverySetID: setID,
		Type:          packetType,
	}
	err := writePacketHeader(buf, h)
	if err != nil {
		return err
	}

	_, err = buf.Write(body)
	return err
}
