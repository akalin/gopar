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
	d.t.Helper()
	d.t.Logf("OnCreatorPacketLoad(%s)", clientID)
}

func (d testDecoderDelegate) OnMainPacketLoad(sliceByteCount, recoverySetCount, nonRecoverySetCount int) {
	d.t.Helper()
	d.t.Logf("OnMainPacketLoad(sliceByteCount=%d, recoverySetCount=%d, nonRecoverySetCount=%d)", sliceByteCount, recoverySetCount, nonRecoverySetCount)
}

func (d testDecoderDelegate) OnFileDescriptionPacketLoad(fileID [16]byte, filename string, byteCount int) {
	d.t.Helper()
	d.t.Logf("OnFileDescriptionPacketLoad(%x, %s, %d)", fileID, filename, byteCount)
}

func (d testDecoderDelegate) OnIFSCPacketLoad(fileID [16]byte) {
	d.t.Helper()
	d.t.Logf("OnIFSCPacketLoad(%x)", fileID)
}

func (d testDecoderDelegate) OnRecoveryPacketLoad(exponent uint16, byteCount int) {
	d.t.Helper()
	d.t.Logf("OnRecoveryPacketLoad(%d, %d)", exponent, byteCount)
}

func (d testDecoderDelegate) OnUnknownPacketLoad(packetType [16]byte, byteCount int) {
	d.t.Helper()
	d.t.Logf("OnUnknownPacketLoad(%x, %d)", packetType, byteCount)
}

func (d testDecoderDelegate) OnOtherPacketSkip(setID [16]byte, packetType [16]byte, byteCount int) {
	d.t.Helper()
	d.t.Logf("OnOtherPacketSkip(%x, %x, %d)", setID, packetType, byteCount)
}

func (d testDecoderDelegate) OnDataFileLoad(i, n int, path string, byteCount, hits, misses int, err error) {
	d.t.Helper()
	d.t.Logf("OnDataFileLoad(%d, %d, %s, byteCount=%d, hits=%d, misses=%d, %v)", i, n, path, byteCount, hits, misses, err)
}

func (d testDecoderDelegate) OnParityFileLoad(i int, path string, err error) {
	d.t.Helper()
	d.t.Logf("OnParityFileLoad(%d, %s, %v)", i, path, err)
}

func (d testDecoderDelegate) OnDetectCorruptDataChunk(fileID [16]byte, path string, startOffset, endOffset int) {
	d.t.Helper()
	d.t.Logf("OnDetectCorruptDataChunk(%x, %s, startOffset=%d, endOffset=%d)", fileID, path, startOffset, endOffset)
}

func (d testDecoderDelegate) OnDetectDataFileHashMismatch(fileID [16]byte, path string, hashInfo HashInfo) {
	d.t.Helper()
	d.t.Logf("OnDetectDataFileHashMismatch(%x, %s, %x)", fileID, path, hashInfo)
}

func (d testDecoderDelegate) OnDetectDataFileWrongByteCount(fileID [16]byte, path string) {
	d.t.Helper()
	d.t.Logf("OnDetectDataFileWrongByteCount(%x, %s)", fileID, path)
}

func (d testDecoderDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {
	d.t.Helper()
	d.t.Logf("OnDataFileWrite(%d, %d, %s, %d, %v)", i, n, path, byteCount, err)
}

func TestFileRoundTrip(t *testing.T) {
	sliceByteCount := 8
	fileID1, fileDescriptionPacket1, ifscPacket1, _ := computeDataFileInfo(sliceByteCount, "file1.txt", []byte("contents 1"))
	fileID2, fileDescriptionPacket2, ifscPacket2, _ := computeDataFileInfo(sliceByteCount, "file2.txt", []byte("contents 2"))
	fileID3, fileDescriptionPacket3, ifscPacket3, _ := computeDataFileInfo(sliceByteCount, "file3.txt", []byte("contents 3"))

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
			0: {data: []byte{0xff, 0xaa, 0xfe, 0xab, 0xfd, 0xac, 0xfc, 0xad}},
			1: {data: []byte{0xef, 0xba, 0xee, 0xbb, 0xed, 0xbc, 0xec, 0xbd}},
		},
		unknownPackets: map[packetType][][]byte{
			{0x1}: {
				{0x1, 0x2, 0x3, 0x4},
				{0x5, 0x6, 0x7, 0x8},
			},
			{0x2}: {
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
