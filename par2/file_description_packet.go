package par2

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
)

var fileDescriptionPacketType = packetType{'P', 'A', 'R', ' ', '2', '.', '0', '\x00', 'F', 'i', 'l', 'e', 'D', 'e', 's', 'c'}

type fileDescriptionPacketHeader struct {
	FileID       [16]byte
	Hash         [16]byte
	SixteenKHash [16]byte
	Length       uint64
}

type fileDescriptionPacket struct {
	hash         [md5.Size]byte
	sixteenKHash [md5.Size]byte
	byteCount    uint64
	filename     string
}

func readFileDescriptionPacket(body []byte) (fileID, fileDescriptionPacket, error) {
	buf := bytes.NewBuffer(body)

	var h fileDescriptionPacketHeader
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return fileID{}, fileDescriptionPacket{}, err
	}

	filenameBytes := buf.Bytes()

	var hashInput []byte
	hashInput = append(hashInput, h.SixteenKHash[:]...)
	var lengthBytes [8]byte
	binary.LittleEndian.PutUint64(lengthBytes[:], h.Length)
	hashInput = append(hashInput, lengthBytes[:]...)
	hashInput = append(hashInput, nullTerminate(filenameBytes)...)

	if md5.Sum(hashInput) != h.FileID {
		return fileID{}, fileDescriptionPacket{}, errors.New("file ID mismatch")
	}

	// TODO: Make sure filename does not point outside of the
	// current dir.
	filename := decodeNullPaddedASCIIString(filenameBytes)

	return h.FileID, fileDescriptionPacket{h.Hash, h.SixteenKHash, h.Length, filename}, nil
}
