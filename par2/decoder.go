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

type inputFileInfo struct {
	fileID        fileID
	filename      string
	byteCount     int
	sixteenKHash  [md5.Size]byte
	hash          [md5.Size]byte
	checksumPairs []checksumPair
}

func inputFileInfoIDs(infos []inputFileInfo) []fileID {
	fileIDs := make([]fileID, len(infos))
	for i, info := range infos {
		fileIDs[i] = info.fileID
	}
	return fileIDs
}

func makeInputFileInfos(fileIDs []fileID, fileDescriptionPackets map[fileID]fileDescriptionPacket, ifscPackets map[fileID]ifscPacket) ([]inputFileInfo, error) {
	var inputFileInfos []inputFileInfo
	for _, fileID := range fileIDs {
		descriptionPacket, ok := fileDescriptionPackets[fileID]
		if !ok {
			return nil, errors.New("file description packet not found")
		}
		ifscPacket, ok := ifscPackets[fileID]
		if !ok {
			return nil, errors.New("input file slice checksum packet not found")
		}
		inputFileInfos = append(inputFileInfos, inputFileInfo{
			fileID,
			descriptionPacket.filename,
			descriptionPacket.byteCount,
			descriptionPacket.sixteenKHash,
			descriptionPacket.hash,
			ifscPacket.checksumPairs,
		})
	}

	return inputFileInfos, nil
}

type fileIntegrityInfo struct {
	missing           bool
	hashMismatch      bool
	corrupt           bool
	hasWrongByteCount bool
	dataShards        [][]uint16
}

func (info fileIntegrityInfo) ok() bool {
	return !info.missing && !info.hashMismatch && !info.corrupt && !info.hasWrongByteCount
}

