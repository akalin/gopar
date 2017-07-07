package par2

import (
	"bytes"
	"io"
	"io/ioutil"
)

type fileIO interface {
	ReadFile(path string) ([]byte, error)
}

type defaultFileIO struct{}

func (io defaultFileIO) ReadFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

// A Decoder keeps track of all information needed to check the
// integrity of a set of data files, and possibly repair any
// missing/corrupted data files from the parity files (that usually
// end in .par2).
type Decoder struct {
	fileIO   fileIO
	delegate DecoderDelegate

	setID   [16]byte
	packets map[packetType][][]byte
}

// DecoderDelegate holds methods that are called during the decode
// process.
type DecoderDelegate interface {
	OnPacketLoad(packetType [16]byte, byteCount int)
	OnPacketSkip(setID [16]byte, packetType [16]byte, byteCount int)
}

func newDecoder(fileIO fileIO, delegate DecoderDelegate, indexFile string) (*Decoder, error) {
	indexBytes, err := fileIO.ReadFile(indexFile)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(indexBytes)
	packets := make(map[packetType][][]byte)
	var setID recoverySetID
	var hasSetID bool
	for {
		packetSetID, packetType, body, err := readNextPacket(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			// TODO: Relax this check.
			return nil, err
		}
		if hasSetID {
			if packetSetID != setID {
				delegate.OnPacketSkip(packetSetID, packetType, len(body))
				continue
			}
		} else {
			setID = packetSetID
			hasSetID = true
		}
		delegate.OnPacketLoad(packetType, len(body))
		packets[packetType] = append(packets[packetType], body)
	}

	return &Decoder{fileIO, delegate, setID, packets}, nil
}

// NewDecoder reads the given index file, which usually has a .par2
// extension.
func NewDecoder(delegate DecoderDelegate, indexFile string) (*Decoder, error) {
	return newDecoder(defaultFileIO{}, delegate, indexFile)
}
