package par1

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/ioutil"
)

// A Decoder keeps track of all information needed to check the
// integrity of a set of data files, and possibly reconstruct any
// missing/corrupted data files from the parity files (.P00, .P01,
// etc.).
type Decoder struct {
	header header
}

// NewDecoder reads the given index file, which usually has a .PAR
// extension.
func NewDecoder(indexFile string) (*Decoder, error) {
	indexBytes, err := ioutil.ReadFile(indexFile)
	if err != nil {
		return nil, err
	}

	indexBuf := bytes.NewBuffer(indexBytes)

	var header header
	err = binary.Read(indexBuf, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}

	if header.ID != expectedID {
		return nil, errors.New("unexpected ID string")
	}

	if (header.VersionNumber & 0xffffffff) != expectedVersion {
		return nil, errors.New("unexpected version")
	}

	// TODO: Check header.ControlHash and header.SetHash.

	if header.VolumeNumber != 0 {
		return nil, errors.New("not a PAR file")
	}

	if header.FileListOffset != expectedFileListOffset {
		return nil, errors.New("unexpected file list offset")
	}

	// TODO: Check count of files saved in volume set, and other
	// offsets and bytes.

	return &Decoder{header}, nil
}
