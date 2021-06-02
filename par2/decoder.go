package par2

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"path"
	"path/filepath"
	"reflect"

	"github.com/akalin/gopar/fs"
	"github.com/akalin/gopar/hashutil"
	"github.com/akalin/gopar/rsec16"
)

type decoderInputFileInfo struct {
	fileID        fileID
	filename      string
	byteCount     int
	hash16k       [md5.Size]byte
	hash          [md5.Size]byte
	checksumPairs []checksumPair
}

func decoderInputFileInfoIDs(infos []decoderInputFileInfo) []fileID {
	fileIDs := make([]fileID, len(infos))
	for i, info := range infos {
		fileIDs[i] = info.fileID
	}
	return fileIDs
}

func makeDecoderInputFileInfos(fileIDs []fileID, fileDescriptionPackets map[fileID]fileDescriptionPacket, ifscPackets map[fileID]ifscPacket) ([]decoderInputFileInfo, error) {
	var decoderInputFileInfos []decoderInputFileInfo
	for _, fileID := range fileIDs {
		descriptionPacket, ok := fileDescriptionPackets[fileID]
		if !ok {
			return nil, errors.New("file description packet not found")
		}
		ifscPacket, ok := ifscPackets[fileID]
		if !ok {
			return nil, errors.New("input file slice checksum packet not found")
		}
		// TODO: Once we're not loading entire files in
		// memory, make decoderInputFileInfo.byteCount to an
		// int64.
		byteCount := int(descriptionPacket.byteCount)
		decoderInputFileInfos = append(decoderInputFileInfos, decoderInputFileInfo{
			fileID,
			descriptionPacket.filename,
			byteCount,
			descriptionPacket.hash16k,
			descriptionPacket.hash,
			ifscPacket.checksumPairs,
		})
	}

	return decoderInputFileInfos, nil
}

type shardLocation struct {
	fileID fileID
	start  int
}

type shardLocationSet map[shardLocation]bool

type checksumShardLocationMap map[uint32]map[[md5.Size]byte]shardLocationSet

func (m checksumShardLocationMap) put(crc32 uint32, md5Hash [md5.Size]byte, location shardLocation) {
	byCRC, ok := m[crc32]
	if !ok {
		m[crc32] = make(map[[md5.Size]byte]shardLocationSet)
		byCRC = m[crc32]
	}
	byMD5, ok := byCRC[md5Hash]
	if !ok {
		byCRC[md5Hash] = make(shardLocationSet)
		byMD5 = byCRC[md5Hash]
	}
	byMD5[location] = true
}

func (m checksumShardLocationMap) get(crc32 uint32, data []byte) shardLocationSet {
	byCRC := m[crc32]
	if len(byCRC) == 0 {
		return nil
	}
	return byCRC[md5.Sum(data)]
}

func makeChecksumShardLocationMap(sliceByteCount int, infos []decoderInputFileInfo) checksumShardLocationMap {
	m := make(checksumShardLocationMap)

	for _, info := range infos {
		for i, checksumPair := range info.checksumPairs {
			// TODO: Handle overflow.
			start := i * sliceByteCount
			m.put(binary.LittleEndian.Uint32(checksumPair.CRC32[:]), checksumPair.MD5, shardLocation{info.fileID, start})
		}
	}

	return m
}

type shardIntegrityInfo struct {
	data      []byte
	locations shardLocationSet
}

func (info shardIntegrityInfo) ok(location shardLocation) bool {
	return len(info.data) != 0 && info.locations[location]
}

type fileIntegrityInfo struct {
	fileID            fileID
	missing           bool
	hashMismatch      bool
	hasWrongByteCount bool
	shardInfos        []shardIntegrityInfo
}

func (info fileIntegrityInfo) allShardsOK(sliceByteCount int) bool {
	for i, shardInfo := range info.shardInfos {
		// TODO: Handle overflow.
		start := i * sliceByteCount
		if !shardInfo.ok(shardLocation{info.fileID, start}) {
			return false
		}
	}
	return true
}

