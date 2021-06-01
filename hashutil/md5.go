package hashutil

import (
	"crypto/md5"
	"fmt"
	"hash"
)

// MD5HasherWith16k is a Writer that keeps track of the MD5 hash of
// the first 16k bytes of the input.
type MD5HasherWith16k struct {
	md5Hash hash.Hash
	written int
	hash16k [md5.Size]byte
}

// MakeMD5HasherWith16k returns a new MD5HasherWith16k.
func MakeMD5HasherWith16k() *MD5HasherWith16k {
	return &MD5HasherWith16k{md5.New(), 0, [md5.Size]byte{}}
}

func (h *MD5HasherWith16k) write(p []byte) int {
	// Here we take advantage that hash.Hash is guaranteed to
	// never return an error.
	n, _ := h.md5Hash.Write(p)
	h.written += n
	return n
}

// Write writes the given data to the underlying MD5 hash object. Like
// hash.Hash.Write, this never returns an error.
func (h *MD5HasherWith16k) Write(p []byte) (n int, err error) {
	if h.written < 16*1024 && h.written+len(p) >= 16*1024 {
		midpoint := 16*1024 - h.written
		n1 := h.write(p[:midpoint])
		h.md5Hash.Sum(h.hash16k[:0])
		n2 := h.write(p[midpoint:])
		return n1 + n2, nil
	}
	return h.write(p), nil
}

// Hashes returns the MD5 hash of all the written data, as well as the
// MD5 hash of the first 16k bytes of the written data, or the MD5
// hash of the full written data again if fewer than 16k bytes were
// written.
func (h *MD5HasherWith16k) Hashes() (hash [md5.Size]byte, hash16k [md5.Size]byte) {
	h.md5Hash.Sum(hash[:0])
	if h.written < 16*1024 {
		hash16k = hash
	} else {
		hash16k = h.hash16k
	}
	return hash, hash16k
}

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
	h := MakeMD5HasherWith16k()
	// Assign to _ to silence errcheck.
	_, _ = h.Write(data)
	return h.Hashes()
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
