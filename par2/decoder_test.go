package par2

import (
	"os"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/akalin/gopar/rsec16"
	"github.com/stretchr/testify/require"
)

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
	dataShardsByID := make(map[fileID][][]uint16)
	for filename, data := range io.fileData {
		fileID, fileDescriptionPacket, ifscPacket, fileDataShards := makeTestFileInfo(sliceByteCount, filename, data)
		recoverySet = append(recoverySet, fileID)
		fileDescriptionPackets[fileID] = fileDescriptionPacket
		ifscPackets[fileID] = ifscPacket
		dataShardsByID[fileID] = fileDataShards
	}

	sort.Slice(recoverySet, func(i, j int) bool {
		return fileIDLess(recoverySet[i], recoverySet[j])
	})

	var dataShards [][]uint16
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

	recoveryFile := indexFile
	recoveryFile.recoveryPackets = recoveryPackets
	_, recoveryFileBytes, err := writeFile(recoveryFile)
	require.NoError(t, err)
	io.fileData[base+".all.par2"] = recoveryFileBytes
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
}