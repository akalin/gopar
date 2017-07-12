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

	setID                  recoverySetID
	clientID               string
	mainPacket             mainPacket
	fileDescriptionPackets map[fileID]fileDescriptionPacket
	ifscPackets            map[fileID]ifscPacket

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

	if indexFile.mainPacket == nil {
		// TODO: Relax this check.
		return nil, errors.New("no main packet found")
	}

	if len(indexFile.recoveryPackets) > 0 {
		// TODO: Relax this check.
		return nil, errors.New("recovery packets found in index file")
	}

	return &Decoder{
		fileIO, delegate,
		indexPath,
		setID,
		indexFile.clientID, *indexFile.mainPacket,
		indexFile.fileDescriptionPackets, indexFile.ifscPackets,
		nil,
		nil,
	}, nil
}

func sixteenKHash(data []byte) [md5.Size]byte {
	if len(data) < 16*1024 {
		return md5.Sum(data)
	}
	return md5.Sum(data[:16*1024])
}

// LoadFileData loads existing file data into memory.
func (d *Decoder) LoadFileData() error {
	fileData := make([][]byte, len(d.mainPacket.recoverySet))

	dir := filepath.Dir(d.indexPath)
	for i, fileID := range d.mainPacket.recoverySet {
		packet, ok := d.fileDescriptionPackets[fileID]
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
		d.delegate.OnDataFileLoad(i+1, len(d.fileDescriptionPackets), path, len(data), corrupt, err)
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

			if parityFile.mainPacket == nil || !reflect.DeepEqual(d.mainPacket, *parityFile.mainPacket) {
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
	for _, file := range parityFiles {
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

func getChunkIfMatchesHash(checksumPair checksumPair, fileData []byte, sliceByteCount, startOffset int) []byte {
	endOffset := startOffset + sliceByteCount
	if startOffset >= len(fileData) {
		return nil
	}

	if endOffset > len(fileData) {
		endOffset = len(fileData)
	}
	inputSlice := fileData[startOffset:endOffset]
	padding := make([]byte, sliceByteCount-len(inputSlice))
	inputSlice = append(inputSlice, padding...)

	// TODO: Update the CRC incrementally. Better yet, make a
	// single pass through the file with the CRC to quickly find
	// chunks.
	crc32Int := crc32.ChecksumIEEE(inputSlice)
	var crc32 [4]byte
	binary.LittleEndian.PutUint32(crc32[:], crc32Int)
	if crc32 != checksumPair.CRC32 {
		return nil
	}

	if md5.Sum(inputSlice) != checksumPair.MD5 {
		return nil
	}

	return inputSlice
}

func findChunk(checksumPair checksumPair, fileData []byte, sliceByteCount, expectedStartOffset int) ([]byte, bool) {
	chunk := getChunkIfMatchesHash(checksumPair, fileData, sliceByteCount, expectedStartOffset)
	if chunk != nil {
		return chunk, true
	}

	// TODO: Probably should cap the search or do something more
	// intelligent, to avoid long processing times for large
	// files.

	// Search forward first.
	for i := expectedStartOffset + 1; i < len(fileData); i++ {
		chunk := getChunkIfMatchesHash(checksumPair, fileData, sliceByteCount, i)
		if chunk != nil {
			return chunk, false
		}
	}

	// Then search backwards, if possible.
	start := expectedStartOffset - 1
	if start >= len(fileData) {
		start = len(fileData) - 1
	}
	for i := start; i >= 0; i-- {
		chunk := getChunkIfMatchesHash(checksumPair, fileData, sliceByteCount, i)
		if chunk != nil {
			return chunk, false
		}
	}

	return nil, false
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
	sliceByteCount := d.mainPacket.sliceByteCount

	var dataShards [][]uint16
	var corruptFileInfos []corruptFileInfo
	for i, fileData := range d.fileData {
		fileID := d.mainPacket.recoverySet[i]

		fileDescriptionPacket, ok := d.fileDescriptionPackets[fileID]
		if !ok {
			return nil, nil, errors.New("missing file description packet")
		}

		ifscPacket, ok := d.ifscPackets[fileID]
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

		startIndex := len(dataShards)
		isCorrupt := false
		for j, checksumPair := range ifscPacket.checksumPairs {
			// TODO: Handle overflow.
			expectedStartByteOffset := j * sliceByteCount
			chunk, ok := findChunk(checksumPair, fileData, sliceByteCount, expectedStartByteOffset)
			// TODO: Pass more info to the delegate
			// method, like where the chunk was detected
			// (if not at the expected location).
			if chunk == nil {
				dataShards = append(dataShards, nil)
				d.delegate.OnDetectCorruptDataChunk(fileID, fileDescriptionPacket.filename, expectedStartByteOffset, expectedStartByteOffset+sliceByteCount)
				isCorrupt = true
				continue
			} else if !ok {
				d.delegate.OnDetectCorruptDataChunk(fileID, fileDescriptionPacket.filename, expectedStartByteOffset, expectedStartByteOffset+sliceByteCount)
				isCorrupt = true
			}

			dataShard := make([]uint16, len(chunk)/2)
			err := binary.Read(bytes.NewBuffer(chunk), binary.LittleEndian, dataShard)
			if err != nil {
				return nil, nil, err
			}

			dataShards = append(dataShards, dataShard)
		}

		if !isCorrupt && len(fileData) != fileDescriptionPacket.byteCount {
			var startByteCount, endByteCount int
			if len(fileData) < fileDescriptionPacket.byteCount {
				startByteCount = len(fileData)
				endByteCount = fileDescriptionPacket.byteCount
			} else {
				startByteCount = fileDescriptionPacket.byteCount
				endByteCount = len(fileData)
			}
			d.delegate.OnDetectCorruptDataChunk(fileID, fileDescriptionPacket.filename, startByteCount, endByteCount)
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
	if len(d.fileData) != len(d.mainPacket.recoverySet) {
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
	if len(d.fileData) != len(d.mainPacket.recoverySet) {
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