func (info fileIntegrityInfo) ok(sliceByteCount int) bool {
	return !info.missing && !info.hashMismatch && !info.hasWrongByteCount && info.allShardsOK(sliceByteCount)
}

// A Decoder keeps track of all information needed to check the
// integrity of a set of data files, and possibly repair any
// missing/corrupt data files from the parity files (that usually end
// in .par2).
type Decoder struct {
	fs       fs.FS
	delegate DecoderDelegate

	indexPath string

	setID          recoverySetID
	clientID       string
	sliceByteCount int
	recoverySet    []decoderInputFileInfo
	nonRecoverySet []decoderInputFileInfo

	numGoroutines int

	checksumToLocation checksumShardLocationMap

	// Indexed the same as recoverySet.
	fileIntegrityInfos []fileIntegrityInfo

	parityShards [][]byte
}

// DecoderDelegate holds methods that are called during the decode
// process.
type DecoderDelegate interface {
	OnCreatorPacketLoad(clientID string)
	OnMainPacketLoad(sliceByteCount, recoverySetCount, nonRecoverySetCount int)
	OnFileDescriptionPacketLoad(fileID [16]byte, filename string, byteCount int64)
	OnIFSCPacketLoad(fileID [16]byte)
	OnRecoveryPacketLoad(exponent uint16, byteCount int)
	OnUnknownPacketLoad(packetType [16]byte, byteCount int)
	OnOtherPacketSkip(setID [16]byte, packetType [16]byte, byteCount int)
	OnDataFileLoad(i, n int, path string, byteCount, hits, misses int, err error)
	OnParityFileLoad(i int, path string, err error)
	OnDetectCorruptDataChunk(fileID [16]byte, path string, startByteOffset, endByteOffset int)
	OnDetectDataFileHashMismatch(fileID [16]byte, path string, err error)
	OnDetectDataFileWrongByteCount(fileID [16]byte, path string)
	OnDataFileWrite(i, n int, path string, byteCount int, err error)
}

// DoNothingDecoderDelegate is an implementation of DecoderDelegate
// that does nothing for all methods.
type DoNothingDecoderDelegate struct{}

// OnCreatorPacketLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnCreatorPacketLoad(clientID string) {}

// OnMainPacketLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnMainPacketLoad(sliceByteCount, recoverySetCount, nonRecoverySetCount int) {
}

// OnFileDescriptionPacketLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnFileDescriptionPacketLoad(fileID [16]byte, filename string, byteCount int64) {
}

// OnIFSCPacketLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnIFSCPacketLoad(fileID [16]byte) {}

// OnRecoveryPacketLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnRecoveryPacketLoad(exponent uint16, byteCount int) {}

// OnUnknownPacketLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnUnknownPacketLoad(packetType [16]byte, byteCount int) {}

// OnOtherPacketSkip implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnOtherPacketSkip(setID [16]byte, packetType [16]byte, byteCount int) {
}

// OnDataFileLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnDataFileLoad(i, n int, path string, byteCount, hits, misses int, err error) {
}

// OnParityFileLoad implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnParityFileLoad(i int, path string, err error) {}

// OnDetectCorruptDataChunk implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnDetectCorruptDataChunk(fileID [16]byte, path string, startByteOffset, endByteOffset int) {
}

// OnDetectDataFileHashMismatch implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnDetectDataFileHashMismatch(fileID [16]byte, path string, err error) {
}

// OnDetectDataFileWrongByteCount implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnDetectDataFileWrongByteCount(fileID [16]byte, path string) {}

