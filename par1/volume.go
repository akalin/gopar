package par1

import (
	"crypto/md5"
	"errors"
	"fmt"

	"io"

	"github.com/akalin/gopar/fs"
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

const controlHashOffset = 0x20

func computeSetHash(entries []fileEntry) [md5.Size]byte {
	h := md5.New()
	for _, entry := range entries {
		if entry.header.Status.savedInVolumeSet() {
			// hash.Hash.Write is guaranteed to never
			// return an error.
			h.Write(entry.header.Hash[:])
		}
	}
	var hash [md5.Size]byte
	h.Sum(hash[:0])
	return hash
}

func bytesRemaining(readStream fs.ReadStream) int64 {
	return readStream.ByteCount() - readStream.Offset()
}

const maxVolumeCount = 256

func readVolume(readStream fs.ReadStream) (v volume, err error) {
	defer func() {
		if readStream != nil {
			closeErr := readStream.Close()
			if err == nil {
				err = closeErr
				if err != nil {
					v = volume{}
				}
			}
		}
	}()

	var headerBytes [headerByteCount]byte
	_, err = fs.ReadFull(readStream, headerBytes[:])
	if err != nil {
		return volume{}, err
	}

	header, err := readHeader(headerBytes)
	if err != nil {
		return volume{}, err
	}

	if uint64(readStream.Offset()) != header.FileListOffset {
		return volume{}, fmt.Errorf("at file list offset %d, expected %d", readStream.Offset(), header.FileListOffset)
	}

	remainingBytes := bytesRemaining(readStream)
	headerRemainingBytes := header.FileListBytes + header.DataBytes
	if uint64(remainingBytes) != headerRemainingBytes {
		return volume{}, fmt.Errorf("have %d remaining file list + data bytes, expected %d", remainingBytes, headerRemainingBytes)
	}

	if header.FileCount > maxVolumeCount {
		return volume{}, fmt.Errorf("file count=%d which is greater than the maximum=%d", header.FileCount, maxVolumeCount)
	}

	h := md5.New()
	h.Write(headerBytes[controlHashOffset:])
	// Use Copy instead of CopyN because we don't want to drop
	// non-EOF errors even if we copy enough bytes.
	written, err := io.Copy(h, io.NewSectionReader(readStream, readStream.Offset(), remainingBytes))
	if err != nil {
		return volume{}, err
	}
	if written < remainingBytes {
		return volume{}, io.EOF
	}
	var controlHash [md5.Size]byte
	h.Sum(controlHash[:0])
	if controlHash != header.ControlHash {
		return volume{}, errors.New("invalid control hash")
	}

	entries := make([]fileEntry, header.FileCount)
	for i := 0; i < int(header.FileCount); i++ {
		var err error
		// TODO: Pass down a better bound.
		maxFilenameByteCount := int(bytesRemaining(readStream))
		entries[i], err = readFileEntry(readStream, maxFilenameByteCount)
		if err != nil {
			return volume{}, err
		}
	}
	setHash := computeSetHash(entries)
	if setHash != header.SetHash {
		return volume{}, errors.New("invalid set hash")
	}

	if uint64(readStream.Offset()) != header.DataOffset {
		return volume{}, fmt.Errorf("a data offset %d, expected %d", readStream.Offset(), header.FileListOffset)
	}

	remainingBytes = bytesRemaining(readStream)
	headerRemainingBytes = header.DataBytes
	if uint64(remainingBytes) != headerRemainingBytes {
		return volume{}, fmt.Errorf("have %d remaining data bytes, expected %d", remainingBytes, headerRemainingBytes)
	}

	data, err := fs.ReadAndClose(readStream)
	readStream = nil
	if err != nil {
		return volume{}, err
	}

	return volume{header, entries, data}, nil
}

func writeVolume(v volume) ([]byte, error) {
	var restData []byte
	for _, entry := range v.entries {
		fileEntryData, err := writeFileEntry(entry)
		if err != nil {
			return nil, err
		}
		restData = append(restData, fileEntryData...)
	}
	restData = append(restData, v.data...)

	header := v.header
	header.FileCount = uint64(len(v.entries))
	header.FileListOffset = expectedFileListOffset
	header.FileListBytes = uint64(len(restData) - len(v.data))
	// We'll run out of memory in building restData well before
	// the calculation below can overflow.
	header.DataOffset = header.FileListOffset + header.FileListBytes
	header.DataBytes = uint64(len(v.data))

	headerData, err := writeHeader(header)
	if err != nil {
		return nil, err
	}

	header.ControlHash = md5.Sum(append(headerData[controlHashOffset:], restData...))

	headerData, err = writeHeader(header)
	if err != nil {
		return nil, err
	}

	return append(headerData[:], restData...), nil
}
