package par2

import (
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"math/rand"
	"path"
	"path/filepath"
	"sort"
	"testing"

	"github.com/akalin/gopar/memfs"
	"github.com/akalin/gopar/rsec16"
	"github.com/stretchr/testify/require"
)

func makeTestFillShardInfoInputs(tb testing.TB, sliceByteCount, dataByteCount int) (fileID, []byte, checksumShardLocationMap, []fileIntegrityInfo, map[fileID]int, []byte) {
	rand := rand.New(rand.NewSource(1))

	id := fileID{0x1}
	data := make([]byte, dataByteCount)
	n, err := rand.Read(data)
	require.NoError(tb, err)
	require.Equal(tb, dataByteCount, n)

	checksumToLocation := make(checksumShardLocationMap)
	for i := 0; i < dataByteCount; i += sliceByteCount {
		slice := sliceAndPadByteArray(data, i, i+sliceByteCount)
		crc32 := crc32.ChecksumIEEE(slice)
		md5 := md5.Sum(slice)
		checksumToLocation.put(crc32, md5, shardLocation{id, i})
	}

	fileIntegrityInfos := []fileIntegrityInfo{{
		shardInfos: make([]shardIntegrityInfo, (dataByteCount+sliceByteCount-1)/sliceByteCount),
	}}
	fileIDIndices := map[fileID]int{id: 0}

	unrelatedData := make([]byte, dataByteCount)
	n, err = rand.Read(unrelatedData)
	require.NoError(tb, err)
	require.Equal(tb, dataByteCount, n)

	return id, data, checksumToLocation, fileIntegrityInfos, fileIDIndices, unrelatedData
}

func TestFillShardInfos(t *testing.T) {
	sliceByteCount := 4
	dataByteCount := 50
	id, data, checksumToLocation, fileIntegrityInfos, fileIDIndices, unrelatedData := makeTestFillShardInfoInputs(t, sliceByteCount, dataByteCount)

	hits, misses := fillShardInfos(sliceByteCount, data, checksumToLocation, id, fileIntegrityInfos, fileIDIndices)
	expectedHits := (dataByteCount + sliceByteCount - 1) / sliceByteCount
	require.Equal(t, expectedHits, hits)
	require.Equal(t, 0, misses)

	hits, misses = fillShardInfos(sliceByteCount, unrelatedData, checksumToLocation, id, fileIntegrityInfos, fileIDIndices)
	require.Equal(t, 0, hits)
	require.Equal(t, dataByteCount, misses)
}

func BenchmarkFillShardInfos(b *testing.B) {
	sliceByteCount := 2000
	dataByteCount := 1024 * 1024
	b.SetBytes(int64(dataByteCount))

	id, data, checksumToLocation, fileIntegrityInfos, fileIDIndices, unrelatedData := makeTestFillShardInfoInputs(b, sliceByteCount, dataByteCount)

	b.Run("related", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fillShardInfos(sliceByteCount, data, checksumToLocation, id, fileIntegrityInfos, fileIDIndices)
		}
	})
	b.Run("unrelated", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fillShardInfos(sliceByteCount, unrelatedData, checksumToLocation, id, fileIntegrityInfos, fileIDIndices)
		}
	})
}

type testFileIO struct {
	t *testing.T
	fileIO
}

func (io testFileIO) ReadFile(path string) (data []byte, err error) {
	io.t.Helper()
	defer func() {
		io.t.Helper()
		io.t.Logf("ReadFile(%s) => (%d bytes, %v)", path, len(data), err)
	}()
	return io.fileIO.ReadFile(path)
}

func (io testFileIO) FindWithPrefixAndSuffix(prefix, suffix string) (matches []string, err error) {
	io.t.Helper()
	defer func() {
		io.t.Helper()
		io.t.Logf("FindWithPrefixAndSuffix(%s, %s) => (%d files, %v)", prefix, suffix, len(matches), err)
	}()
	return io.fileIO.FindWithPrefixAndSuffix(prefix, suffix)
}

func (io testFileIO) WriteFile(path string, data []byte) (err error) {
	io.t.Helper()
	defer func() {
		io.t.Helper()
		io.t.Logf("WriteFile(%s, %d bytes) => %v", path, len(data), err)
	}()
	return io.fileIO.WriteFile(path, data)
}

