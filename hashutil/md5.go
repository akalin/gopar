package hashutil

import "crypto/md5"

// MD5Hash16k returns the MD5 hash of the first 16k bytes of the
// input, or the MD5 hash of the full input if it has fewer than 16k
// bytes.
func MD5Hash16k(data []byte) [md5.Size]byte {
	if len(data) < 16*1024 {
		return md5.Sum(data)
	}
	return md5.Sum(data[:16*1024])
}
