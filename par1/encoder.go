package par1

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"

	"github.com/akalin/gopar/fs"
	"github.com/akalin/gopar/hashutil"
	"github.com/klauspost/reedsolomon"
)

type fileInfo struct {
	readStream fs.ReadStream
	hasher     *hashutil.MD5HasherWith16k
}

func makeFileInfo(fs fs.FS, path string) (fileInfo, error) {
	readStream, err := fs.GetReadStream(path)
	if err != nil {
		return fileInfo{}, err
	}
	hasher := hashutil.MakeMD5HasherWith16k()
	readStream = hashutil.TeeReadStream(readStream, hasher)
	return fileInfo{readStream, hasher}, nil
}

func (f fileInfo) Close() error {
	if f.readStream != nil {
		return f.readStream.Close()
	}
	return nil
}

type fileInfoList []fileInfo

func (l fileInfoList) Close() error {
	var firstErr error
	for _, info := range l {
		err := info.Close()
		if firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// An Encoder keeps track of all information needed to create parity
// volumes for a set of data files, and write them out to parity files
// (.PAR, .P00, .P01, etc.).
type Encoder struct {
	fs       fs.FS
	delegate EncoderDelegate

	filePaths   []string
	volumeCount int

	shardByteCount int
	fileData       fileInfoList
	parityData     [][]byte
}

// EncoderDelegate holds methods that are called during the encode
// process.
type EncoderDelegate interface {
	OnDataFileLoad(i, n int, path string, byteCount int, err error)
	OnVolumeFileWrite(i, n int, path string, dataByteCount, byteCount int, err error)
}

func newEncoder(fs fs.FS, delegate EncoderDelegate, filePaths []string, volumeCount int) (*Encoder, error) {
	filenames := make(map[string]bool)
	for _, p := range filePaths {
		filename := filepath.Base(p)
		if filenames[filename] {
			return nil, errors.New("filename collision")
		}
		filenames[filename] = true
	}
	// TODO: Check len(filePaths) and volumeCount.
	return &Encoder{fs, delegate, filePaths, volumeCount, 0, nil, nil}, nil
}

// NewEncoder creates an encoder with the given list of file paths,
// and with the given number of intended parity volumes.
func NewEncoder(delegate EncoderDelegate, filePaths []string, volumeCount int) (*Encoder, error) {
	return newEncoder(fs.MakeDefaultFS(), delegate, filePaths, volumeCount)
}

// LoadFileData loads the file data into memory.
func (e *Encoder) LoadFileData() (err error) {
	shardByteCount := 0
	fileData := make(fileInfoList, len(e.filePaths))
	defer func() {
		if err != nil {
			_ = fileData.Close()
		}
	}()
	for i, path := range e.filePaths {
		fileInfo, err := makeFileInfo(e.fs, path)
		byteCount := fileInfo.readStream.ByteCount()
		e.delegate.OnDataFileLoad(i+1, len(e.filePaths), path, int(byteCount), err)
		if err != nil {
			return err
		}
		fileData[i] = fileInfo
		if int(byteCount) > shardByteCount {
			shardByteCount = int(byteCount)
		}
	}

	e.shardByteCount = shardByteCount
	e.fileData = fileData
	return nil
}

func (e *Encoder) buildShards() ([][]byte, error) {
	shards := make([][]byte, len(e.fileData)+e.volumeCount)
	for i, info := range e.fileData {
		data, err := fs.ReadAll(info.readStream)
		if err != nil {
			return nil, err
		}
		byteCount := info.readStream.ByteCount()
		padding := make([]byte, e.shardByteCount-int(byteCount))
		shards[i] = append(data, padding...)
	}

	for i := 0; i < e.volumeCount; i++ {
		shards[len(e.fileData)+i] = make([]byte, e.shardByteCount)
	}

	return shards, nil
}

// ComputeParityData computes the parity data for the files.
func (e *Encoder) ComputeParityData() error {
	shards, err := e.buildShards()
	if err != nil {
		return err
	}

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

func (e *Encoder) Write(indexPath string) error {
	var entries []fileEntry
	for i, k := range e.filePaths {
		info := e.fileData[i]
		var status fileEntryStatus
		status.setSavedInVolumeSet(true)
		hash, hash16k := info.hasher.Hashes()
		entry := fileEntry{
			header: fileEntryHeader{
				Status:    status,
				FileBytes: uint64(info.readStream.ByteCount()),
				Hash:      hash,
				Hash16k:   hash16k,
			},
			filename: filepath.Base(k),
		}
		entries = append(entries, entry)
	}

	vTemplate := volume{
		header: header{
			ID:            expectedID,
			VersionNumber: makeVersionNumber(expectedVersion),
			SetHash:       computeSetHash(entries),
		},
		entries: entries,
	}

	indexVolume := vTemplate
	indexVolume.header.VolumeNumber = 0
	indexVolumeBytes, err := writeVolume(indexVolume)
	if err != nil {
		return err
	}

	// TODO: Sanity-check indexPath.
	ext := path.Ext(indexPath)
	base := indexPath[:len(indexPath)-len(ext)]

	realIndexPath := base + ".par"
	err = func() error {
		writeStream, err := e.fs.GetWriteStream(realIndexPath)
		if err != nil {
			return err
		}
		return fs.WriteAndClose(writeStream, indexVolumeBytes)
	}()
	e.delegate.OnVolumeFileWrite(0, len(e.parityData), realIndexPath, len(indexVolume.data), len(indexVolumeBytes), err)
	if err != nil {
		return err
	}

	for i, parityShard := range e.parityData {
		vol := vTemplate
		vol.header.VolumeNumber = uint64(i + 1)
		vol.data = parityShard
		volBytes, err := writeVolume(vol)
		if err != nil {
			return err
		}

		// TODO: Handle more than 99 parity files.
		volumePath := fmt.Sprintf("%s.p%02d", base, i+1)
		err = func() error {
			writeStream, err := e.fs.GetWriteStream(volumePath)
			if err != nil {
				return err
			}
			return fs.WriteAndClose(writeStream, volBytes)
		}()
		e.delegate.OnVolumeFileWrite(i+1, len(e.parityData), volumePath, len(vol.data), len(volBytes), err)
		if err != nil {
			return err
		}
	}

	return nil
}

// Close closes any files opened by the encoder.
func (e *Encoder) Close() error {
	return e.fileData.Close()
}
