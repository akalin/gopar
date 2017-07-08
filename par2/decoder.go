package par2

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"

	"github.com/akalin/gopar/rsec16"
)

type fileIO interface {
	ReadFile(path string) ([]byte, error)
	FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error)
}

type defaultFileIO struct{}

func (io defaultFileIO) ReadFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func (io defaultFileIO) FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error) {
	return filepath.Glob(prefix + "*" + suffix)
}

// A Decoder keeps track of all information needed to check the
// integrity of a set of data files, and possibly repair any
// missing/corrupted data files from the parity files (that usually
// end in .par2).
type Decoder struct {
	fileIO   fileIO
	delegate DecoderDelegate

	indexPath string

	setID     recoverySetID
	indexFile file

	fileData [][]byte

	parityShards [][]uint16
}

// DecoderDelegate holds methods that are called during the decode
// process.
type DecoderDelegate interface {
	OnCreatorPacketLoad(clientID string)
	OnMainPacketLoad(sliceByteCount, recoverySetCount, nonRecoverySetCount int)
	OnFileDescriptionPacketLoad(fileID [16]byte, filename string, byteCount int)
	OnIFSCPacketLoad(fileID [16]byte)
	OnRecoveryPacketLoad(exponent uint16, byteCount int)
	OnUnknownPacketLoad(packetType [16]byte, byteCount int)
	OnOtherPacketSkip(setID [16]byte, packetType [16]byte, byteCount int)
	OnDataFileLoad(i, n int, path string, byteCount int, corrupt bool, err error)
	OnParityFileLoad(i int, path string, err error)
}

func newDecoder(fileIO fileIO, delegate DecoderDelegate, indexPath string) (*Decoder, error) {
	indexBytes, err := fileIO.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}

	setID, indexFile, err := readFile(delegate, nil, indexBytes)
	if err != nil {
		return nil, err
	}

	return &Decoder{fileIO, delegate, indexPath, setID, indexFile, nil, nil}, nil
}

func sixteenKHash(data []byte) [md5.Size]byte {
	if len(data) < 16*1024 {
		return md5.Sum(data)
	}
	return md5.Sum(data[:16*1024])
}

