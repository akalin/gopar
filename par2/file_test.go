package par2

import (
	"crypto/md5"
	"encoding/binary"
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/require"
)

type testDecoderDelegate struct {
	t *testing.T
}

func (d testDecoderDelegate) OnCreatorPacketLoad(clientID string) {
	d.t.Logf("OnCreatorPacketLoad(%s)", clientID)
}

func (d testDecoderDelegate) OnMainPacketLoad(sliceByteCount, recoverySetCount, nonRecoverySetCount int) {
	d.t.Logf("OnMainPacketLoad(sliceByteCount=%d, recoverySetCount=%d, nonRecoverySetCount=%d)", sliceByteCount, recoverySetCount, nonRecoverySetCount)
}

func (d testDecoderDelegate) OnFileDescriptionPacketLoad(fileID [16]byte, filename string, byteCount int) {
	d.t.Logf("OnFileDescriptionPacketLoad(%x, %s, %d)", fileID, filename, byteCount)
}

func (d testDecoderDelegate) OnIFSCPacketLoad(fileID [16]byte) {
	d.t.Logf("OnIFSCPacketLoad(%x)", fileID)
}

func (d testDecoderDelegate) OnRecoveryPacketLoad(exponent uint16, byteCount int) {
	d.t.Logf("OnRecoveryPacketLoad(%d, %d)", exponent, byteCount)
}

func (d testDecoderDelegate) OnUnknownPacketLoad(packetType [16]byte, byteCount int) {
	d.t.Logf("OnUnknownPacketLoad(%x, %d)", packetType, byteCount)
}

func (d testDecoderDelegate) OnOtherPacketSkip(setID [16]byte, packetType [16]byte, byteCount int) {
	d.t.Logf("OnOtherPacketSkip(%x, %x, %d)", setID, packetType, byteCount)
}

func (d testDecoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, hashMismatch, corrupt, hasWrongByteCount bool, err error) {
	d.t.Logf("OnDataFileLoad(%d, %d, %s, %d, hashMismatch=%t, corrupt=%t, hasWrongByteCount=%t, %v)", i, n, path, byteCount, hashMismatch, corrupt, hasWrongByteCount, err)
}

func (d testDecoderDelegate) OnParityFileLoad(i int, path string, err error) {
	d.t.Logf("OnParityFileLoad(%d, %s, %v)", i, path, err)
}

func (d testDecoderDelegate) OnDetectCorruptDataChunk(fileID [16]byte, filename string, startOffset, endOffset int) {
	d.t.Logf("OnDetectCorruptDataChunk(%x, %s, startOffset=%d, endOffset=%d)", fileID, filename, startOffset, endOffset)
}

func (d testDecoderDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {
	d.t.Logf("OnDataFileWrite(%d, %d, %s, %d, %v)", i, n, path, byteCount, err)
}

func makeTestFileInfo(sliceByteCount int, filename string, data []byte) (fileID, fileDescriptionPacket, ifscPacket, [][]uint16) {
	hash := md5.Sum(data)
	sixteenKHash := sixteenKHash(data)
	fileID := computeFileID(sixteenKHash, uint64(len(data)), []byte(filename))
	fileDescriptionPacket := fileDescriptionPacket{
		hash:         hash,
		sixteenKHash: sixteenKHash,
		byteCount:    len(data),
		filename:     filename,
	}
	var dataShards [][]uint16
	var checksumPairs []checksumPair
	for i := 0; i < len(data); i += sliceByteCount {
		slice := sliceAndPadByteArray(data, i, i+sliceByteCount)
		dataShards = append(dataShards, byteToUint16LEArray(slice))
		crc32 := crc32.ChecksumIEEE(slice)
		var crc32Bytes [4]byte
		binary.LittleEndian.PutUint32(crc32Bytes[:], crc32)
		checksumPairs = append(checksumPairs, checksumPair{
			MD5:   md5.Sum(slice),
			CRC32: crc32Bytes,
		})
	}
	return fileID, fileDescriptionPacket, ifscPacket{checksumPairs}, dataShards
}

func TestFileRoundTrip(t *testing.T) {
	sliceByteCount := 8
	fileID1, fileDescriptionPacket1, ifscPacket1, _ := makeTestFileInfo(sliceByteCount, "file1.txt", []byte("contents 1"))
	fileID2, fileDescriptionPacket2, ifscPacket2, _ := makeTestFileInfo(sliceByteCount, "file2.txt", []byte("contents 2"))
	fileID3, fileDescriptionPacket3, ifscPacket3, _ := makeTestFileInfo(sliceByteCount, "file3.txt", []byte("contents 3"))

	mainPacket := mainPacket{
		sliceByteCount: sliceByteCount,
		recoverySet:    []fileID{fileID2, fileID1},
		nonRecoverySet: []fileID{fileID3},
	}

	mainPacketBytes, err := writeMainPacket(mainPacket)
	require.NoError(t, err)
	expectedSetID := recoverySetID(md5.Sum(padPacketBytes(mainPacketBytes)))

	file := file{
		clientID:   "test client",
		mainPacket: &mainPacket,
		fileDescriptionPackets: map[fileID]fileDescriptionPacket{
			fileID1: fileDescriptionPacket1,
			fileID2: fileDescriptionPacket2,
			fileID3: fileDescriptionPacket3,
		},
		ifscPackets: map[fileID]ifscPacket{
			fileID1: ifscPacket1,
			fileID2: ifscPacket2,
			fileID3: ifscPacket3,
		},
		recoveryPackets: map[exponent]recoveryPacket{
			// TODO: Change this to match sliceByteCount.
			0: {data: []uint16{0x1, 0x2, 0x3, 0x4}},
			1: {data: []uint16{0x5, 0x6, 0x7, 0x8}},
		},
		unknownPackets: map[packetType][][]byte{
			packetType{0x1}: [][]byte{
				{0x1, 0x2, 0x3, 0x4},
				{0x5, 0x6, 0x7, 0x8},
			},
			packetType{0x2}: [][]byte{
				{0xa1, 0xa2, 0xa3, 0xa4},
				{0xa5, 0xa6, 0xa7, 0xa8},
			},
		},
	}

	setID, fileBytes, err := writeFile(file)
	require.NoError(t, err)
	require.Equal(t, expectedSetID, setID)

	roundTripSetID, roundTripFile, err := readFile(testDecoderDelegate{t}, &setID, fileBytes)
	require.NoError(t, err)
	require.Equal(t, setID, roundTripSetID)
	require.Equal(t, file, roundTripFile)
}
