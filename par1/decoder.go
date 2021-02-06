package par1

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/klauspost/reedsolomon"
)

// A Decoder keeps track of all information needed to check the
// integrity of a set of data files, and possibly repair any
// missing/corrupted data files from the parity files (.P00, .P01,
// etc.).
type Decoder struct {
	fileIO   fileIO
	delegate DecoderDelegate

	indexFile   string
	indexVolume volume

	fileData [][]byte

	shardByteCount int
	parityData     [][]byte
}

// DecoderDelegate holds methods that are called during the decode
// process.
type DecoderDelegate interface {
	OnHeaderLoad(headerInfo string)
	OnFileEntryLoad(i, n int, filename, entryInfo string)
	OnCommentLoad(comment []byte)
	OnDataFileLoad(i, n int, path string, byteCount int, corrupt bool, err error)
	OnDataFileWrite(i, n int, path string, byteCount int, err error)
	OnVolumeFileLoad(i uint64, path string, storedSetHash, computedSetHash [16]byte, dataByteCount int, err error)
}

func newDecoder(fileIO fileIO, delegate DecoderDelegate, indexFile string) (*Decoder, error) {
	indexVolume, err := func() (volume, error) {
		bytes, err := fileIO.ReadFile(indexFile)
		if err != nil {
			return volume{}, err
		}

		indexVolume, err := readVolume(bytes)
		if err != nil {
			return volume{}, err
		}

		if indexVolume.header.VolumeNumber != 0 {
			// TODO: Relax this check.
			return volume{}, errors.New("expected volume number 0 for index volume")
		}
		return indexVolume, nil
	}()
	delegate.OnVolumeFileLoad(0, indexFile, indexVolume.header.SetHash, indexVolume.setHash, len(indexVolume.data), err)
	if err != nil {
		return nil, err
	}

	delegate.OnHeaderLoad(indexVolume.header.String())
	for i, entry := range indexVolume.entries {
		delegate.OnFileEntryLoad(i+1, len(indexVolume.entries), entry.filename, entry.header.String())
	}
	// The comment could be in any encoding, so just pass it
	// through as bytes.
	delegate.OnCommentLoad(indexVolume.data)

	return &Decoder{
		fileIO, delegate,
		indexFile, indexVolume,
		nil,
		0, nil,
	}, nil
}

// NewDecoder reads the given index file, which usually has a .PAR
// extension.
func NewDecoder(delegate DecoderDelegate, indexFile string) (*Decoder, error) {
	return newDecoder(defaultFileIO{}, delegate, indexFile)
}

func sixteenKHash(data []byte) [md5.Size]byte {
	if len(data) < 16*1024 {
		return md5.Sum(data)
	}
	return md5.Sum(data[:16*1024])
}

func (d *Decoder) getFilePath(entry fileEntry) (string, error) {
	filename := entry.filename
	if filepath.Base(filename) != filename {
		return "", errors.New("bad filename")
	}

	basePath := filepath.Dir(d.indexFile)
	return filepath.Join(basePath, filename), nil
}

// LoadFileData loads existing file data into memory.
func (d *Decoder) LoadFileData() error {
	fileData := make([][]byte, 0, len(d.indexVolume.entries))

	for i, entry := range d.indexVolume.entries {
		if !entry.header.Status.savedInVolumeSet() {
			continue
		}

		path, err := d.getFilePath(entry)
		if err != nil {
			return err
		}

		data, corrupt, err := func() ([]byte, bool, error) {
			data, err := d.fileIO.ReadFile(path)
			if os.IsNotExist(err) {
				return nil, true, err
			} else if err != nil {
				return nil, false, err
			} else if sixteenKHash(data) != entry.header.SixteenKHash {
				return nil, true, errors.New("hash mismatch (16k)")
			} else if md5.Sum(data) != entry.header.Hash {
				return nil, true, errors.New("hash mismatch")
			}
			return data, false, nil
		}()
		d.delegate.OnDataFileLoad(i+1, len(d.indexVolume.entries), path, len(data), corrupt, err)
		if corrupt {
			fileData = append(fileData, nil)
			continue
		} else if err != nil {
			return err
		}

		// We use nil to mark missing entries, but ReadFile
		// might return nil, so convert that to a non-nil
		// empty slice.
		if data == nil {
			data = make([]byte, 0)
		}
		fileData = append(fileData, data)
	}

	if len(fileData) == 0 {
		return errors.New("no file data found")
	}

	d.fileData = fileData
	return nil
}

func (d *Decoder) volumePath(volumeNumber uint64) string {
	if volumeNumber == 0 {
		panic("unexpected zero volume number")
	}
	ext := path.Ext(d.indexFile)
	base := d.indexFile[:len(d.indexFile)-len(ext)]
	return base + fmt.Sprintf(".p%02d", volumeNumber)
}

