package par1

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/akalin/gopar/fs"
	"github.com/akalin/gopar/hashutil"
	"github.com/klauspost/reedsolomon"
)

// A Decoder keeps track of all information needed to check the
// integrity of a set of data files, and possibly repair any
// missing/corrupt data files from the parity files (.P00, .P01,
// etc.).
type Decoder struct {
	fs       fs.FS
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

// DoNothingDecoderDelegate is an implementation of DecoderDelegate
// that does nothing for all methods.
type DoNothingDecoderDelegate struct{}

// OnHeaderLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnHeaderLoad(headerInfo string) {}

// OnFileEntryLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnFileEntryLoad(i, n int, filename, entryInfo string) {}

// OnCommentLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnCommentLoad(comment []byte) {}

// OnDataFileLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, corrupt bool, err error) {
}

// OnDataFileWrite implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {}

// OnVolumeFileLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnVolumeFileLoad(i uint64, path string, storedSetHash, computedSetHash [16]byte, dataByteCount int, err error) {
}

func newDecoder(filesystem fs.FS, delegate DecoderDelegate, indexFile string) (*Decoder, error) {
	indexVolume, err := func() (volume, error) {
		readStream, err := filesystem.GetReadStream(indexFile)
		if err != nil {
			return volume{}, err
		}
		bytes, err := fs.ReadAndClose(readStream)
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
		filesystem, delegate,
		indexFile, indexVolume,
		nil,
		0, nil,
	}, nil
}

// NewDecoder reads the given index file, which usually has a .PAR
// extension.
func NewDecoder(delegate DecoderDelegate, indexFile string) (*Decoder, error) {
	return newDecoder(fs.MakeDefaultFS(), delegate, indexFile)
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
			readStream, err := d.fs.GetReadStream(path)
			if os.IsNotExist(err) {
				return nil, true, err
			} else if err != nil {
				return nil, false, err
			}
			hasher := hashutil.MakeMD5HashCheckerWith16k(entry.header.Hash16k, entry.header.Hash, false)
			data, err := fs.ReadAndClose(hashutil.TeeReadStream(readStream, hasher))
			if _, ok := err.(hashutil.HashMismatchError); ok {
				return nil, true, err
			} else if err != nil {
				return nil, false, err
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
			readStream, err := d.fs.GetReadStream(volumePath)
			if err != nil {
				return volume{}, 0, err
			}
			volumeBytes, err := fs.ReadAndClose(readStream)
			if err != nil {
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

// FileCounts contains file counts which can be used to deduce whether
// repair is necessary and/or possible.
type FileCounts struct {
	// UsableDataFileCount is the number of data files that are
	// usable, i.e. not missing and not corrupt.
	UsableDataFileCount int
	// UnusableDataFileCount is the number of data files that are
	// unusable, i.e. missing or corrupt.
	UnusableDataFileCount int

	// UsableParityFileCount is the number of parity files that
	// exist, i.e. not missing and not corrupt.
	UsableParityFileCount int
	// UnusableDataFileCount is the number of parity files that
	// are unusable, i.e. missing or corrupt.
	//
	// Note that we can only infer missing parity files from gaps
	// in the counts, e.g. if we only have foo.p01 and foo.p03,
	// then we can infer that foo.p02 is missing, but if we have
	// foo.p01 and foo.p02, then foo.p03 might be missing or it
	// might not have existed in the first place.
	UnusableParityFileCount int
}

// RepairNeeded returns whether repair is needed, i.e. whether
// UnusableDataFileCount is non-zero.
func (fc FileCounts) RepairNeeded() bool {
	return fc.UnusableDataFileCount > 0
}

// RepairPossible returns whether repair is possible i.e. whether
// UsableParityFileCount >= UnusableDataFileCount.
func (fc FileCounts) RepairPossible() bool {
	return fc.UsableParityFileCount >= fc.UnusableDataFileCount
}

// AllFilesUsable returns whether or not all files are usable,
// i.e. whether UnusableDataFileCount == 0 and UnusableParityFileCount
// == 0.
func (fc FileCounts) AllFilesUsable() bool {
	return fc.UnusableDataFileCount == 0 && fc.UnusableParityFileCount == 0
}

// FileCounts returns a FileCounts object for the current file set.
func (d *Decoder) FileCounts() FileCounts {
	usableDataFileCount := 0
	unusableDataFileCount := 0

	for _, data := range d.fileData {
		if data == nil {
			unusableDataFileCount++
		} else {
			usableDataFileCount++
		}
	}

	usableParityFileCount := 0
	unusableParityFileCount := 0

	for _, data := range d.parityData {
		if data == nil {
			unusableParityFileCount++
		} else {
			usableParityFileCount++
		}
	}

	return FileCounts{
		UsableDataFileCount:     usableDataFileCount,
		UnusableDataFileCount:   unusableDataFileCount,
		UsableParityFileCount:   usableParityFileCount,
		UnusableParityFileCount: unusableParityFileCount,
	}
}

// VerifyAllData checks and returns whether all data and parity files
// contain correct data. Note that this is worth calling only when
// there are no unusable data or parity files, since if there are,
// then false is guaranteed to be returned for ok.
func (d *Decoder) VerifyAllData() (ok bool, err error) {
	rs, err := d.newReedSolomon()
	if err != nil {
		return false, err
	}

	shards := d.buildShards()

	return rs.Verify(shards)
}

// Repair tries to repair any missing or corrupt data, using the
// parity volumes. Returns a list of paths to files that were
// successfully repaired (relative to the indexFile passed to
// NewDecoder) in no particular order, which is present even if an
// error is returned. If checkParity is true, extra checking is done
// of the reconstructed parity data.
func (d *Decoder) Repair(checkParity bool) ([]string, error) {
	rs, err := d.newReedSolomon()
	if err != nil {
		return nil, err
	}

	shards := d.buildShards()

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
		if err := hashutil.CheckMD5Hashes(data, entry.header.Hash16k, entry.header.Hash, true); err != nil {
			return repairedPaths, err
		}

		path, err := d.getFilePath(entry)
		if err != nil {
			return repairedPaths, err
		}

		err = func() error {
			writeStream, err := d.fs.GetWriteStream(path)
			if err != nil {
				return err
			}
			return fs.WriteAndClose(writeStream, data)
		}()
		d.delegate.OnDataFileWrite(i+1, len(d.fileData), path, len(data), err)
		if err != nil {
			return repairedPaths, err
		}

		repairedPaths = append(repairedPaths, path)
		d.fileData[i] = data
	}

	// TODO: Repair missing parity volumes, too, and then make
	// sure d.VerifyAllData() passes.

	return repairedPaths, nil
}
