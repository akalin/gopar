package par2

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"reflect"
)

var ifscPacketType = packetType{'P', 'A', 'R', ' ', '2', '.', '0', '\x00', 'I', 'F', 'S', 'C'}

type checksumPair struct {
	MD5   [md5.Size]byte
	CRC32 [4]byte
}

type ifscPacket struct {
	checksumPairs []checksumPair
}

func readIFSCPacket(body []byte) (fileID, ifscPacket, error) {
	buf := bytes.NewBuffer(body)

	var id fileID
	err := binary.Read(buf, binary.LittleEndian, &id)
	if err != nil {
		return fileID{}, ifscPacket{}, err
	}

	checksumPairSize := int(reflect.TypeOf(checksumPair{}).Size())
	if buf.Len() == 0 || buf.Len()%checksumPairSize != 0 {
		return fileID{}, ifscPacket{}, errors.New("invalid size")
	}
	checksumPairs := make([]checksumPair, buf.Len()/checksumPairSize)
	err = binary.Read(buf, binary.LittleEndian, checksumPairs)
	if err != nil {
		return fileID{}, ifscPacket{}, err
	}

	return id, ifscPacket{checksumPairs}, nil
}

func writeIFSCPacket(id fileID, packet ifscPacket) ([]byte, error) {
	if len(packet.checksumPairs) == 0 {
		return nil, errors.New("no checksum pairs to write")
	}

	buf := bytes.NewBuffer(nil)

	err := binary.Write(buf, binary.LittleEndian, id)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, packet.checksumPairs)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
