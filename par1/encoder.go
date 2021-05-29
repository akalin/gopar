package par1

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"hash"
	"io"
	"path"
	"path/filepath"

	"github.com/akalin/gopar/fs"
	"github.com/klauspost/reedsolomon"
)

type sixteenKHasher struct {
	h       hash.Hash
	written int
}

func (h *sixteenKHasher) Write(p []byte) (n int, err error) {
	if h.written < 16*1024 {
		toWrite := 16*1024 - h.written
		if len(p) < toWrite {
			toWrite = len(p)
		}
		n, err := h.h.Write(p[:toWrite])
		if err != nil {
			return n, err
		}
		h.written += toWrite
	}
	return len(p), nil
}

func (h *sixteenKHasher) Sum(b []byte) []byte {
	return h.h.Sum(b)
}

func (h *sixteenKHasher) Reset() {
	h.h.Reset()
	h.written = 0
}

func (h *sixteenKHasher) Size() int {
	return h.h.Size()
}

func (h *sixteenKHasher) BlockSize() int {
	return h.h.BlockSize()
}

type fileState struct {
	reader io.Reader
	md5    hash.Hash
	md516k hash.Hash
	size   int64
}

func (state fileState) Md5Hash() [md5.Size]byte {
	var hash [md5.Size]byte
	copy(hash[:], state.md5.Sum(nil))
	return hash
}

func (state fileState) SixteenKHash() [md5.Size]byte {
	var hash [md5.Size]byte
	copy(hash[:], state.md516k.Sum(nil))
	return hash
}

// An Encoder keeps track of all information needed to create parity
// volumes for a set of data files, and write them out to parity files
// (.PAR, .P00, .P01, etc.).
type Encoder struct {
	fs       fs.FS
	delegate EncoderDelegate

	filePaths   []string
	volumeCount int

	maxByteCount int64
	fileData     []fileState
	parityData   [][]byte
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
	return newEncoder(fs.DefaultFS{}, delegate, filePaths, volumeCount)
}

// LoadFileData loads the file data into memory.
func (e *Encoder) LoadFileData() error {
	maxByteCount := int64(0)
	fileData := make([]fileState, len(e.filePaths))
	for i, path := range e.filePaths {
		data, err := func() ([]byte, error) {
			readStream, err := e.fs.GetReadStream(path)
			if err != nil {
				return nil, err
			}
			return fs.ReadAndClose(readStream)
		}()
		e.delegate.OnDataFileLoad(i+1, len(e.filePaths), path, len(data), err)
		if err != nil {
			return err
		}

		if int64(len(data)) > maxByteCount {
			maxByteCount = int64(len(data))
		}

		fileData[i] = fileState{bytes.NewBuffer(data), md5.New(), &sixteenKHasher{md5.New(), 0}, int64(len(data))}
	}

	e.maxByteCount = maxByteCount
	e.fileData = fileData
	return nil
}

func (e *Encoder) fillDataShards(shards [][]byte, off int64, fillByteCount int) error {
	for i, fileState := range e.fileData {
		bytesToRead := 0
		if off < fileState.size {
			bytesToRead = fillByteCount
			if int64(bytesToRead) > fileState.size-off {
				bytesToRead = int(fileState.size - off)
			}
			shard := shards[i][:bytesToRead]
			_, err := io.ReadFull(fileState.reader, shard)
			if err != nil {
				return err
			}
			_, err = fileState.md5.Write(shard)
			if err != nil {
				return err
			}
			_, err = fileState.md516k.Write(shard)
			if err != nil {
				return err
			}
		}
		for j := bytesToRead; j < fillByteCount; j++ {
			shards[i][j] = 0
		}
	}
	return nil
}

// ComputeParityData computes the parity data for the files.
func (e *Encoder) ComputeParityData(bufByteCount int) error {
	if bufByteCount <= 0 {
		return errors.New("bufByteCount must be positive")
	}

	if int64(bufByteCount) > e.maxByteCount {
		bufByteCount = int(e.maxByteCount)
	}

	rs, err := reedsolomon.New(len(e.fileData), e.volumeCount, reedsolomon.WithPAR1Matrix())
	if err != nil {
		return err
	}

	shards := make([][]byte, len(e.fileData)+e.volumeCount)
	for i := range shards {
		shards[i] = make([]byte, bufByteCount)
	}

	e.parityData = make([][]byte, e.volumeCount)

	for off := int64(0); off < e.maxByteCount; off += int64(bufByteCount) {
		fillByteCount := bufByteCount
		if e.maxByteCount-off < int64(fillByteCount) {
			fillByteCount = int(e.maxByteCount - off)
		}
		err := e.fillDataShards(shards, off, fillByteCount)
		if err != nil {
			return err
		}

		err = rs.Encode(shards)
		if err != nil {
			return err
		}

		for i := range e.parityData {
			e.parityData[i] = append(e.parityData[i], shards[len(e.fileData)+i][:fillByteCount]...)
		}
	}

	return nil
}

func (e *Encoder) Write(indexPath string) error {
	var entries []fileEntry
	for i, k := range e.filePaths {
		state := e.fileData[i]
		var status fileEntryStatus
		status.setSavedInVolumeSet(true)
		hash := state.Md5Hash()
		entry := fileEntry{
			header: fileEntryHeader{
				Status:    status,
				FileBytes: uint64(state.size),
				Hash:      hash,
				Hash16k:   state.SixteenKHash(),
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
