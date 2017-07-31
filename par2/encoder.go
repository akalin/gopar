package par2

import (
	"fmt"
	"path"
	"sort"

	"github.com/akalin/gopar/rsec16"
)

type encoderInputFileInfo struct {
	fileDescriptionPacket fileDescriptionPacket
	ifscPacket            ifscPacket
	dataShards            [][]byte
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

	numGoroutines int

	recoverySet      []fileID
	recoverySetInfos map[fileID]encoderInputFileInfo

	parityShards [][]byte
}

// EncoderDelegate holds methods that are called during the encode
// process.
type EncoderDelegate interface {
	OnDataFileLoad(i, n int, path string, byteCount int, err error)
	OnIndexFileWrite(path string, byteCount int, err error)
	OnRecoveryFileWrite(start, count, total int, path string, dataByteCount, byteCount int, err error)
}

func newEncoder(fileIO fileIO, delegate EncoderDelegate, filePaths []string, sliceByteCount, parityShardCount, numGoroutines int) (*Encoder, error) {
	// TODO: Check filePaths, sliceByteCount, and parityShardCount.
	return &Encoder{fileIO, delegate, filePaths, sliceByteCount, parityShardCount, numGoroutines, nil, nil, nil}, nil
}

// NewEncoder creates an encoder with the given list of file paths,
// and with the given number of intended parity volumes.
func NewEncoder(delegate EncoderDelegate, filePaths []string, sliceByteCount, parityShardCount, numGoroutines int) (*Encoder, error) {
	return newEncoder(defaultFileIO{}, delegate, filePaths, sliceByteCount, parityShardCount, numGoroutines)
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
	var dataShards [][]byte
	for _, fileID := range e.recoverySet {
		dataShards = append(dataShards, e.recoverySetInfos[fileID].dataShards...)
	}

	coder, err := rsec16.NewCoderPAR2Vandermonde(len(dataShards), e.parityShardCount, e.numGoroutines)
	if err != nil {
		return err
	}

	e.parityShards = coder.GenerateParity(dataShards)
	return nil
}

const clientID = "gopar"

func (e *Encoder) Write(indexPath string) error {
	mainPacket := mainPacket{
		sliceByteCount: e.sliceByteCount,
		recoverySet:    e.recoverySet,
	}

	fileDescriptionPackets := make(map[fileID]fileDescriptionPacket)
	ifscPackets := make(map[fileID]ifscPacket)
	for fileID, info := range e.recoverySetInfos {
		fileDescriptionPackets[fileID] = info.fileDescriptionPacket
		ifscPackets[fileID] = info.ifscPacket
	}

	parityFile := file{
		clientID:               clientID,
		mainPacket:             &mainPacket,
		fileDescriptionPackets: fileDescriptionPackets,
		ifscPackets:            ifscPackets,
	}

	_, parityFileBytes, err := writeFile(parityFile)
	if err != nil {
		return err
	}

	var base string
	ext := path.Ext(indexPath)
	base = indexPath[:len(indexPath)-len(ext)]

	filename := base + ".par2"
	err = e.fileIO.WriteFile(filename, parityFileBytes)
	e.delegate.OnIndexFileWrite(filename, len(parityFileBytes), err)
	if err != nil {
		return err
	}

	volumeCount := 1
	for i := 0; i < e.parityShardCount; {
		recoveryFile := parityFile
		recoveryFile.recoveryPackets = make(map[exponent]recoveryPacket, volumeCount)
		if i+volumeCount > e.parityShardCount {
			volumeCount = e.parityShardCount - i
		}
		for j := 0; j < volumeCount; j++ {
			recoveryFile.recoveryPackets[exponent(i+j)] = recoveryPacket{data: e.parityShards[i+j]}
		}

		_, recoveryFileBytes, err := writeFile(recoveryFile)
		if err != nil {
			return err
		}

		// TODO: Figure out how to handle when either i or
		// volumeCount is >= 100.
		filename := fmt.Sprintf("%s.vol%02d+%02d.par2", base, i, volumeCount)
		err = e.fileIO.WriteFile(filename, recoveryFileBytes)
		e.delegate.OnRecoveryFileWrite(i, volumeCount, e.parityShardCount, filename, len(recoveryFileBytes)-len(parityFileBytes), len(recoveryFileBytes), err)
		if err != nil {
			return err
		}

		i += volumeCount
		volumeCount *= 2
	}

	return nil
}
