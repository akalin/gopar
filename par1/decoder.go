package par1

import (
	"errors"
	"fmt"
	"io"
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

	fileReadStreams []fs.ReadStream

	shardByteCount int
	parityData     [][]byte
}

// DecoderDelegate holds methods that are called during the decode
// process.
type DecoderDelegate interface {
	OnHeaderLoad(headerInfo string)
	OnFileEntryLoad(i, n int, filename, entryInfo string)
	OnCommentLoad(commentReader io.Reader, commentByteCount int64)
	OnDataFileLoad(i, n int, path string, byteCount int, corrupt bool, err error)
	OnDataFileWrite(i, n int, path string, byteCount int, err error)
	OnVolumeFileLoad(i uint64, path string, setHash [16]byte, dataByteCount int64, err error)
}

// DoNothingDecoderDelegate is an implementation of DecoderDelegate
// that does nothing for all methods.
type DoNothingDecoderDelegate struct{}

// OnHeaderLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnHeaderLoad(headerInfo string) {}

// OnFileEntryLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnFileEntryLoad(i, n int, filename, entryInfo string) {}

// OnCommentLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnCommentLoad(commentReader io.Reader, commentByteCount int64) {}

// OnDataFileLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, corrupt bool, err error) {
}

// OnDataFileWrite implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {}

// OnVolumeFileLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnVolumeFileLoad(i uint64, path string, setHash [16]byte, dataByteCount int64, err error) {
}

func newDecoder(fs fs.FS, delegate DecoderDelegate, indexFile string) *Decoder {
	return &Decoder{fs, delegate, indexFile, volume{}, nil, 0, nil}
}

