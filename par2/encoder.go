package par2

import (
	"sort"

	"github.com/akalin/gopar/rsec16"
)

type encoderInputFileInfo struct {
	fileDescriptionPacket fileDescriptionPacket
	ifscPacket            ifscPacket
	dataShards            [][]uint16
}

// An Encoder keeps track of all information needed to create parity
// volumes for a set of data files, and write them out to parity files
// (that usually end in .par2).
type Encoder struct {
	fileIO   fileIO
	delegate EncoderDelegate

	filePaths []string

	sliceByteCount   int
	parityShardCount int

	recoverySet      []fileID
	recoverySetInfos map[fileID]encoderInputFileInfo

	parityShards [][]uint16
}

// EncoderDelegate holds methods that are called during the encode
// process.
type EncoderDelegate interface {
	OnDataFileLoad(i, n int, path string, byteCount int, err error)
	OnParityFileWrite(i, n int, path string, dataByteCount, byteCount int, err error)
}

func newEncoder(fileIO fileIO, delegate EncoderDelegate, filePaths []string, sliceByteCount, parityShardCount int) (*Encoder, error) {
	// TODO: Check filePaths, sliceByteCount, and parityShardCount.
	return &Encoder{fileIO, delegate, filePaths, sliceByteCount, parityShardCount, nil, nil, nil}, nil
}

// NewEncoder creates an encoder with the given list of file paths,
// and with the given number of intended parity volumes.
func NewEncoder(delegate EncoderDelegate, filePaths []string, sliceByteCount, parityShardCount int) (*Encoder, error) {
	return newEncoder(defaultFileIO{}, delegate, filePaths, sliceByteCount, parityShardCount)
}

// LoadFileData loads the file data into memory.
func (e *Encoder) LoadFileData() error {
	var recoverySet []fileID
	recoverySetInfos := make(map[fileID]encoderInputFileInfo)

	for i, path := range e.filePaths {
		data, err := e.fileIO.ReadFile(path)
		e.delegate.OnDataFileLoad(i+1, len(e.filePaths), path, len(data), err)
		if err != nil {
			return err
		}

		fileID, fileDescriptionPacket, ifscPacket, dataShards := computeDataFileInfo(e.sliceByteCount, path, data)
		recoverySet = append(recoverySet, fileID)
		recoverySetInfos[fileID] = encoderInputFileInfo{
			fileDescriptionPacket, ifscPacket, dataShards,
		}
	}

	sort.Slice(recoverySet, func(i, j int) bool {
		return fileIDLess(recoverySet[i], recoverySet[j])
	})

	e.recoverySet = recoverySet
	e.recoverySetInfos = recoverySetInfos
	return nil
}

// ComputeParityData computes the parity data for the files.
func (e *Encoder) ComputeParityData() error {
	var dataShards [][]uint16
	for _, fileID := range e.recoverySet {
		dataShards = append(dataShards, e.recoverySetInfos[fileID].dataShards...)
	}

	coder, err := rsec16.NewCoderPAR2Vandermonde(len(dataShards), e.parityShardCount)
	if err != nil {
		return err
	}

	e.parityShards = coder.GenerateParity(dataShards)
	return nil
}
