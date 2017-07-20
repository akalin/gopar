package par2

import (
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"math/rand"
	"os"
	"path"
	"sort"
	"strings"
	"testing"

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
	expectedMisses := dataByteCount - expectedHits
	require.Equal(t, expectedHits, hits)
	require.Equal(t, expectedMisses, misses)

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
	t        *testing.T
	fileData map[string][]byte
}

func (io testFileIO) ReadFile(path string) (data []byte, err error) {
	defer func() {
		io.t.Logf("ReadFile(%s) => (%d, %v)", path, len(data), err)
	}()
	if data, ok := io.fileData[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (io testFileIO) FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error) {
	var matches []string
	for filename := range io.fileData {
		if len(filename) >= len(prefix)+len(suffix) && strings.HasPrefix(filename, prefix) && strings.HasSuffix(filename, suffix) {
			matches = append(matches, filename)
		}
	}
	io.t.Logf("FindWithPrefixAndSuffix(%s, %s) => %d files", prefix, suffix, len(matches))
	return matches, nil
}

func (io testFileIO) WriteFile(path string, data []byte) error {
	io.t.Logf("WriteFile(%s, %d bytes)", path, len(data))
	io.fileData[path] = data
	return nil
}

func buildPAR2Data(t *testing.T, io testFileIO, sliceByteCount, parityShardCount int) {
	var recoverySet []fileID
	fileDescriptionPackets := make(map[fileID]fileDescriptionPacket)
	ifscPackets := make(map[fileID]ifscPacket)
	dataShardsByID := make(map[fileID][][]byte)
	for filename, data := range io.fileData {
		fileID, fileDescriptionPacket, ifscPacket, fileDataShards := computeDataFileInfo(sliceByteCount, filename, data)
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

	coder, err := rsec16.NewCoderPAR2Vandermonde(len(dataShards), parityShardCount)
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
	for filename := range io.fileData {
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

	io.fileData[base+".par2"] = indexFileBytes

	for exp, packet := range recoveryPackets {
		recoveryFile := indexFile
		recoveryFile.recoveryPackets = map[exponent]recoveryPacket{
			exp: packet,
		}
		_, recoveryFileBytes, err := writeFile(recoveryFile)
		require.NoError(t, err)
		filename := fmt.Sprintf("%s.vol%02d+01.par2", base, exp)
		io.fileData[filename] = recoveryFileBytes
	}
}

func TestVerify(t *testing.T) {
	io := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3, 0x4},
			"file.r01": {0x5, 0x6, 0x7},
			"file.r02": {0x8, 0x9, 0xa, 0xb, 0xc},
			"file.r03": {0xe, 0xf},
			"file.r04": {0xd},
		},
	}

	buildPAR2Data(t, io, 4, 3)

	decoder, err := newDecoder(io, testDecoderDelegate{t}, "file.par2")
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	ok, err := decoder.Verify()
	require.NoError(t, err)
	require.True(t, ok)

	fileData5 := io.fileData["file.r04"]
	fileData5[len(fileData5)-1]++
	err = decoder.LoadFileData()
	require.NoError(t, err)
	ok, err = decoder.Verify()
	require.NoError(t, err)
	require.False(t, ok)

	fileData5[len(fileData5)-1]--
	err = decoder.LoadFileData()
	require.NoError(t, err)
	ok, err = decoder.Verify()
	require.NoError(t, err)
	require.True(t, ok)

	vol2Data := io.fileData["file.vol02+01.par2"]
	delete(io.fileData, "file.vol02+01.par2")
	err = decoder.LoadParityData()
	require.NoError(t, err)
	ok, err = decoder.Verify()
	require.NoError(t, err)
	require.True(t, ok)

	io.fileData["file.vol02+01.par2"] = vol2Data
	delete(io.fileData, "file.vol01+01.par2")
	err = decoder.LoadParityData()
	require.NoError(t, err)
	ok, err = decoder.Verify()
	require.NoError(t, err)
	require.False(t, ok)
}

func TestSetIDMismatch(t *testing.T) {
	io1 := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3, 0x4},
			"file.r01": {0x5, 0x6, 0x7},
			"file.r02": {0x8, 0x9, 0xa, 0xb, 0xc},
			"file.r03": {0xe, 0xf},
			"file.r04": {0xd},
		},
	}

	io2 := testFileIO{
		t:        t,
		fileData: make(map[string][]byte),
	}
	for k, v := range io1.fileData {
		io2.fileData[k] = make([]byte, len(v))
		copy(io2.fileData[k], v)
	}
	io2.fileData["file.rar"][0]++

	buildPAR2Data(t, io1, 4, 3)
	buildPAR2Data(t, io2, 4, 3)
	// Insert a parity volume that has a different set hash.
	io1.fileData["file.vol01+01.par2"] = io2.fileData["file.vol01+01.par2"]

	decoder, err := newDecoder(io1, testDecoderDelegate{t}, "file.par2")
	require.NoError(t, err)
	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)
	ok, err := decoder.Verify()
	require.NoError(t, err)
	require.False(t, ok)
}