// NewDecoder reads the given index file, which usually has a .PAR
// extension. The returned Decoder must be closed when it is not
// needed anymore.
func NewDecoder(delegate DecoderDelegate, indexFile string) *Decoder {
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

// Exactly one of readStream and err is non-nil, but then v may still
// be non-empty even if err is non-nil.
func readAndCheckVolume(filesystem fs.FS, path string, checkVolumeFn func(volume) error) (v volume, readStream fs.ReadStream, err error) {
	readStream, err = filesystem.GetReadStream(path)
	if err != nil {
		return volume{}, nil, err
	}

	defer func() {
		if err != nil && readStream != nil {
			_ = readStream.Close()
			readStream = nil
		}
	}()

	v, err = readVolume(readStream)
	if err != nil {
		return volume{}, readStream, err
	}

	err = checkVolumeFn(v)
	if err != nil {
		return v, readStream, err
	}

	return v, readStream, nil
}

func checkIndexVolume(indexVolume volume) error {
	if indexVolume.header.VolumeNumber != 0 {
		// TODO: Relax this check.
		return errors.New("expected volume number 0 for index volume")
	}
	return nil
}

// LoadIndexFile reads the data from the index file.
func (d *Decoder) LoadIndexFile() (err error) {
	indexVolume, readStream, err := readAndCheckVolume(d.fs, d.indexFile, checkIndexVolume)
	var dataByteCount int64
	if readStream != nil {
		dataByteCount = readStream.ByteCount() - readStream.Offset()
	}
	d.delegate.OnVolumeFileLoad(0, d.indexFile, indexVolume.header.SetHash, dataByteCount, err)
	if err != nil {
		return err
	}
	defer fs.CloseCloser(readStream, &err)

	d.delegate.OnHeaderLoad(indexVolume.header.String())
	for i, entry := range indexVolume.entries {
		d.delegate.OnFileEntryLoad(i+1, len(indexVolume.entries), entry.filename, entry.header.String())
	}

	d.delegate.OnCommentLoad(readStream, dataByteCount)

	d.indexVolume = indexVolume
	return nil
}

// LoadFileData loads existing file data into memory.
func (d *Decoder) LoadFileData() (err error) {
	fileReadStreams := make([]fs.ReadStream, 0, len(d.indexVolume.entries))
	defer func() {
		if err != nil {
			_ = closeAll(fileReadStreams)
		}
	}()

	for i, entry := range d.indexVolume.entries {
		if !entry.header.Status.savedInVolumeSet() {
			continue
		}

		path, err := d.getFilePath(entry)
		if err != nil {
			return err
		}

		readStream, corrupt, err := func() (readStream fs.ReadStream, corrupt bool, err error) {
			readStream, err = d.fs.GetReadStream(path)
			if os.IsNotExist(err) {
				return nil, true, err
			} else if err != nil {
				return nil, false, err
			}

			defer func() {
				if err != nil && readStream != nil {
					_ = readStream.Close()
					readStream = nil
				}
			}()

			err = func() error {
				hasher := hashutil.MakeMD5HashCheckerWith16k(entry.header.Hash16k, entry.header.Hash, false)
				// Use Copy instead of CopyN because
				// we don't want to drop non-EOF
				// errors even if we copy enough
				// bytes.
				written, err := io.Copy(hasher, io.NewSectionReader(readStream, 0, readStream.ByteCount()))
				if err != nil {
					return err
				} else if written < readStream.ByteCount() {
					return io.EOF
				}
				return hasher.Close()
			}()
			if _, ok := err.(hashutil.HashMismatchError); ok {
				return readStream, true, err
			} else if err != nil {
				return readStream, false, err
			}
			return readStream, false, nil
		}()
		var byteCount int
		if readStream != nil {
			byteCount = int(readStream.ByteCount())
		}
		d.delegate.OnDataFileLoad(i+1, len(d.indexVolume.entries), path, byteCount, corrupt, err)
		if corrupt {
			fileReadStreams = append(fileReadStreams, nil)
			continue
		} else if err != nil {
			return err
		}

		fileReadStreams = append(fileReadStreams, readStream)
	}

	if len(fileReadStreams) == 0 {
		return errors.New("no file data found")
	}

	d.fileReadStreams = fileReadStreams
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

func checkParityVolume(parityVolume volume, expectedSetHash [16]byte, expectedVolumeNumber uint64) error {
	if parityVolume.header.SetHash != expectedSetHash {
		// TODO: Relax this check.
		return errors.New("unexpected set hash for parity volume")
	}

	if parityVolume.header.VolumeNumber != expectedVolumeNumber {
		// TODO: Relax this check.
		return errors.New("unexpected volume number for parity volume")
	}
	return nil
}

func (d *Decoder) loadParityFile(volumeNumber uint64, volumePath string, shardByteCount *int) (volume, []byte, error) {
	parityVolume, readStream, err := readAndCheckVolume(d.fs, volumePath, func(parityVolume volume) error {
		return checkParityVolume(parityVolume, d.indexVolume.header.SetHash, volumeNumber)
	})
	if err != nil {
		return parityVolume, nil, err
	}

	data, err := fs.ReadAndClose(readStream)
	if err != nil {
		return parityVolume, nil, err
	}

	if len(data) == 0 {
		// TODO: Relax this check.
		return parityVolume, data, errors.New("no parity data in volume")
	}
	if *shardByteCount == 0 {
		*shardByteCount = len(data)
	} else if len(data) != *shardByteCount {
		// TODO: Relax this check.
		return parityVolume, data, errors.New("mismatched parity data byte counts")
	}
	return parityVolume, data, nil
}

// LoadParityData searches for parity volumes and loads them into
// memory.
func (d *Decoder) LoadParityData() error {
	// TODO: Support searching for volume data without relying on
	// filenames.

	// TODO: Count only files saved in volume set.
	fileCount := d.indexVolume.header.FileCount
	maxParityVolumeCount := maxVolumeCount - fileCount
	// TODO: Support more than 99 parity volumes.
	if maxParityVolumeCount > 99 {
		maxParityVolumeCount = 99
	}

	shardByteCount := 0
	parityData := make([][]byte, maxParityVolumeCount)
	var maxI uint64
	for i := uint64(0); i < maxParityVolumeCount; i++ {
		volumeNumber := i + 1
		// TODO: Find the file case-insensitively.
		volumePath := d.volumePath(volumeNumber)
		parityVolume, data, err := d.loadParityFile(volumeNumber, volumePath, &shardByteCount)
		d.delegate.OnVolumeFileLoad(volumeNumber, volumePath, parityVolume.header.SetHash, int64(len(data)), err)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}

		parityData[i] = data
		maxI = i
	}

	d.shardByteCount = shardByteCount
	d.parityData = parityData[:maxI+1]
	return nil
}

func (d *Decoder) buildShards() ([][]byte, error) {
	shards := make([][]byte, len(d.fileReadStreams)+len(d.parityData))
	for i, readStream := range d.fileReadStreams {
		if readStream == nil {
			continue
		}
		data, err := fs.ReadRemaining(readStream)
		if err != nil {
			return nil, err
		}
		padding := make([]byte, d.shardByteCount-len(data))
		shards[i] = append(data, padding...)
	}

	for i, data := range d.parityData {
		if data == nil {
			continue
		}
		shards[len(d.fileReadStreams)+i] = data
	}

	return shards, nil
}

func (d *Decoder) newReedSolomon() (reedsolomon.Encoder, error) {
	return reedsolomon.New(len(d.fileReadStreams), len(d.parityData), reedsolomon.WithPAR1Matrix())
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

	for _, readStream := range d.fileReadStreams {
		if readStream == nil {
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

	shards, err := d.buildShards()
	if err != nil {
		return false, err
	}

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

	shards, err := d.buildShards()
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

	for i, readStream := range d.fileReadStreams {
		if readStream != nil {
			continue
		}

		entry := d.indexVolume.entries[i]
		data := shards[i][:entry.header.FileBytes]
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
		d.delegate.OnDataFileWrite(i+1, len(d.fileReadStreams), path, len(data), err)
		if err != nil {
			return repairedPaths, err
		}

		repairedPaths = append(repairedPaths, path)
	}

	// TODO: Repair missing parity volumes, too, and then make
	// sure d.VerifyAllData() passes.

	return repairedPaths, nil
}

func closeAll(readStreams []fs.ReadStream) error {
	var firstErr error
	for i, readStream := range readStreams {
		if readStream == nil {
			continue
		}
		closeErr := readStream.Close()
		readStreams[i] = nil
		if firstErr == nil {
			firstErr = closeErr
		}
	}
	return firstErr
}

// Close the decoder and any files it may have open.
func (d *Decoder) Close() error {
	return closeAll(d.fileReadStreams)
}
