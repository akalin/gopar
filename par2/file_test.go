package par2

import (
	"crypto/md5"
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

func makeTestFileInfo(sliceByteCount int, hash [md5.Size]byte, byteCount int, filename string) (fileID, fileDescriptionPacket, ifscPacket) {
	fileID := computeFileID(hash, uint64(byteCount), []byte(filename))
	fileDescriptionPacket := fileDescriptionPacket{
		hash:         hash,
		sixteenKHash: hash,
		byteCount:    byteCount,
		filename:     filename,
	}
	var checksumPairs []checksumPair
	for i := 0; i < byteCount; i += sliceByteCount {
		checksumPairs = append(checksumPairs, checksumPair{
			MD5:   [md5.Size]byte{byte(i)},
			CRC32: [4]byte{byte(i)},
		})
	}
	return fileID, fileDescriptionPacket, ifscPacket{checksumPairs}
}

func TestFileRoundTrip(t *testing.T) {
	sliceByteCount := 8
	fileID1, fileDescriptionPacket1, ifscPacket1 := makeTestFileInfo(sliceByteCount, [md5.Size]byte{0x1}, 11, "file1.txt")
	fileID2, fileDescriptionPacket2, ifscPacket2 := makeTestFileInfo(sliceByteCount, [md5.Size]byte{0x2}, 5, "file2.txt")
	fileID3, fileDescriptionPacket3, ifscPacket3 := makeTestFileInfo(sliceByteCount, [md5.Size]byte{0x3}, 5, "file3.txt")

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
