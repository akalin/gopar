package par2

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"path"
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
	byteCount    int
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

	if h.Length == 0 {
		// This isn't specified by the spec, but par2 skips
		// empty files.
		//
		// TODO: Figure out if other programs create empty
		// files.
		return fileID{}, fileDescriptionPacket{}, errors.New("empty files not allowed")
	}

	filename := decodeNullPaddedASCIIString(filenameBytes)
	if path.IsAbs(filename) {
		// TODO: Allow this via an option.
		return fileID{}, fileDescriptionPacket{}, errors.New("absolute paths not allowed")
	}
	filename = path.Clean(filename)
	if filename[0] == '.' {
		return fileID{}, fileDescriptionPacket{}, errors.New("traversing outside of the current directory is not allowed")
	}

	// We have to do this since we load entire files in memory.
	//
	// TODO: Avoid loading entire files in memory, so that this
	// could work on 32-bit systems.
	maxInt := int(^uint(0) >> 1)
	if h.Length > uint64(maxInt) {
		return fileID{}, fileDescriptionPacket{}, errors.New("file length too big")
	}

	byteCount := int(h.Length)
	return h.FileID, fileDescriptionPacket{h.Hash, h.SixteenKHash, byteCount, filename}, nil
}
