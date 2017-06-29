package par1

import (
	"errors"
	"fmt"
	"io/ioutil"
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
	fileIO fileIO

	indexFile   string
	indexVolume volume

	fileData [][]byte

	shardByteCount int
	parityData     [][]byte
}

type fileIO interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
}

func newDecoder(fileIO fileIO, indexFile string) (*Decoder, error) {
	bytes, err := fileIO.ReadFile(indexFile)
	if err != nil {
		return nil, err
	}

	indexVolume, err := readVolume(bytes)
	if err != nil {
		return nil, err
	}

	if indexVolume.header.VolumeNumber != 0 {
		// TODO: Relax this check.
		return nil, errors.New("expected volume number 0 for index volume")
	}

	return &Decoder{
		fileIO,
		indexFile, indexVolume,
		nil,
		0, nil,
	}, nil
}

type defaultFileIO struct{}

func (io defaultFileIO) ReadFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func (io defaultFileIO) WriteFile(path string, data []byte) error {
	return ioutil.WriteFile(path, data, 0600)
}

// NewDecoder reads the given index file, which usually has a .PAR
// extension.
func NewDecoder(indexFile string) (*Decoder, error) {
	return newDecoder(defaultFileIO{}, indexFile)
}

// LoadFileData loads existing file data into memory.
func (d *Decoder) LoadFileData() error {
	fileData := make([][]byte, len(d.indexVolume.entries))

	dir := filepath.Dir(d.indexFile)
	for i, entry := range d.indexVolume.entries {
		// TODO: Check file status and skip if necessary.
		path := filepath.Join(dir, entry.filename)
		data, err := d.fileIO.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			// TODO: Relax this check.
			return err
		}
		// We use nil to mark missing entries, but ReadFile
		// might return nil, so convert that to a non-nil
		// empty slice.
		if data == nil {
			data = make([]byte, 0)
		}
		// TODO: Check file checksum.
		fileData[i] = data
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
		volumeBytes, err := d.fileIO.ReadFile(d.volumePath(volumeNumber))
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}

		parityVolume, err := readVolume(volumeBytes)
		// TODO: Check set hash.
		if err != nil {
			// TODO: Relax this check.
			return err
		} else if parityVolume.header.VolumeNumber != volumeNumber {
			// TODO: Relax this check.
			return errors.New("unexpected volume number for parity volume")
		}
		if len(parityVolume.data) == 0 {
			// TODO: Relax this check.
			return errors.New("no parity data in volume")
		}
		if shardByteCount == 0 {
			shardByteCount = len(parityVolume.data)
		} else if len(parityVolume.data) != shardByteCount {
			// TODO: Relax this check.
			return errors.New("mismatched parity data byte counts")
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

// Verify checks that all file and parity data are consistent with
// each other, and returns the result. If any files or parity volumes
// are missing, Verify returns false.
func (d *Decoder) Verify() (bool, error) {
	for _, data := range d.fileData {
		if data == nil {
			return false, nil
		}
	}

	for _, data := range d.parityData {
		if data == nil {
			return false, nil
		}
	}

	shards := d.buildShards()

	rs, err := d.newReedSolomon()
	if err != nil {
		return false, err
	}

	return rs.Verify(shards)
}

// Repair tries to repair any missing or corrupted data, using the
// parity volumes. Returns a list of files that were successfully
// repaired, which is present even if an error is returned.
func (d *Decoder) Repair() ([]string, error) {
	shards := d.buildShards()

	rs, err := d.newReedSolomon()
	if err != nil {
		return nil, err
	}

	err = rs.Reconstruct(shards)
	if err != nil {
		return nil, err
	}

	ok, err := rs.Verify(shards)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, errors.New("repair failed")
	}

	dir := filepath.Dir(d.indexFile)

	var repairedFiles []string

	for i, data := range d.fileData {
		if data != nil {
			continue
		}

		entry := d.indexVolume.entries[i]
		data = shards[i][:entry.header.FileBytes]
		// TODO: Check hash of data.

		filename := d.indexVolume.entries[i].filename
		path := filepath.Join(dir, filename)
		err = d.fileIO.WriteFile(path, data)
		if err != nil {
			return repairedFiles, err
		}

		repairedFiles = append(repairedFiles, filename)
		d.fileData[i] = data
	}

	// TODO: Repair missing parity volumes, too, and then make
	// sure d.Verify() passes.

	return repairedFiles, nil
}