// A Decoder keeps track of all information needed to check the
// integrity of a set of data files, and possibly repair any
// missing/corrupted data files from the parity files (that usually
// end in .par2).
type Decoder struct {
	fileIO   fileIO
	delegate DecoderDelegate

	indexPath string

	setID          recoverySetID
	clientID       string
	sliceByteCount int
	recoverySet    []inputFileInfo
	nonRecoverySet []inputFileInfo

	// Indexed the same as recoverySet.
	fileIntegrityInfos []fileIntegrityInfo

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
	OnDataFileLoad(i, n int, path string, byteCount int, hashMismatch, corrupt, hasWrongByteCount bool, err error)
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

	recoverySet, err := makeInputFileInfos(indexFile.mainPacket.recoverySet, indexFile.fileDescriptionPackets, indexFile.ifscPackets)
	if err != nil {
		return nil, err
	}

	nonRecoverySet, err := makeInputFileInfos(indexFile.mainPacket.nonRecoverySet, indexFile.fileDescriptionPackets, indexFile.ifscPackets)
	if err != nil {
		return nil, err
	}

	return &Decoder{
		fileIO, delegate,
		indexPath,
		setID,
		indexFile.clientID, indexFile.mainPacket.sliceByteCount,
		recoverySet, nonRecoverySet,
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

func (d *Decoder) buildFileIntegrityInfo(info inputFileInfo) (int, fileIntegrityInfo, error) {
	path := filepath.Join(filepath.Dir(d.indexPath), info.filename)
	data, err := d.fileIO.ReadFile(path)
	if os.IsNotExist(err) {
		dataShards := make([][]uint16, len(info.checksumPairs))
		return 0, fileIntegrityInfo{
			missing:    true,
			dataShards: dataShards,
		}, nil
	} else if err != nil {
		return len(data), fileIntegrityInfo{}, err
	}

	hashMismatch := sixteenKHash(data) != info.sixteenKHash || md5.Sum(data) != info.hash

	var dataShards [][]uint16
	corrupt := false
	for i, checksumPair := range info.checksumPairs {
		// TODO: Handle overflow.
		expectedStartByteOffset := i * d.sliceByteCount
		chunk, ok := findChunk(checksumPair, data, d.sliceByteCount, expectedStartByteOffset)
		// TODO: Pass more info to the delegate method, like
		// where the chunk was detected (if not at the
		// expected location).
		if chunk == nil {
			dataShards = append(dataShards, nil)
			d.delegate.OnDetectCorruptDataChunk(info.fileID, info.filename, expectedStartByteOffset, expectedStartByteOffset+d.sliceByteCount)
			corrupt = true
			continue
		} else if !ok {
			d.delegate.OnDetectCorruptDataChunk(info.fileID, info.filename, expectedStartByteOffset, expectedStartByteOffset+d.sliceByteCount)
			corrupt = true
		}

		dataShard := make([]uint16, len(chunk)/2)
		err := binary.Read(bytes.NewBuffer(chunk), binary.LittleEndian, dataShard)
		if err != nil {
			return len(data), fileIntegrityInfo{}, err
		}

		dataShards = append(dataShards, dataShard)
	}

	hasWrongByteCount := false
	if len(data) != info.byteCount {
		var startByteOffset, endByteOffset int
		if len(data) < info.byteCount {
			startByteOffset = len(data)
			endByteOffset = info.byteCount
		} else {
			startByteOffset = info.byteCount
			endByteOffset = len(data)
		}
		d.delegate.OnDetectCorruptDataChunk(info.fileID, info.filename, startByteOffset, endByteOffset)
		hasWrongByteCount = true
	}

	return len(data), fileIntegrityInfo{
		hashMismatch:      hashMismatch,
		corrupt:           corrupt,
		hasWrongByteCount: hasWrongByteCount,
		dataShards:        dataShards,
	}, nil
}

// LoadFileData loads existing file data into memory.
func (d *Decoder) LoadFileData() error {
	fileIntegrityInfos := make([]fileIntegrityInfo, len(d.recoverySet))
	for i, info := range d.recoverySet {
		byteCount, fileIntegrityInfo, err := d.buildFileIntegrityInfo(info)
		d.delegate.OnDataFileLoad(i+1, len(d.recoverySet), info.filename, byteCount, fileIntegrityInfo.hashMismatch, fileIntegrityInfo.corrupt, fileIntegrityInfo.hasWrongByteCount, err)
		if err != nil {
			return err
		}

		fileIntegrityInfos[i] = fileIntegrityInfo
	}

	d.fileIntegrityInfos = fileIntegrityInfos
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

func (recoveryDelegate) OnDataFileLoad(i, n int, path string, byteCount int, hashMismatch, corrupt, hasWrongByteCount bool, err error) {
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

			if d.sliceByteCount != parityFile.mainPacket.sliceByteCount {
				return file{}, errors.New("slice size mismatch")
			}

			if !reflect.DeepEqual(inputFileInfoIDs(d.recoverySet), parityFile.mainPacket.recoverySet) {
				return file{}, errors.New("recovery set mismatch")
			}

			if !reflect.DeepEqual(inputFileInfoIDs(d.nonRecoverySet), parityFile.mainPacket.nonRecoverySet) {
				return file{}, errors.New("non-recovery set mismatch")
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

func (d *Decoder) newCoderAndShards() (rsec16.Coder, [][]uint16, error) {
	var dataShards [][]uint16
	for _, info := range d.fileIntegrityInfos {
		dataShards = append(dataShards, info.dataShards...)
	}
	coder, err := rsec16.NewCoderPAR2Vandermonde(len(dataShards), len(d.parityShards))
	if err != nil {
		return rsec16.Coder{}, nil, err
	}

	return coder, dataShards, err
}

// Verify checks that all file and known parity data are consistent
// with each other, and returns the result. If any files are missing,
// Verify returns false.
func (d *Decoder) Verify() (bool, error) {
	if len(d.fileIntegrityInfos) == 0 {
		return false, errors.New("no file integrity info")
	}

	if len(d.parityShards) == 0 {
		return false, errors.New("no parity data")
	}

	for _, info := range d.fileIntegrityInfos {
		if !info.ok() {
			return false, nil
		}
	}

	for _, shard := range d.parityShards {
		if shard == nil {
			return false, nil
		}
	}

	coder, dataShards, err := d.newCoderAndShards()
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
	if len(d.fileIntegrityInfos) == 0 {
		return nil, errors.New("no file integrity info")
	}

	if len(d.parityShards) == 0 {
		return nil, errors.New("no parity shards")
	}

	coder, dataShards, err := d.newCoderAndShards()
	if err != nil {
		return nil, err
	}

	err = coder.ReconstructData(dataShards, d.parityShards)
	if err != nil {
		return nil, err
	}

	computedParityShards := coder.GenerateParity(dataShards)
	for i, shard := range d.parityShards {
		if len(shard) == 0 {
			continue
		}

		eq := reflect.DeepEqual(computedParityShards[i], shard)
		if !eq {
			return nil, errors.New("repair failed")
		}
	}

	dir := filepath.Dir(d.indexPath)

	var repairedFiles []string

	j := 0
	for i, info := range d.fileIntegrityInfos {
		shardCount := len(info.dataShards)
		info.dataShards = dataShards[j : j+shardCount]
		j += shardCount
		d.fileIntegrityInfos[i] = info
	}

	for i, inputFileInfo := range d.recoverySet {
		fileIntegrityInfo := d.fileIntegrityInfos[i]
		if fileIntegrityInfo.ok() {
			continue
		}

		buf := bytes.NewBuffer(nil)
		for _, dataShard := range fileIntegrityInfo.dataShards {
			err := binary.Write(buf, binary.LittleEndian, dataShard)
			if err != nil {
				return repairedFiles, err
			}
		}

		data := buf.Bytes()[:inputFileInfo.byteCount]
		if sixteenKHash(data) != inputFileInfo.sixteenKHash {
			return repairedFiles, errors.New("hash mismatch (16k) in reconstructed data")
		} else if md5.Sum(data) != inputFileInfo.hash {
			return repairedFiles, errors.New("hash mismatch in reconstructed data")
		}

		path := filepath.Join(dir, inputFileInfo.filename)
		err = d.fileIO.WriteFile(path, data)
		d.delegate.OnDataFileWrite(i+1, len(d.recoverySet), path, len(data), err)
		if err != nil {
			return repairedFiles, err
		}

		repairedFiles = append(repairedFiles, inputFileInfo.filename)
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