func TestRepair(t *testing.T) {
	io := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": {0x1, 0x2, 0x3, 0x4},
			"file.r01": {0x5, 0x6, 0x7},
			"file.r02": {0x8, 0x9, 0xa, 0xb, 0xc},
			"file.r03": {0xe, 0xf},
			"file.r04": {0xd},
		},
	}

	buildPAR2Data(t, io, 4, 3)

	decoder, err := newDecoder(io, testDecoderDelegate{t}, "file.par2")
	require.NoError(t, err)

	r02Data := io.fileData["file.r02"]
	r02DataCopy := make([]byte, len(r02Data))
	copy(r02DataCopy, r02Data)
	r02Data[len(r02Data)-1]++
	r03Data := io.fileData["file.r03"]
	delete(io.fileData, "file.r03")
	r04Data := io.fileData["file.r04"]
	delete(io.fileData, "file.r04")

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repaired, err := decoder.Repair()
	require.NoError(t, err)

	require.Equal(t, []string{"file.r02", "file.r03", "file.r04"}, repaired)
	require.Equal(t, r02DataCopy, io.fileData["file.r02"])
	require.Equal(t, r03Data, io.fileData["file.r03"])
	require.Equal(t, r04Data, io.fileData["file.r04"])
}

func TestRepairAddedBytes(t *testing.T) {
	io := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": []byte{
				0x01, 0x02, 0x03, 0x04, 0x05,
				0x11, 0x12, 0x13, 0x14, 0x15,
				0x21, 0x22, 0x23, 0x24, 0x25,
				0x31, 0x32, 0x33, 0x34, 0x35,
			},
		},
	}

	buildPAR2Data(t, io, 4, 3)

	decoder, err := newDecoder(io, testDecoderDelegate{t}, "file.par2")
	require.NoError(t, err)

	rarData := io.fileData["file.rar"]
	rarDataCopy := make([]byte, len(rarData))
	copy(rarDataCopy, rarData)
	io.fileData["file.rar"] = append([]byte{0x00}, rarData...)

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repaired, err := decoder.Repair()
	require.NoError(t, err)

	require.Equal(t, []string{"file.rar"}, repaired)
	require.Equal(t, rarDataCopy, io.fileData["file.rar"])
}

func TestRepairRemovedBytes(t *testing.T) {
	io := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": []byte{
				0x01, 0x02, 0x03, 0x04, 0x05,
				0x11, 0x12, 0x13, 0x14, 0x15,
				0x21, 0x22, 0x23, 0x24, 0x25,
				0x31, 0x32, 0x33, 0x34, 0x35,
			},
		},
	}

	buildPAR2Data(t, io, 4, 3)

	decoder, err := newDecoder(io, testDecoderDelegate{t}, "file.par2")
	require.NoError(t, err)

	rarData := io.fileData["file.rar"]
	rarDataCopy := make([]byte, len(rarData))
	copy(rarDataCopy, rarData)
	io.fileData["file.rar"] = rarData[2:]

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repaired, err := decoder.Repair()
	require.NoError(t, err)

	require.Equal(t, []string{"file.rar"}, repaired)
	require.Equal(t, rarDataCopy, io.fileData["file.rar"])
}

func TestRepairSwappedFiles(t *testing.T) {
	io := testFileIO{
		t: t,
		fileData: map[string][]byte{
			"file.rar": []byte{
				0x01, 0x02, 0x03, 0x04, 0x05,
				0x11, 0x12, 0x13, 0x14, 0x15,
				0x21, 0x22, 0x23, 0x24, 0x25,
				0x31, 0x32, 0x33, 0x34, 0x35,
			},
			"file.r01": []byte{
				0x41, 0x42, 0x43, 0x44, 0x45,
				0x51, 0x52, 0x53, 0x54, 0x55,
				0x61, 0x62, 0x63, 0x64, 0x65,
			},
		},
	}

	buildPAR2Data(t, io, 4, 3)

	decoder, err := newDecoder(io, testDecoderDelegate{t}, "file.par2")
	require.NoError(t, err)

	rarData := io.fileData["file.rar"]
	r01Data := io.fileData["file.r01"]
	io.fileData["file.rar"] = r01Data
	io.fileData["file.r01"] = rarData

	err = decoder.LoadFileData()
	require.NoError(t, err)
	err = decoder.LoadParityData()
	require.NoError(t, err)

	repaired, err := decoder.Repair()
	require.NoError(t, err)

	require.Equal(t, []string{"file.rar", "file.r01"}, repaired)
	require.Equal(t, rarData, io.fileData["file.rar"])
	require.Equal(t, r01Data, io.fileData["file.r01"])
}