func buildPAR2Data(t *testing.T, fs memfs.MemFS, sliceByteCount, parityShardCount int) {
	var recoverySet []fileID
	fileDescriptionPackets := make(map[fileID]fileDescriptionPacket)
	ifscPackets := make(map[fileID]ifscPacket)
	dataShardsByID := make(map[fileID][][]byte)
	paths := fs.Paths()
	for _, path := range paths {
		data, err := fs.ReadFile(path)
		require.NoError(t, err)
		fileID, fileDescriptionPacket, ifscPacket, fileDataShards := computeDataFileInfo(sliceByteCount, filepath.Base(path), data)
		recoverySet = append(recoverySet, fileID)
		fileDescriptionPackets[fileID] = fileDescriptionPacket
		ifscPackets[fileID] = ifscPacket
		dataShardsByID[fileID] = fileDataShards
	}

	sort.Slice(recoverySet, func(i, j int) bool {
		return fileIDLess(recoverySet[i], recoverySet[j])
	})

	var dataShards [][]byte
	for _, fileID := range recoverySet {
		dataShards = append(dataShards, dataShardsByID[fileID]...)
	}

	coder, err := rsec16.NewCoderPAR2Vandermonde(len(dataShards), parityShardCount, rsec16.DefaultNumGoroutines())
	require.NoError(t, err)

	parityShards := coder.GenerateParity(dataShards)
	recoveryPackets := make(map[exponent]recoveryPacket)
	for i, parityShard := range parityShards {
		recoveryPackets[exponent(i)] = recoveryPacket{data: parityShard}
	}

	mainPacket := mainPacket{
		sliceByteCount: sliceByteCount,
		recoverySet:    recoverySet,
	}

	indexFile := file{
		clientID:               "test client",
		mainPacket:             &mainPacket,
		fileDescriptionPackets: fileDescriptionPackets,
		ifscPackets:            ifscPackets,
	}

	_, indexFileBytes, err := writeFile(indexFile)
	require.NoError(t, err)

	// Require that all files have the same base.
	var base string
	for _, filename := range paths {
		ext := path.Ext(filename)
		filenameBase := filename[:len(filename)-len(ext)]
		if base == "" {
			base = filenameBase
		} else {
			require.Equal(t, base, filenameBase)
		}
		break
	}
	require.NotEmpty(t, base)

	require.NoError(t, fs.WriteFile(base+".par2", indexFileBytes))

	for exp, packet := range recoveryPackets {
		recoveryFile := indexFile
		recoveryFile.recoveryPackets = map[exponent]recoveryPacket{
			exp: packet,
		}
		_, recoveryFileBytes, err := writeFile(recoveryFile)
		require.NoError(t, err)
		filename := fmt.Sprintf("%s.vol%02d+01.par2", base, exp)
		require.NoError(t, fs.WriteFile(filename, recoveryFileBytes))
	}
}

func newDecoderForTest(t *testing.T, fs memfs.MemFS, indexPath string) (*Decoder, error) {
	return newDecoder(testFileIO{t, fs}, testDecoderDelegate{t}, indexPath, rsec16.DefaultNumGoroutines())
}

func makeDecoderMemFS() memfs.MemFS {
	return memfs.MakeMemFS(memfs.RootDir(), map[string][]byte{
		"file.rar": {0x1, 0x2, 0x3, 0x4},
		"file.r01": {0x5, 0x6, 0x7},
		"file.r02": {0x8, 0x9, 0xa, 0xb, 0xc},
		"file.r03": {0xe, 0xf},
		"file.r04": {0xd},
	})
}

func TestVerify(t *testing.T) {
	fs := makeDecoderMemFS()

	buildPAR2Data(t, fs, 4, 3)

	decoder, err := newDecoderForTest(t, fs, "file.par2")
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	needsRepair, err := decoder.Verify()
	require.NoError(t, err)
	require.False(t, needsRepair)

	fileData5, err := fs.ReadFile("file.r04")
	require.NoError(t, err)
	fileData5[len(fileData5)-1]++
	err = decoder.LoadFileData()
	require.NoError(t, err)
	needsRepair, err = decoder.Verify()
	require.NoError(t, err)
	require.True(t, needsRepair)

	fileData5[len(fileData5)-1]--
	err = decoder.LoadFileData()
	require.NoError(t, err)
	needsRepair, err = decoder.Verify()
	require.NoError(t, err)
	require.False(t, needsRepair)

	vol2Data, err := fs.RemoveFile("file.vol02+01.par2")
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)
	needsRepair, err = decoder.Verify()
	require.NoError(t, err)
	require.False(t, needsRepair)

	require.NoError(t, fs.WriteFile("file.vol02+01.par2", vol2Data))
	_, err = fs.RemoveFile("file.vol01+01.par2")
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)
	needsRepair, err = decoder.Verify()
	require.NoError(t, err)
	require.False(t, needsRepair)
}

func TestSetIDMismatch(t *testing.T) {
	fs1 := makeDecoderMemFS()
	fs2 := makeDecoderMemFS()
	rarData, err := fs2.ReadFile("file.rar")
	require.NoError(t, err)
	rarData[0]++

	buildPAR2Data(t, fs1, 4, 3)
	buildPAR2Data(t, fs2, 4, 3)
	// Insert a parity volume that has a different set hash.
	vol1Data, err := fs2.ReadFile("file.vol01+01.par2")
	require.NoError(t, err)
	require.NoError(t, fs1.WriteFile("file.vol01+01.par2", vol1Data))

	decoder, err := newDecoderForTest(t, fs1, "file.par2")
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)
	needsRepair, err := decoder.Verify()
	require.NoError(t, err)
	require.False(t, needsRepair)
}

