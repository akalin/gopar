package par2

import (
	"crypto/md5"
	"encoding/binary"
	"hash/crc32"

	"github.com/akalin/gopar/fs"
	"github.com/akalin/gopar/hashutil"
)

func computeDataFileInfo(sliceByteCount int, filename string, readStream fs.ReadStream) (fileID, fileDescriptionPacket, ifscPacket, [][]byte, int64, error) {
	h := hashutil.MakeMD5HasherWith16k()
	data, err := fs.ReadAndClose(hashutil.TeeReadStream(readStream, h))
	if err != nil {
		return fileID{}, fileDescriptionPacket{}, ifscPacket{}, nil, 0, err
	}

	size := len(data)
	hash, hash16k := h.Hashes()
	fileID := computeFileID(hash16k, uint64(size), []byte(filename))
	fileDescriptionPacket := fileDescriptionPacket{
		hash:      hash,
		hash16k:   hash16k,
		byteCount: int64(size),
		filename:  filename,
	}
	var dataShards [][]byte
	var checksumPairs []checksumPair
	for i := 0; i < size; i += sliceByteCount {
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
	return fileID, fileDescriptionPacket, ifscPacket{checksumPairs}, dataShards, int64(size), nil
}
