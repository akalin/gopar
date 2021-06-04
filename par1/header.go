package par1

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

type versionNumber uint64

func makeVersionNumber(version uint32) versionNumber {
	return versionNumber(version)
}

func (n versionNumber) version() uint32 {
	return uint32(n & 0xffffffff)
}

func (n versionNumber) id() uint32 {
	return uint32(n >> 32)
}

func (n versionNumber) String() string {
	return fmt.Sprintf("versionNumber{version:%08x, id:%08x}", n.version(), n.id())
}

type header struct {
	ID             [8]byte
	VersionNumber  versionNumber
	ControlHash    [16]byte
	SetHash        [16]byte
	VolumeNumber   uint64
	FileCount      uint64
	FileListOffset uint64
	FileListBytes  uint64
	DataOffset     uint64
	DataBytes      uint64
}

const (
	headerByteCount = 96
)

func (h header) String() string {
	return fmt.Sprintf("header{VersionNumber:%s, ControlHash:%x, SetHash:%x, VolumeNumber:%d, FileCount:%d, FileListOffset:%d, FileListBytes:%d, DataOffset:%d, DataBytes:%d}",
		h.VersionNumber, h.ControlHash, h.SetHash, h.VolumeNumber,
		h.FileCount, h.FileListOffset, h.FileListBytes, h.DataOffset, h.DataBytes)
}

var expectedID = [8]byte{'P', 'A', 'R'}

const expectedVersion uint32 = 0x00010000

const expectedFileListOffset uint64 = 0x00000060

func readHeader(headerBytes [headerByteCount]byte) (header, error) {
	r := bytes.NewReader(headerBytes[:])
	var h header
	err := binary.Read(r, binary.LittleEndian, &h)
	if err != nil {
		return header{}, err
	}
	if r.Len() != 0 {
		return header{}, errors.New("headerBytes reader unexpectedly non-empty")
	}

	if h.ID != expectedID {
		return header{}, errors.New("unexpected ID string")
	}

	if h.VersionNumber.version() != expectedVersion {
		return header{}, errors.New("unexpected version")
	}

	if h.FileListOffset != expectedFileListOffset {
		return header{}, errors.New("unexpected file list offset")
	}

	return h, nil
}

func writeHeader(h header) ([headerByteCount]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := binary.Write(buf, binary.LittleEndian, h)
	if err != nil {
		return [headerByteCount]byte{}, err
	}
	if buf.Len() != headerByteCount {
		return [headerByteCount]byte{}, errors.New("unexpected buf length")
	}
	var headerBytes [headerByteCount]byte
	copy(headerBytes[:], buf.Bytes())
	return headerBytes, nil
}
