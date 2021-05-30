package hashutil

import (
	"crypto/md5"
	"fmt"
)

// MD5Hash16k returns the MD5 hash of the first 16k bytes of the
// input, or the MD5 hash of the full input if it has fewer than 16k
// bytes.
func MD5Hash16k(data []byte) [md5.Size]byte {
	if len(data) < 16*1024 {
		return md5.Sum(data)
	}
	return md5.Sum(data[:16*1024])
}

// MD5HashWith16k returns the MD5 hash of the input, as well as the
// MD5 hash of the first 16k bytes of the input, or the MD5 hash of
// the full input again if it has fewer than 16k bytes.
func MD5HashWith16k(data []byte) (hash [md5.Size]byte, hash16k [md5.Size]byte) {
	hash = md5.Sum(data)
	if len(data) < 16*1024 {
		return hash, hash
	}
	return hash, md5.Sum(data[:16*1024])
}

// CheckMD5Hashes calculates the MD5 hashes of the given data and
// compares them to the given expected ones. If the 16k hash
// comparison fails, then the full hash isn't done.
func CheckMD5Hashes(data []byte, expectedHash16k, expectedHash [md5.Size]byte) error {
	if hash16k := MD5Hash16k(data); hash16k != expectedHash16k {
		return fmt.Errorf("hash mismatch (16k): expected=%x, actual=%x", expectedHash16k, hash16k)
	} else if hash := md5.Sum(data); hash != expectedHash {
		return fmt.Errorf("hash mismatch: expected=%x, actual=%x", expectedHash, hash)
	}
	return nil
}
