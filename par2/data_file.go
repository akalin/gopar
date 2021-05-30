package par2

import (
	"crypto/md5"
	"encoding/binary"
	"hash/crc32"

	"github.com/akalin/gopar/hashutil"
)

func computeDataFileInfo(sliceByteCount int, filename string, data []byte) (fileID, fileDescriptionPacket, ifscPacket, [][]byte) {
	hash := md5.Sum(data)
	hash16k := hashutil.MD5Hash16k(data)
	fileID := computeFileID(hash16k, uint64(len(data)), []byte(filename))
	fileDescriptionPacket := fileDescriptionPacket{
		hash:      hash,
		hash16k:   hash16k,
		byteCount: len(data),
		filename:  filename,
	}
	var dataShards [][]byte
	var checksumPairs []checksumPair
	for i := 0; i < len(data); i += sliceByteCount {
		slice := sliceAndPadByteArray(data, i, i+sliceByteCount)
		dataShards = append(dataShards, slice)
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
