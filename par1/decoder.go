package par1

import (
	"bytes"
	"encoding/binary"
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

	// TODO: Sanity-check header fields.

	return &Decoder{header}, nil
}
