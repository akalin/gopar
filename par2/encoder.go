package par2

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"

	"github.com/akalin/gopar/fs"
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
	fs       fs.FS
	delegate EncoderDelegate

	basePath     string
	relFilePaths []string

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

func newEncoder(fs fs.FS, delegate EncoderDelegate, basePath string, filePaths []string, sliceByteCount, parityShardCount, numGoroutines int) (*Encoder, error) {
	if !filepath.IsAbs(basePath) {
		return nil, errors.New("basePath must be absolute")
	}

	relFilePaths := make([]string, len(filePaths))
	for i, path := range filePaths {
		var relPath string
		if !filepath.IsAbs(path) {
			return nil, errors.New("all elements of filePaths must be absolute")
		}
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return nil, err
		}
		if relPath[0] == '.' {
			return nil, errors.New("data files must lie in basePath")
		}
		relFilePaths[i] = relPath
	}

	// TODO: Check parityShardCount.
	if sliceByteCount == 0 || sliceByteCount%4 != 0 {
		return nil, errors.New("invalid slice byte count")
	}
	return &Encoder{fs, delegate, basePath, relFilePaths, sliceByteCount, parityShardCount, numGoroutines, nil, nil, nil}, nil
}

// NewEncoder creates an encoder with the given list of file paths,
// and with the given number of intended parity volumes. basePath must
// be absolute. Elements of filePaths must be absolute, and must also
// lie in basePath.
func NewEncoder(delegate EncoderDelegate, basePath string, filePaths []string, sliceByteCount, parityShardCount, numGoroutines int) (*Encoder, error) {
	return newEncoder(fs.MakeDefaultFS(), delegate, basePath, filePaths, sliceByteCount, parityShardCount, numGoroutines)
}

// LoadFileData loads the file data into memory.
func (e *Encoder) LoadFileData() error {
	var recoverySet []fileID
	recoverySetInfos := make(map[fileID]encoderInputFileInfo)

	for i, relPath := range e.relFilePaths {
		path := filepath.Join(e.basePath, relPath)
		data, err := func() ([]byte, error) {
			readStream, err := e.fs.GetReadStream(path)
			if err != nil {
				return nil, err
			}
			return fs.ReadAndClose(readStream)
		}()
		e.delegate.OnDataFileLoad(i+1, len(e.relFilePaths), path, len(data), err)
		if err != nil {
			return err
		}

		fileID, fileDescriptionPacket, ifscPacket, dataShards := computeDataFileInfo(e.sliceByteCount, relPath, data)
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
	err = func() error {
		writeStream, err := e.fs.GetWriteStream(filename)
		if err != nil {
			return err
		}
		return fs.WriteAndClose(writeStream, parityFileBytes)
	}()
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
		err = func() error {
			writeStream, err := e.fs.GetWriteStream(filename)
			if err != nil {
				return err
			}
			return fs.WriteAndClose(writeStream, recoveryFileBytes)
		}()
		e.delegate.OnRecoveryFileWrite(i, volumeCount, e.parityShardCount, filename, len(recoveryFileBytes)-len(parityFileBytes), len(recoveryFileBytes), err)
		if err != nil {
			return err
		}

		i += volumeCount
		volumeCount *= 2
	}

	return nil
}
