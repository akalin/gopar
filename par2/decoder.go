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
	WriteFile(path string, data []byte) error
}

type defaultFileIO struct{}

func (io defaultFileIO) ReadFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func (io defaultFileIO) FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error) {
	return filepath.Glob(prefix + "*" + suffix)
}

func (io defaultFileIO) WriteFile(path string, data []byte) error {
	return ioutil.WriteFile(path, data, 0600)
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
	OnDetectCorruptDataChunk(fileID [16]byte, filename string, startByteOffset, endByteOffset int)
	OnDataFileWrite(i, n int, path string, byteCount int, err error)
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
			}
			corrupt := sixteenKHash(data) != packet.sixteenKHash || md5.Sum(data) != packet.hash
			// If corrupt, load data anyway, and let
			// buildDataShards() sort out which chunks
			// specifically are corrupt.
			return data, corrupt, nil
		}()
		d.delegate.OnDataFileLoad(i+1, len(d.indexFile.fileDescriptionPackets), path, len(data), corrupt, err)
		if err != nil && !corrupt {
			return err
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

func (recoveryDelegate) OnDetectCorruptDataChunk(fileID [16]byte, filename string, startByteOffset, endByteOffset int) {
}

func (recoveryDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {}

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

type corruptFileInfo struct {
	i                    int
	filename             string
	byteCount            int
	startIndex, endIndex int
	hash                 [md5.Size]byte
	sixteenKHash         [md5.Size]byte
}

func (d *Decoder) buildDataShards() ([][]uint16, []corruptFileInfo, error) {
	sliceByteCount := d.indexFile.mainPacket.sliceByteCount

	var dataShards [][]uint16
	var corruptFileInfos []corruptFileInfo
	for i, fileData := range d.fileData {
		fileID := d.indexFile.mainPacket.recoverySet[i]

		fileDescriptionPacket, ok := d.indexFile.fileDescriptionPackets[fileID]
		if !ok {
			return nil, nil, errors.New("missing file description packet")
		}

		ifscPacket, ok := d.indexFile.ifscPackets[fileID]
		if !ok {
			return nil, nil, errors.New("missing input file slice checksums")
		}

		if len(fileData) == 0 {
			startIndex := len(dataShards)
			dataShards = append(dataShards, make([][]uint16, len(ifscPacket.checksumPairs))...)
			d.delegate.OnDetectCorruptDataChunk(fileID, fileDescriptionPacket.filename, 0, fileDescriptionPacket.byteCount)
			endIndex := len(dataShards)
			corruptFileInfos = append(corruptFileInfos, corruptFileInfo{
				i,
				fileDescriptionPacket.filename,
				fileDescriptionPacket.byteCount,
				startIndex, endIndex,
				fileDescriptionPacket.hash, fileDescriptionPacket.sixteenKHash,
			})
			continue
		}

		// TODO: Handle file corruption more robustly,
		// e.g. corruption that adds or deletes bytes.

		startIndex := len(dataShards)
		isCorrupt := false
		for j, checksumPair := range ifscPacket.checksumPairs {
			// TODO: Handle overflow.
			startByteOffset := j * sliceByteCount
			endByteOffset := startByteOffset + sliceByteCount
			if startByteOffset >= len(fileData) {
				dataShards = append(dataShards, nil)
				d.delegate.OnDetectCorruptDataChunk(fileID, fileDescriptionPacket.filename, startByteOffset, endByteOffset)
				isCorrupt = true
				continue
			}

			if endByteOffset > len(fileData) {
				endByteOffset = len(fileData)
			}
			inputSlice := fileData[startByteOffset:endByteOffset]
			padding := make([]byte, sliceByteCount-len(inputSlice))
			inputSlice = append(inputSlice, padding...)
			if md5.Sum(inputSlice) != checksumPair.MD5 {
				dataShards = append(dataShards, nil)
				d.delegate.OnDetectCorruptDataChunk(fileID, fileDescriptionPacket.filename, startByteOffset, endByteOffset)
				isCorrupt = true
				continue
			}
			crc32Int := crc32.ChecksumIEEE(inputSlice)
			var crc32 [4]byte
			binary.LittleEndian.PutUint32(crc32[:], crc32Int)
			if crc32 != checksumPair.CRC32 {
				dataShards = append(dataShards, nil)
				d.delegate.OnDetectCorruptDataChunk(fileID, fileDescriptionPacket.filename, startByteOffset, endByteOffset)
				isCorrupt = true
				continue
			}

			dataShard := make([]uint16, len(inputSlice)/2)
			err := binary.Read(bytes.NewBuffer(inputSlice), binary.LittleEndian, dataShard)
			if err != nil {
				return nil, nil, err
			}

			dataShards = append(dataShards, dataShard)
		}

		if !isCorrupt && len(fileData) != fileDescriptionPacket.byteCount {
			isCorrupt = true
		}

		if isCorrupt {
			endIndex := len(dataShards)
			corruptFileInfos = append(corruptFileInfos, corruptFileInfo{
				i,
				fileDescriptionPacket.filename,
				fileDescriptionPacket.byteCount,
				startIndex, endIndex,
				fileDescriptionPacket.hash, fileDescriptionPacket.sixteenKHash,
			})
		}
	}

	return dataShards, corruptFileInfos, nil
}

func (d *Decoder) newCoder(dataShards [][]uint16) (rsec16.Coder, error) {
	return rsec16.NewCoderPAR2Vandermonde(len(dataShards), len(d.parityShards))
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

	dataShards, corruptFileInfos, err := d.buildDataShards()
	if err != nil {
		return false, err
	}

	if len(corruptFileInfos) > 0 {
		return false, nil
	}

	coder, err := d.newCoder(dataShards)
	if err != nil {
		return false, err
	}

	computedParityShards := coder.GenerateParity(dataShards)
	eq := reflect.DeepEqual(computedParityShards, d.parityShards)
	return eq, nil
}

// Repair tries to repair any missing or corrupted data, using the
// parity volumes. Returns a list of files that were successfully
// repaired, which is present even if an error is returned.
func (d *Decoder) Repair() ([]string, error) {
	if d.indexFile.mainPacket == nil {
		return nil, errors.New("main packet not loaded")
	}

	if len(d.fileData) != len(d.indexFile.mainPacket.recoverySet) {
		return nil, errors.New("file data count mismatch")
	}

	dataShards, corruptFileInfos, err := d.buildDataShards()
	if err != nil {
		return nil, err
	}

	coder, err := d.newCoder(dataShards)
	if err != nil {
		return nil, err
	}

	err = coder.ReconstructData(dataShards, d.parityShards)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(d.indexPath)

	var repairedFiles []string

	for _, corruptFileInfo := range corruptFileInfos {
		var data []uint16
		for i := corruptFileInfo.startIndex; i < corruptFileInfo.endIndex; i++ {
			data = append(data, dataShards[i]...)
		}

		buf := bytes.NewBuffer(nil)
		err := binary.Write(buf, binary.LittleEndian, data)
		if err != nil {
			return repairedFiles, err
		}

		// TODO: Handle overflow.
		byteData := buf.Bytes()[:corruptFileInfo.byteCount]
		if sixteenKHash(byteData) != corruptFileInfo.sixteenKHash {
			return repairedFiles, errors.New("hash mismatch (16k) in reconstructed data")
		} else if md5.Sum(byteData) != corruptFileInfo.hash {
			return repairedFiles, errors.New("hash mismatch in reconstructed data")
		}

		path := filepath.Join(dir, corruptFileInfo.filename)
		err = d.fileIO.WriteFile(path, byteData)
		d.delegate.OnDataFileWrite(corruptFileInfo.i+1, len(d.fileData), path, len(byteData), err)
		if err != nil {
			return repairedFiles, err
		}

		repairedFiles = append(repairedFiles, corruptFileInfo.filename)
		d.fileData[corruptFileInfo.i] = byteData
	}

	// TODO: Repair missing parity volumes, too, and then make
	// sure d.Verify() passes.

	return repairedFiles, nil
}

// NewDecoder reads the given index file, which usually has a .par2
// extension.
func NewDecoder(delegate DecoderDelegate, indexFile string) (*Decoder, error) {
	return newDecoder(defaultFileIO{}, delegate, indexFile)
}
