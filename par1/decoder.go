package par1

import (
	"bytes"
	"io/ioutil"
)

// A Decoder keeps track of all information needed to check the
// integrity of a set of data files, and possibly reconstruct any
// missing/corrupted data files from the parity files (.P00, .P01,
// etc.).
type Decoder struct {
	header  header
	entries []fileEntry
}

// NewDecoder reads the given index file, which usually has a .PAR
// extension.
func NewDecoder(indexFile string) (*Decoder, error) {
	indexBytes, err := ioutil.ReadFile(indexFile)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(indexBytes)

	header, err := readHeader(buf)
	if err != nil {
		return nil, err
	}

	entries := make([]fileEntry, header.FileCount)
	for i := uint64(0); i < header.FileCount; i++ {
		var err error
		entries[i], err = readFileEntry(buf)
		if err != nil {
			return nil, err
		}
	}

	return &Decoder{header, entries}, nil
}