// LoadParityData searches for parity volumes and loads them into
// memory.
func (d *Decoder) LoadParityData() error {
	// TODO: Support searching for volume data without relying on
	// filenames.

	// TODO: Count only files saved in volume set.
	fileCount := d.indexVolume.header.FileCount
	maxParityVolumeCount := 256 - fileCount
	// TODO: Support more than 99 parity volumes.
	if maxParityVolumeCount > 99 {
		maxParityVolumeCount = 99
	}

	shardByteCount := 0
	parityData := make([][]byte, maxParityVolumeCount)
	var maxI uint64
	for i := uint64(0); i < maxParityVolumeCount; i++ {
		// TODO: Find the file case-insensitively.
		volumeNumber := i + 1
		volumePath := d.volumePath(volumeNumber)
		parityVolume, byteCount, err := func() (volume, int, error) {
			volumeBytes, err := d.fileIO.ReadFile(volumePath)
			if os.IsNotExist(err) {
				return volume{}, 0, err
			} else if err != nil {
				return volume{}, 0, err
			}

			parityVolume, err := readVolume(volumeBytes)
			// TODO: Check set hash.
			if err != nil {
				// TODO: Relax this check.
				return volume{}, 0, err
			}

			byteCount := len(parityVolume.data)

			if parityVolume.header.SetHash != d.indexVolume.header.SetHash {
				// TODO: Relax this check.
				return volume{}, byteCount, errors.New("unexpected set hash for parity volume")
			}

			if parityVolume.header.VolumeNumber != uint64(i+1) {
				// TODO: Relax this check.
				return volume{}, byteCount, errors.New("unexpected volume number for parity volume")
			}

			if byteCount == 0 {
				// TODO: Relax this check.
				return volume{}, byteCount, errors.New("no parity data in volume")
			}
			if shardByteCount == 0 {
				shardByteCount = byteCount
			} else if byteCount != shardByteCount {
				// TODO: Relax this check.
				return volume{}, byteCount, errors.New("mismatched parity data byte counts")
			}
			return parityVolume, byteCount, nil
		}()
		d.delegate.OnVolumeFileLoad(volumeNumber, volumePath, parityVolume.header.SetHash, parityVolume.setHash, byteCount, err)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}

		parityData[i] = parityVolume.data
		maxI = i
	}

	d.shardByteCount = shardByteCount
	d.parityData = parityData[:maxI+1]
	return nil
}

func (d *Decoder) buildShards() [][]byte {
	shards := make([][]byte, len(d.fileData)+len(d.parityData))
	for i, data := range d.fileData {
		if data == nil {
			continue
		}
		padding := make([]byte, d.shardByteCount-len(data))
		shards[i] = append(data, padding...)
	}

	for i, data := range d.parityData {
		if data == nil {
			continue
		}
		shards[len(d.fileData)+i] = data
	}

	return shards
}

func (d *Decoder) newReedSolomon() (reedsolomon.Encoder, error) {
	return reedsolomon.New(len(d.fileData), len(d.parityData), reedsolomon.WithPAR1Matrix())
}

// Verify checks whether repair is needed. It returns a bool for
// needsRepair and an error; if error is non-nil, needsRepair may
// or may not be filled in.
func (d *Decoder) Verify() (needsRepair bool, err error) {
	shards := d.buildShards()

	needsRepair = false
	for _, data := range d.fileData {
		if data == nil {
			needsRepair = true
			break
		}
	}

	rs, err := d.newReedSolomon()
	if err != nil {
		return needsRepair, err
	}

	_, err = rs.Verify(shards)
	return needsRepair, err
}

// Repair tries to repair any missing or corrupted data, using the
// parity volumes. Returns a list of paths to files that were
// successfully repaired (relative to the indexFile passed to
// NewDecoder) in no particular order, which is present even if an
// error is returned. If checkParity is true, extra checking is done
// of the reconstructed parity data.
func (d *Decoder) Repair(checkParity bool) ([]string, error) {
	shards := d.buildShards()

	rs, err := d.newReedSolomon()
	if err != nil {
		return nil, err
	}

	err = rs.Reconstruct(shards)
	if err != nil {
		return nil, err
	}

	if checkParity {
		ok, err := rs.Verify(shards)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, errors.New("repair failed")
		}
	}

	var repairedPaths []string

	for i, data := range d.fileData {
		if data != nil {
			continue
		}

		entry := d.indexVolume.entries[i]
		data = shards[i][:entry.header.FileBytes]
		if sixteenKHash(data) != entry.header.SixteenKHash {
			return repairedPaths, errors.New("hash mismatch (16k) in reconstructed data")
		} else if md5.Sum(data) != entry.header.Hash {
			return repairedPaths, errors.New("hash mismatch in reconstructed data")
		}

		path, err := d.getFilePath(entry)
		if err != nil {
			return repairedPaths, err
		}

		err = d.fileIO.WriteFile(path, data)
		d.delegate.OnDataFileWrite(i+1, len(d.fileData), path, len(data), err)
		if err != nil {
			return repairedPaths, err
		}

		repairedPaths = append(repairedPaths, path)
		d.fileData[i] = data
	}

	// TODO: Repair missing parity volumes, too, and then make
	// sure d.Verify() passes.

	return repairedPaths, nil
}
