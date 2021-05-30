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
	FileID  [16]byte
	Hash    [16]byte
	Hash16k [16]byte
	Length  uint64
}

type fileDescriptionPacket struct {
	hash      [md5.Size]byte
	hash16k   [md5.Size]byte
	byteCount int
	filename  string
}

func computeFileID(hash16k [md5.Size]byte, byteCount uint64, filenameBytes []byte) fileID {
	var hashInput []byte
	hashInput = append(hashInput, hash16k[:]...)
	var byteCountBytes [8]byte
	binary.LittleEndian.PutUint64(byteCountBytes[:], byteCount)
	hashInput = append(hashInput, byteCountBytes[:]...)
	hashInput = append(hashInput, filenameBytes...)
	return md5.Sum(hashInput)
}

// TODO: It's theoretically possible for filename to contain
// backslashes -- check to see what par2cmdline does on Windows. We'd
// then have to handle par files where the filenames have backslashes
// but the current OS uses only forward slashes.
func checkFilename(filename string) error {
	// Filenames shouldn't be absolute, to preclude the repair process overwriting arbitrary
	// files.
	if path.IsAbs(filename) {
		return errors.New("absolute paths not allowed")
	}
	filename = path.Clean(filename)
	if filename[0] == '.' {
		return errors.New("traversing outside of the current directory is not allowed")
	}
	return nil
}

func readFileDescriptionPacket(body []byte) (fileID, fileDescriptionPacket, error) {
	buf := bytes.NewBuffer(body)

	var h fileDescriptionPacketHeader
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return fileID{}, fileDescriptionPacket{}, err
	}

	filenameBytes := buf.Bytes()
	computedFileID := computeFileID(h.Hash16k, h.Length, nullTerminate(filenameBytes))
	if computedFileID != h.FileID {
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
	err = checkFilename(filename)
	if err != nil {
		return fileID{}, fileDescriptionPacket{}, err
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
	return h.FileID, fileDescriptionPacket{h.Hash, h.Hash16k, byteCount, filename}, nil
}

func writeFileDescriptionPacket(fileID fileID, packet fileDescriptionPacket) ([]byte, error) {
	if packet.byteCount <= 0 {
		return nil, errors.New("invalid byte count")
	}

	err := checkFilename(packet.filename)
	if err != nil {
		return nil, err
	}

	filenameBytes, err := encodeASCIIString(packet.filename)
	if err != nil {
		return nil, err
	}

	byteCount := uint64(packet.byteCount)
	computedFileID := computeFileID(packet.hash16k, byteCount, filenameBytes)
	if computedFileID != fileID {
		return nil, errors.New("file ID mismatch")
	}

	buf := bytes.NewBuffer(nil)

	h := fileDescriptionPacketHeader{
		FileID:  fileID,
		Hash:    packet.hash,
		Hash16k: packet.hash16k,
		Length:  byteCount,
	}
	err = binary.Write(buf, binary.LittleEndian, h)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(filenameBytes)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
