package hashutil

import (
	"crypto/md5"
	"fmt"
	"hash"
)

// md5Hash16k returns the MD5 hash of the first 16k bytes of the
// input, or the MD5 hash of the full input if it has fewer than 16k
// bytes, along with the Hash used to compute it (for further
// computations).
func md5Hash16k(data []byte) (hash16k [md5.Size]byte, h hash.Hash) {
	h = md5.New()
	if len(data) < 16*1024 {
		h.Write(data)
	} else {
		h.Write(data[:16*1024])
	}
	h.Sum(hash16k[:0])
	return hash16k, h
}

// MD5HashWith16k returns the MD5 hash of the input, as well as the
// MD5 hash of the first 16k bytes of the input, or the MD5 hash of
// the full input again if it has fewer than 16k bytes.
func MD5HashWith16k(data []byte) (hash [md5.Size]byte, hash16k [md5.Size]byte) {
	hash16k, h := md5Hash16k(data)
	if len(data) < 16*1024 {
		return hash16k, hash16k
	}
	h.Write(data[16*1024:])
	h.Sum(hash[:0])
	return hash, hash16k
}

// CheckMD5Hashes calculates the MD5 hashes of the given data and
// compares them to the given expected ones. If the 16k hash
// comparison fails, then the full hash isn't done.
func CheckMD5Hashes(data []byte, expectedHash16k, expectedHash [md5.Size]byte, isReconstructedData bool) error {
	suffix := ""
	if isReconstructedData {
		suffix = " in reconstructed data"
	}
	hash16k, h := md5Hash16k(data)
	if hash16k != expectedHash16k {
		return fmt.Errorf("hash mismatch (16k)%s: expected=%x, actual=%x", suffix, expectedHash16k, hash16k)
	} else if len(data) < 16*1024 {
		return nil
	}

	h.Write(data[16*1024:])
	var hash [md5.Size]byte
	h.Sum(hash[:0])
	if hash != expectedHash {
		return fmt.Errorf("hash mismatch%s: expected=%x, actual=%x", suffix, expectedHash, hash)
	}
	return nil
}
