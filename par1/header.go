package par1

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type header struct {
	ID             [8]byte
	VersionNumber  uint64
	ControlHash    [16]byte
	SetHash        [16]byte
	VolumeNumber   uint64
	FileCount      uint64
	FileListOffset uint64
	FileListBytes  uint64
	DataOffset     uint64
	DataBytes      uint64
}

var expectedID = [8]byte{'P', 'A', 'R'}

const expectedVersion uint64 = 0x00010000

const expectedFileListOffset uint64 = 0x00000060

func readHeader(buf *bytes.Buffer) (header, error) {
	var h header
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return header{}, err
	}

	if h.ID != expectedID {
		return header{}, errors.New("unexpected ID string")
	}

	if (h.VersionNumber & 0xffffffff) != expectedVersion {
		return header{}, errors.New("unexpected version")
	}

	if h.FileListOffset != expectedFileListOffset {
		return header{}, errors.New("unexpected file list offset")
	}

	return h, nil
}