// OnDataFileWrite implements the DecoderDelegate interface.
func (DoNothingDecoderDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {}

func newDecoder(filesystem fs.FS, delegate DecoderDelegate, indexPath string, numGoroutines int) (*Decoder, error) {
	readStream, err := filesystem.GetReadStream(indexPath)
	if err != nil {
		return nil, err
	}
	indexBytes, err := fs.ReadAndClose(readStream)
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

	recoverySet, err := makeDecoderInputFileInfos(indexFile.mainPacket.recoverySet, indexFile.fileDescriptionPackets, indexFile.ifscPackets)
	if err != nil {
		return nil, err
	}

	nonRecoverySet, err := makeDecoderInputFileInfos(indexFile.mainPacket.nonRecoverySet, indexFile.fileDescriptionPackets, indexFile.ifscPackets)
	if err != nil {
		return nil, err
	}

	return &Decoder{
		filesystem, delegate,
		indexPath,
		setID,
		indexFile.clientID, indexFile.mainPacket.sliceByteCount,
		recoverySet, nonRecoverySet,
		numGoroutines,
		nil,
		nil,
		nil,
	}, nil
}

func sliceAndPadByteArray(bs []byte, start, end int) []byte {
	padLength := 0
	if end > len(bs) {
		padLength = end - len(bs)
		end = len(bs)
	}
	slice := bs[start:end]
	if padLength > 0 {
		slice = append(slice, make([]byte, padLength)...)
	}
	return slice
}

func fillShardInfos(sliceByteCount int, data []byte, checksumToLocation checksumShardLocationMap, fileID fileID, fileIntegrityInfos []fileIntegrityInfo, fileIDIndices map[fileID]int) (int, int) {
	hits := 0
	misses := 0

	justMissed := false
	window := newCRC32Window(sliceByteCount)
	var crcSlice uint32
	for j := 0; j < len(data); {
		slice := sliceAndPadByteArray(data, j, j+sliceByteCount)
		if justMissed {
			crcSlice = window.update(crcSlice, data[j-1], slice[len(slice)-1])
		} else {
			crcSlice = crc32.ChecksumIEEE(slice)
		}
		foundLocations := checksumToLocation.get(crcSlice, slice)
		if len(foundLocations) == 0 {
			j++
			misses++
			justMissed = true
			continue
		}

		location := shardLocation{fileID, j}
		for foundLocation := range foundLocations {
			integrityInfo := fileIntegrityInfos[fileIDIndices[foundLocation.fileID]]
			shardInfo := &integrityInfo.shardInfos[foundLocation.start/sliceByteCount]
			if shardInfo.data == nil {
				*shardInfo = shardIntegrityInfo{
					slice,
					shardLocationSet{},
				}
			}
			shardInfo.locations[location] = true
		}

		justMissed = false
		j += sliceByteCount
		hits++
	}

	return hits, misses
}

func (d *Decoder) getFilePath(info decoderInputFileInfo) string {
	// TODO: Make this configurable.
	basePath := filepath.Dir(d.indexPath)
	return filepath.Join(basePath, info.filename)
}

func (d *Decoder) fillFileIntegrityInfos(checksumToLocation checksumShardLocationMap, fileIntegrityInfos []fileIntegrityInfo, fileIDIndices map[fileID]int, i int, info decoderInputFileInfo) (int, int, int, error) {
	path := d.getFilePath(info)
	readStream, err := d.fs.GetReadStream(path)
	if os.IsNotExist(err) {
		fileIntegrityInfos[i].missing = true
		return 0, 0, 0, nil
	} else if err != nil {
		return 0, 0, 0, err
	}
	hasher := hashutil.MakeMD5HashCheckerWith16k(info.hash16k, info.hash, false)
	data, err := fs.ReadAndClose(hashutil.TeeReadStream(readStream, hasher))
	hashErr, ok := err.(hashutil.HashMismatchError)
	if ok {
		err = nil
	} else if err != nil {
		return len(data), 0, 0, nil
	}

	if ok {
		fileIntegrityInfos[i].hashMismatch = true
		d.delegate.OnDetectDataFileHashMismatch(info.fileID, path, hashErr)
	} else {
		fileIntegrityInfos[i].hashMismatch = false
	}

	hits, misses := fillShardInfos(d.sliceByteCount, data, checksumToLocation, info.fileID, fileIntegrityInfos, fileIDIndices)

	hasWrongByteCount := len(data) != info.byteCount
	fileIntegrityInfos[i].hasWrongByteCount = hasWrongByteCount
	if hasWrongByteCount {
		d.delegate.OnDetectDataFileWrongByteCount(info.fileID, path)
	}

	return len(data), hits, misses, nil
}

// LoadFileData loads existing file data into memory.
func (d *Decoder) LoadFileData() error {
	checksumToLocation := makeChecksumShardLocationMap(d.sliceByteCount, d.recoverySet)

	fileIntegrityInfos := make([]fileIntegrityInfo, len(d.recoverySet))
	fileIDIndices := make(map[fileID]int)
	for i, info := range d.recoverySet {
		fileIntegrityInfos[i] = fileIntegrityInfo{
			fileID:     info.fileID,
			shardInfos: make([]shardIntegrityInfo, len(info.checksumPairs)),
		}
		fileIDIndices[info.fileID] = i
	}

	for i, info := range d.recoverySet {
		path := d.getFilePath(info)
		byteCount, hits, misses, err := d.fillFileIntegrityInfos(checksumToLocation, fileIntegrityInfos, fileIDIndices, i, info)
		d.delegate.OnDataFileLoad(i+1, len(d.recoverySet), path, byteCount, hits, misses, err)
		if err != nil {
			return err
		}

		if byteCount != info.byteCount {
			var startByteOffset, endByteOffset int
			if byteCount < info.byteCount {
				startByteOffset = byteCount
				endByteOffset = info.byteCount
			} else {
				startByteOffset = info.byteCount
				endByteOffset = byteCount
			}
			d.delegate.OnDetectCorruptDataChunk(info.fileID, path, startByteOffset, endByteOffset)
		}
	}

	for i, info := range d.recoverySet {
		integrityInfo := fileIntegrityInfos[i]
		corruptStartByteOffset := -1
		corruptEndByteOffset := -1
		for j, shardInfo := range integrityInfo.shardInfos {
			startByteOffset := j * d.sliceByteCount
			endByteOffset := startByteOffset + d.sliceByteCount
			if endByteOffset > info.byteCount {
				endByteOffset = info.byteCount
			}
			if shardInfo.ok(shardLocation{info.fileID, startByteOffset}) {
				if corruptStartByteOffset != -1 {
					d.delegate.OnDetectCorruptDataChunk(info.fileID, d.getFilePath(info), corruptStartByteOffset, corruptEndByteOffset)
					corruptStartByteOffset = -1
					corruptEndByteOffset = -1
				}
			} else {
				if corruptStartByteOffset == -1 {
					corruptStartByteOffset = startByteOffset
				}
				corruptEndByteOffset = endByteOffset
			}
		}

		if corruptStartByteOffset != -1 {
			d.delegate.OnDetectCorruptDataChunk(info.fileID, d.getFilePath(info), corruptStartByteOffset, info.byteCount)
		}
	}

	d.checksumToLocation = checksumToLocation
	d.fileIntegrityInfos = fileIntegrityInfos
	return nil
}

type recoveryDelegate struct {
	d DecoderDelegate
}

func (recoveryDelegate) OnCreatorPacketLoad(clientID string) {}

func (recoveryDelegate) OnMainPacketLoad(sliceByteCount, recoverySetCount, nonRecoverySetCount int) {}

func (recoveryDelegate) OnFileDescriptionPacketLoad(fileID [16]byte, filename string, byteCount int64) {
}

func (recoveryDelegate) OnIFSCPacketLoad(fileID [16]byte) {}

func (r recoveryDelegate) OnRecoveryPacketLoad(exponent uint16, byteCount int) {
	r.d.OnRecoveryPacketLoad(exponent, byteCount)
}

func (r recoveryDelegate) OnUnknownPacketLoad(packetType [16]byte, byteCount int) {
	r.d.OnUnknownPacketLoad(packetType, byteCount)
}

func (recoveryDelegate) OnOtherPacketSkip(setID [16]byte, packetType [16]byte, byteCount int) {}

func (recoveryDelegate) OnDataFileLoad(i, n int, path string, byteCount, hits, misses int, err error) {
}

func (recoveryDelegate) OnParityFileLoad(i int, path string, err error) {}

func (recoveryDelegate) OnDetectCorruptDataChunk(fileID [16]byte, path string, startByteOffset, endByteOffset int) {
}

func (recoveryDelegate) OnDetectDataFileHashMismatch(fileID [16]byte, path string, err error) {}

func (recoveryDelegate) OnDetectDataFileWrongByteCount(fileID [16]byte, path string) {}

func (recoveryDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {}

// LoadParityData searches for parity volumes and loads them into
// memory.
func (d *Decoder) LoadParityData() error {
	ext := path.Ext(d.indexPath)
	base := d.indexPath[:len(d.indexPath)-len(ext)]
	matches, err := d.fs.FindWithPrefixAndSuffix(base+".", ext)
	if err != nil {
		return err
	}

	var parityFiles []file
	for i, match := range matches {
		parityFile, err := func() (*file, error) {
			readStream, err := d.fs.GetReadStream(match)
			if err != nil {
				return nil, err
			}
			volumeBytes, err := fs.ReadAndClose(readStream)
			if err != nil {
				return nil, err
			}

			// Ignore all the other packet types other
			// than recovery packets.
			_, parityFile, err := readFile(recoveryDelegate{d.delegate}, &d.setID, volumeBytes)
			if _, ok := err.(noPacketsFoundError); ok {
				return nil, nil
			} else if err != nil {
				// TODO: Relax this check.
				return nil, err
			}

			if d.sliceByteCount != parityFile.mainPacket.sliceByteCount {
				return nil, errors.New("slice byte count mismatch")
			}

			if !reflect.DeepEqual(decoderInputFileInfoIDs(d.recoverySet), parityFile.mainPacket.recoverySet) {
				return nil, errors.New("recovery set mismatch")
			}

			if !reflect.DeepEqual(decoderInputFileInfoIDs(d.nonRecoverySet), parityFile.mainPacket.nonRecoverySet) {
				return nil, errors.New("non-recovery set mismatch")
			}

			return &parityFile, nil
		}()
		d.delegate.OnParityFileLoad(i+1, match, err)
		if err != nil {
			return err
		}
		if parityFile == nil {
			continue
		}

		parityFiles = append(parityFiles, *parityFile)
	}

	var parityShards [][]byte
	for _, file := range parityFiles {
		for exponent, packet := range file.recoveryPackets {
			if int(exponent) >= len(parityShards) {
				parityShards = append(parityShards, make([][]byte, int(exponent+1)-len(parityShards))...)
			}
			parityShards[exponent] = packet.data
		}
	}

	d.parityShards = parityShards
	return nil
}

func (d *Decoder) newCoderAndShards() (rsec16.Coder, [][]byte, error) {
	if len(d.fileIntegrityInfos) == 0 {
		return rsec16.Coder{}, nil, errors.New("no file integrity info")
	}

	if len(d.parityShards) == 0 {
		return rsec16.Coder{}, nil, errors.New("no parity shards")
	}

	var dataShards [][]byte
	for _, info := range d.fileIntegrityInfos {
		for _, shardInfo := range info.shardInfos {
			dataShards = append(dataShards, shardInfo.data)
		}
	}
	coder, err := rsec16.NewCoderPAR2Vandermonde(len(dataShards), len(d.parityShards), d.numGoroutines)
	if err != nil {
		return rsec16.Coder{}, nil, err
	}

	return coder, dataShards, err
}

// ShardCounts contains shard counts which can be used to deduce
// whether repair is necessary and/or possible.
type ShardCounts struct {
	// UsableDataShardCount is the number of data shards that are
	// usable, i.e. not missing and not corrupt.
	UsableDataShardCount int
	// UnusableDataShardCount is the number of data shards that
	// are unusable, i.e. missing or corrupt.
	UnusableDataShardCount int

	// UsableParityShardCount is the number of parity shards that
	// exist, i.e. not missing and not corrupt.
	UsableParityShardCount int
	// UnusableDataShardCount is the number of parity shards that
	// are unusable, i.e. missing or corrupt.
	UnusableParityShardCount int
}

// RepairNeeded returns whether repair is needed, i.e. whether
// UnusableDataShardCount is non-zero.
func (fc ShardCounts) RepairNeeded() bool {
	return fc.UnusableDataShardCount > 0
}

// RepairPossible returns whether repair is possible i.e. whether
// UsableParityShardCount >= UnusableDataShardCount.
func (fc ShardCounts) RepairPossible() bool {
	return fc.UsableParityShardCount >= fc.UnusableDataShardCount
}

// ShardCounts returns a ShardCounts object for the current shard set.
func (d *Decoder) ShardCounts() ShardCounts {
	usableDataShardCount := 0
	unusableDataShardCount := 0

	for _, info := range d.fileIntegrityInfos {
		for _, shardInfo := range info.shardInfos {
			if shardInfo.data == nil {
				unusableDataShardCount++
			} else {
				usableDataShardCount++
			}
		}
	}

	usableParityShardCount := 0
	unusableParityShardCount := 0

	for _, shard := range d.parityShards {
		if shard == nil {
			unusableParityShardCount++
		} else {
			usableParityShardCount++
		}
	}

	return ShardCounts{
		UsableDataShardCount:     usableDataShardCount,
		UnusableDataShardCount:   unusableDataShardCount,
		UsableParityShardCount:   usableParityShardCount,
		UnusableParityShardCount: unusableParityShardCount,
	}
}

// Repair tries to repair any missing or corrupt data, using the
// parity volumes. Returns a list of paths to files that were
// successfully repaired (relative to the indexFile passed to
// NewDecoder) in no particular order, which is present even if an
// error is returned. If checkParity is true, extra checking is done
// of the reconstructed parity data.
func (d *Decoder) Repair(checkParity bool) ([]string, error) {
	coder, dataShards, err := d.newCoderAndShards()
	if err != nil {
		return nil, err
	}

	err = coder.ReconstructData(dataShards, d.parityShards)
	if err != nil {
		return nil, err
	}

	if checkParity {
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
	}

	wasOK := make([]bool, len(d.fileIntegrityInfos))

	k := 0
	for i, info := range d.fileIntegrityInfos {
		wasOK[i] = info.ok(d.sliceByteCount)
		shardCount := len(info.shardInfos)
		for j, shard := range dataShards[k : k+shardCount] {
			info.shardInfos[j] = shardIntegrityInfo{
				data:      shard,
				locations: d.checksumToLocation.get(crc32.ChecksumIEEE(shard), shard),
			}
		}
		k += shardCount
		d.fileIntegrityInfos[i] = info
	}

	var repairedPaths []string

	for i, decoderInputFileInfo := range d.recoverySet {
		fileIntegrityInfo := d.fileIntegrityInfos[i]
		if wasOK[i] {
			continue
		}

		buf := bytes.NewBuffer(nil)
		for _, shardInfo := range fileIntegrityInfo.shardInfos {
			err := binary.Write(buf, binary.LittleEndian, shardInfo.data)
			if err != nil {
				return repairedPaths, err
			}
		}

		data := buf.Bytes()[:decoderInputFileInfo.byteCount]
		if err := hashutil.CheckMD5Hashes(data, decoderInputFileInfo.hash16k, decoderInputFileInfo.hash, true); err != nil {
			return repairedPaths, err
		}

		path := d.getFilePath(decoderInputFileInfo)
		err = func() error {
			writeStream, err := d.fs.GetWriteStream(path)
			if err != nil {
				return err
			}
			return fs.WriteAndClose(writeStream, data)
		}()
		d.delegate.OnDataFileWrite(i+1, len(d.recoverySet), path, len(data), err)
		if err != nil {
			return repairedPaths, err
		}

		repairedPaths = append(repairedPaths, path)
	}

	// TODO: Repair missing parity volumes, too, and then make
	// sure d.Verify() passes.

	return repairedPaths, nil
}

// NewDecoder reads the given index file, which usually has a .par2
// extension.
func NewDecoder(delegate DecoderDelegate, indexFile string, numGoroutines int) (*Decoder, error) {
	return newDecoder(fs.MakeDefaultFS(), delegate, indexFile, numGoroutines)
}