func TestRepair(t *testing.T) {
	fs := makeDecoderMemFS()

	buildPAR2Data(t, fs, 4, 3)

	decoder, err := newDecoderForTest(t, fs, "file.par2")
	require.NoError(t, err)

	r02Data, err := fs.ReadFile("file.r02")
	require.NoError(t, err)
	r02DataCopy := make([]byte, len(r02Data))
	copy(r02DataCopy, r02Data)
	r02Data[len(r02Data)-1]++
	r03Data, err := fs.RemoveFile("file.r03")
	require.NoError(t, err)
	r04Data, err := fs.RemoveFile("file.r04")
	require.NoError(t, err)

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repairedPaths, err := decoder.Repair(true)
	require.NoError(t, err)

	require.Equal(t, []string{"file.r02", "file.r03", "file.r04"}, repairedPaths)
	repairedR02Data, err := fs.ReadFile("file.r02")
	require.NoError(t, err)
	require.Equal(t, r02DataCopy, repairedR02Data)
	repairedR03Data, err := fs.ReadFile("file.r03")
	require.NoError(t, err)
	require.Equal(t, r03Data, repairedR03Data)
	repairedR04Data, err := fs.ReadFile("file.r04")
	require.NoError(t, err)
	require.Equal(t, r04Data, repairedR04Data)
}

func TestRepairAddedBytes(t *testing.T) {
	fs := memfs.MakeMemFS(memfs.RootDir(), map[string][]byte{
		"file.rar": {
			0x01, 0x02, 0x03, 0x04, 0x05,
			0x11, 0x12, 0x13, 0x14, 0x15,
			0x21, 0x22, 0x23, 0x24, 0x25,
			0x31, 0x32, 0x33, 0x34, 0x35,
		},
	})

	buildPAR2Data(t, fs, 4, 3)

	decoder, err := newDecoderForTest(t, fs, "file.par2")
	require.NoError(t, err)

	rarData, err := fs.ReadFile("file.rar")
	require.NoError(t, err)
	rarDataCopy := make([]byte, len(rarData))
	copy(rarDataCopy, rarData)
	require.NoError(t, fs.WriteFile("file.rar", append([]byte{0x00}, rarData...)))

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repairedPaths, err := decoder.Repair(true)
	require.NoError(t, err)

	require.Equal(t, []string{"file.rar"}, repairedPaths)
	repairedRarData, err := fs.ReadFile("file.rar")
	require.NoError(t, err)
	require.Equal(t, rarDataCopy, repairedRarData)
}

func TestRepairRemovedBytes(t *testing.T) {
	fs := memfs.MakeMemFS(memfs.RootDir(), map[string][]byte{
		"file.rar": {
			0x01, 0x02, 0x03, 0x04, 0x05,
			0x11, 0x12, 0x13, 0x14, 0x15,
			0x21, 0x22, 0x23, 0x24, 0x25,
			0x31, 0x32, 0x33, 0x34, 0x35,
		},
	})

	buildPAR2Data(t, fs, 4, 3)

	decoder, err := newDecoderForTest(t, fs, "file.par2")
	require.NoError(t, err)

	rarData, err := fs.ReadFile("file.rar")
	require.NoError(t, err)
	rarDataCopy := make([]byte, len(rarData))
	copy(rarDataCopy, rarData)
	require.NoError(t, fs.WriteFile("file.rar", rarData[2:]))

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repairedPaths, err := decoder.Repair(true)
	require.NoError(t, err)

	require.Equal(t, []string{"file.rar"}, repairedPaths)
	repairedRarData, err := fs.ReadFile("file.rar")
	require.NoError(t, err)
	require.Equal(t, rarDataCopy, repairedRarData)
}

func TestRepairSwappedFiles(t *testing.T) {
	fs := memfs.MakeMemFS(memfs.RootDir(), map[string][]byte{
		"file.rar": {
			0x01, 0x02, 0x03, 0x04, 0x05,
			0x11, 0x12, 0x13, 0x14, 0x15,
			0x21, 0x22, 0x23, 0x24, 0x25,
			0x31, 0x32, 0x33, 0x34, 0x35,
		},
		"file.r01": {
			0x41, 0x42, 0x43, 0x44, 0x45,
			0x51, 0x52, 0x53, 0x54, 0x55,
			0x61, 0x62, 0x63, 0x64, 0x65,
		},
	})

	buildPAR2Data(t, fs, 4, 3)

	decoder, err := newDecoderForTest(t, fs, "file.par2")
	require.NoError(t, err)

	rarData, err := fs.ReadFile("file.rar")
	require.NoError(t, err)
	r01Data, err := fs.ReadFile("file.r01")
	require.NoError(t, err)
	require.NoError(t, fs.WriteFile("file.rar", r01Data))
	require.NoError(t, fs.WriteFile("file.r01", rarData))

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repairedPaths, err := decoder.Repair(true)
	require.NoError(t, err)

	require.Equal(t, []string{"file.rar", "file.r01"}, repairedPaths)
	repairedRarData, err := fs.ReadFile("file.rar")
	require.NoError(t, err)
	require.Equal(t, rarData, repairedRarData)
	repairedR01Data, err := fs.ReadFile("file.r01")
	require.NoError(t, err)
	require.Equal(t, r01Data, repairedR01Data)
}
