package par1

import "github.com/klauspost/reedsolomon"

// An Encoder keeps track of all information needed to create parity
// volumes for a set of data files, and write them out to parity files
// (.PAR, .P00, .P01, etc.).
type Encoder struct {
	fileIO fileIO

	filePaths   []string
	volumeCount int

	shardByteCount int
	fileData       [][]byte
	parityData     [][]byte
}

func newEncoder(fileIO fileIO, filePaths []string, volumeCount int) (*Encoder, error) {
	// TODO: Check len(filePaths) and volumeCount.
	return &Encoder{fileIO, filePaths, volumeCount, 0, nil, nil}, nil
}

// NewEncoder creates an encoder with the given list of file paths,
// and with the given number of intended parity volumes.
func NewEncoder(filePaths []string, volumeCount int) (*Encoder, error) {
	return newEncoder(defaultFileIO{}, filePaths, volumeCount)
}

// LoadFileData loads the file data into memory.
func (e *Encoder) LoadFileData() error {
	shardByteCount := 0
	fileData := make([][]byte, len(e.filePaths))
	for i, path := range e.filePaths {
		var err error
		fileData[i], err = e.fileIO.ReadFile(path)
		if err != nil {
			return err
		}

		if len(fileData[i]) > shardByteCount {
			shardByteCount = len(fileData[i])
		}
	}

	e.shardByteCount = shardByteCount
	e.fileData = fileData
	return nil
}

func (e *Encoder) buildShards() [][]byte {
	shards := make([][]byte, len(e.fileData)+e.volumeCount)
	for i, data := range e.fileData {
		padding := make([]byte, e.shardByteCount-len(data))
		shards[i] = append(data, padding...)
	}

	for i := 0; i < e.volumeCount; i++ {
		shards[len(e.fileData)+i] = make([]byte, e.shardByteCount)
	}

	return shards
}

// ComputeParityData computes the parity data for the files.
func (e *Encoder) ComputeParityData() error {
	shards := e.buildShards()

	rs, err := reedsolomon.New(len(e.fileData), e.volumeCount, reedsolomon.WithPAR1Matrix())
	if err != nil {
		return err
	}

	err = rs.Encode(shards)
	if err != nil {
		return err
	}

	e.parityData = shards[len(e.fileData):]
	return nil
}