// LoadFileData loads existing file data into memory.
func (d *Decoder) LoadFileData() error {
	if d.indexFile.mainPacket == nil {
		return errors.New("main packet not loaded")
	}

	fileData := make([][]byte, len(d.indexFile.mainPacket.recoverySet))

	dir := filepath.Dir(d.indexPath)
	for i, fileID := range d.indexFile.mainPacket.recoverySet {
		packet, ok := d.indexFile.fileDescriptionPackets[fileID]
		if !ok {
			return errors.New("could not find file description packet for")
		}

		path := filepath.Join(dir, packet.filename)
		data, corrupt, err := func() ([]byte, bool, error) {
			data, err := d.fileIO.ReadFile(path)
			if os.IsNotExist(err) {
				return nil, true, err
			} else if err != nil {
				return nil, false, err
			} else if sixteenKHash(data) != packet.sixteenKHash {
				return nil, true, errors.New("hash mismatch (16k)")
			} else if md5.Sum(data) != packet.hash {
				return nil, true, errors.New("hash mismatch")
			}
			return data, false, nil
		}()
		d.delegate.OnDataFileLoad(i+1, len(d.indexFile.fileDescriptionPackets), path, len(data), corrupt, err)
		if corrupt {
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
		fileData[i] = data
	}

	d.fileData = fileData
	return nil
}

type recoveryDelegate struct {
	d DecoderDelegate
}

func (recoveryDelegate) OnCreatorPacketLoad(clientID string) {}

func (recoveryDelegate) OnMainPacketLoad(sliceByteCount, recoverySetCount, nonRecoverySetCount int) {}

func (recoveryDelegate) OnFileDescriptionPacketLoad(fileID [16]byte, filename string, byteCount int) {}

func (recoveryDelegate) OnIFSCPacketLoad(fileID [16]byte) {}

func (r recoveryDelegate) OnRecoveryPacketLoad(exponent uint16, byteCount int) {
	r.d.OnRecoveryPacketLoad(exponent, byteCount)
}

func (r recoveryDelegate) OnUnknownPacketLoad(packetType [16]byte, byteCount int) {
	r.d.OnUnknownPacketLoad(packetType, byteCount)
}

func (recoveryDelegate) OnOtherPacketSkip(setID [16]byte, packetType [16]byte, byteCount int) {}

func (recoveryDelegate) OnDataFileLoad(i, n int, path string, byteCount int, corrupt bool, err error) {
}

func (recoveryDelegate) OnParityFileLoad(i int, path string, err error) {}

// LoadParityData searches for parity volumes and loads them into
// memory.
func (d *Decoder) LoadParityData() error {
	if d.indexFile.mainPacket == nil {
		return errors.New("main packet not loaded")
	}

	ext := path.Ext(d.indexPath)
	base := d.indexPath[:len(d.indexPath)-len(ext)]
	matches, err := d.fileIO.FindWithPrefixAndSuffix(base+".", ext)
	if err != nil {
		return err
	}

	var parityFiles []file
	for i, match := range matches {
		parityFile, err := func() (file, error) {
			volumeBytes, err := d.fileIO.ReadFile(match)
			if err != nil {
				return file{}, err
			}

			// Ignore all the other packet types other
			// than recovery packets.
			_, parityFile, err := readFile(recoveryDelegate{d.delegate}, &d.setID, volumeBytes)
			if err != nil {
				// TODO: Relax this check.
				return file{}, err
			}

			if !reflect.DeepEqual(d.indexFile.mainPacket, parityFile.mainPacket) {
				return file{}, errors.New("main packet mismatch")
			}

			return parityFile, nil
		}()
		d.delegate.OnParityFileLoad(i+1, match, err)
		if err != nil {
			return err
		}

		parityFiles = append(parityFiles, parityFile)
	}

	var parityShards [][]uint16
	for _, file := range append(parityFiles, d.indexFile) {
		for exponent, packet := range file.recoveryPackets {
			if int(exponent) >= len(parityShards) {
				parityShards = append(parityShards, make([][]uint16, int(exponent+1)-len(parityShards))...)
			}
			parityShards[exponent] = packet.data
		}
	}

	d.parityShards = parityShards
	return nil
}

// Verify checks that all file and known parity data are consistent
// with each other, and returns the result. If any files are missing,
// Verify returns false.
func (d *Decoder) Verify() (bool, error) {
	if d.indexFile.mainPacket == nil {
		return false, errors.New("main packet not loaded")
	}

	if len(d.fileData) != len(d.indexFile.mainPacket.recoverySet) {
		return false, errors.New("file data count mismatch")
	}

	for _, data := range d.fileData {
		if data == nil {
			return false, nil
		}
	}

	for _, shard := range d.parityShards {
		if shard == nil {
			return false, nil
		}
	}

	sliceByteCount := d.indexFile.mainPacket.sliceByteCount

	var dataShards [][]uint16
	for i, fileData := range d.fileData {
		fileID := d.indexFile.mainPacket.recoverySet[i]
		ifscPacket, ok := d.indexFile.ifscPackets[fileID]
		if !ok {
			return false, errors.New("missing input file slice checksums")
		}

		for j, checksumPair := range ifscPacket.checksumPairs {
			// TODO: Handle overflow.
			startOffset := j * sliceByteCount
			if startOffset >= len(fileData) {
				return false, errors.New("start index out of bounds")
			}
			endOffset := startOffset + sliceByteCount
			if endOffset > len(fileData) {
				endOffset = len(fileData)
			}
			inputSlice := fileData[startOffset:endOffset]
			padding := make([]byte, sliceByteCount-len(inputSlice))
			inputSlice = append(inputSlice, padding...)
			if md5.Sum(inputSlice) != checksumPair.MD5 {
				return false, errors.New("md5 mismatch")
			}
			crc32Int := crc32.ChecksumIEEE(inputSlice)
			var crc32 [4]byte
			binary.LittleEndian.PutUint32(crc32[:], crc32Int)
			if crc32 != checksumPair.CRC32 {
				return false, errors.New("crc32 mismatch")
			}

			dataShard := make([]uint16, len(inputSlice)/2)
			err := binary.Read(bytes.NewBuffer(inputSlice), binary.LittleEndian, dataShard)
			if err != nil {
				return false, err
			}

			dataShards = append(dataShards, dataShard)
		}
	}

	// TODO: Make coder use the PAR2 matrix.

	coder := rsec16.NewCoder(len(dataShards), len(d.parityShards))
	computedParityShards := coder.GenerateParity(dataShards)
	eq := reflect.DeepEqual(computedParityShards, d.parityShards)
	return eq, nil
}

// NewDecoder reads the given index file, which usually has a .par2
// extension.
func NewDecoder(delegate DecoderDelegate, indexFile string) (*Decoder, error) {
	return newDecoder(defaultFileIO{}, delegate, indexFile)
}
