package par1

import (
	"errors"
	"fmt"
	"os"
	"path"
)

// A Decoder keeps track of all information needed to check the
// integrity of a set of data files, and possibly reconstruct any
// missing/corrupted data files from the parity files (.P00, .P01,
// etc.).
type Decoder struct {
	indexFile   string
	indexVolume volume
	parityData  [][]byte
}

// NewDecoder reads the given index file, which usually has a .PAR
// extension.
func NewDecoder(indexFile string) (*Decoder, error) {
	indexVolume, err := readVolume(indexFile)
	if err != nil {
		return nil, err
	}

	if indexVolume.header.VolumeNumber != 0 {
		// TODO: Relax this check.
		return nil, errors.New("expected volume number 0 for index volume")
	}

	return &Decoder{indexFile, indexVolume, nil}, nil
}

// LoadParityData searches for parity volumes and loads them into
// memory.
func (d *Decoder) LoadParityData() error {
	ext := path.Ext(d.indexFile)
	base := d.indexFile[:len(d.indexFile)-len(ext)]

	// TODO: Support searching for volume data without relying on
	// filenames.

	// TODO: Count only files saved in volume set.
	fileCount := d.indexVolume.header.FileCount
	maxParityVolumeCount := 256 - fileCount
	// TODO: Support more than 99 parity volumes.
	if maxParityVolumeCount > 99 {
		maxParityVolumeCount = 99
	}
	parityData := make([][]byte, maxParityVolumeCount)
	var maxI uint64
	for i := uint64(0); i < maxParityVolumeCount; i++ {
		// TODO: Find the file case-insensitively.
		volumePath := base + fmt.Sprintf(".p%02d", i+1)
		parityVolume, err := readVolume(volumePath)
		// TODO: Check set hash.
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			// TODO: Relax this check.
			return err
		} else if parityVolume.header.VolumeNumber != uint64(i+1) {
			// TODO: Relax this check.
			return errors.New("unexpected volume number for parity volume")
		}

		parityData[i] = parityVolume.data
		maxI = i
	}

	d.parityData = parityData[:maxI+1]
	return nil
}
