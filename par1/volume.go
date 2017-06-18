package par1

import (
	"bytes"
	"io/ioutil"
)

// A volume contains information about the volume set, and a data
// payload. For the index volume, h.VolumeNumber is 0 and data
// contains a comment for the set. For parity volumes, h.VolumeNumber
// is greater than 0, and data contains the parity data for that
// volume. All other data should be the same for all volumes in a set
// (identified by h.SetHash).
type volume struct {
	header  header
	entries []fileEntry
	data    []byte
}

func readVolume(path string) (volume, error) {
	volumeBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return volume{}, err
	}

	buf := bytes.NewBuffer(volumeBytes)

	header, err := readHeader(buf)
	if err != nil {
		return volume{}, err
	}

	// TODO: Check h.ControlHash and h.SetHash.

	// TODO: Check count of files saved in volume set, and other
	// offsets and bytes.

	entries := make([]fileEntry, header.FileCount)
	for i := uint64(0); i < header.FileCount; i++ {
		var err error
		entries[i], err = readFileEntry(buf)
		if err != nil {
			return volume{}, err
		}
	}

	data, err := ioutil.ReadAll(buf)
	if err != nil {
		return volume{}, err
	}

	return volume{header, entries, data}, nil
}
