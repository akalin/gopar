package hashutil

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"hash"
	"io"
)

type md5HashProcessor struct {
	md5Hash        hash.Hash
	written        int
	processHash16k func([md5.Size]byte) error
	processHash    func([md5.Size]byte) error
}

func (h *md5HashProcessor) write(p []byte) int {
	n, _ := h.md5Hash.Write(p)
	h.written += n
	return n
}

func (h md5HashProcessor) sum() [md5.Size]byte {
	var hash [md5.Size]byte
	h.md5Hash.Sum(hash[:0])
	return hash
}

// Write writes the given data to the underlying MD5 hash object.
// When 16k bytes have been written, it calls processHash16k with the
// hash of those 16k bytes, which may return an error.
func (h *md5HashProcessor) Write(p []byte) (n int, err error) {
	if h.written < 16*1024 && h.written+len(p) >= 16*1024 {
		midpoint := 16*1024 - h.written
		n1 := h.write(p[:midpoint])
		err = h.processHash16k(h.sum())
		if err != nil {
			return n1, err
		}
		n2 := h.write(p[midpoint:])
		return n1 + n2, nil
	}
	return h.write(p), nil
}

// Close calls processHash16k with the computed hash if it hasn't been
// called, which may return an error. If it doesn't, it also calls
// processHash with the computed hash which also may return an error.
func (h *md5HashProcessor) Close() error {
	hash := h.sum()
	if h.written < 16*1024 {
		err := h.processHash16k(hash)
		if err != nil {
			return err
		}
	}
	return h.processHash(hash)
}

// MD5HasherWith16k is a Writer that keeps track of the MD5 hash of
// the first 16k bytes of the input.
type MD5HasherWith16k struct {
	hp      md5HashProcessor
	hash16k *[md5.Size]byte
}

// MakeMD5HasherWith16k returns a new MD5HasherWith16k.
func MakeMD5HasherWith16k() *MD5HasherWith16k {
	var h MD5HasherWith16k
	h.hp = md5HashProcessor{
		md5Hash: md5.New(),
		processHash16k: func(hash16k [md5.Size]byte) error {
			h.hash16k = &hash16k
			return nil
		},
		processHash: func(hash [md5.Size]byte) error {
			return nil
		},
	}
	return &h
}

// Write writes the given data to the underlying MD5 hash object. Like
// hash.Hash.Write, this never returns an error.
func (h *MD5HasherWith16k) Write(p []byte) (n int, err error) {
	return h.hp.Write(p)
}

// Hashes returns the MD5 hash of all the written data, as well as the
// MD5 hash of the first 16k bytes of the written data, or the MD5
// hash of the full written data again if fewer than 16k bytes were
// written.
func (h MD5HasherWith16k) Hashes() (hash [md5.Size]byte, hash16k [md5.Size]byte) {
	hash = h.hp.sum()
	if h.hash16k != nil {
		hash16k = *h.hash16k
	} else {
		hash16k = hash
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

// A HashMismatchError is an error returned when a hash check fails in
// the io.WriteCloser given by MakeMD5HashCheckerWith16k.
type HashMismatchError struct {
	expectedHash        [md5.Size]byte
	hash                [md5.Size]byte
	is16k               bool
	isReconstructedData bool
}

func (e HashMismatchError) Error() string {
	suffix := ""
	if e.is16k {
		suffix += " (16k)"
	}
	if e.isReconstructedData {
		suffix += " in reconstructed data"
	}
	return fmt.Sprintf("hash mismatch%s: expected=%x, actual=%x", suffix, e.expectedHash, e.hash)
}

type md5HashCheckerWith16k struct {
	hp md5HashProcessor
}

// MakeMD5HashCheckerWith16k returns an io.WriteCloser that checks the
// written data against the given hashes. Hash mismatch errors are of
// type HashMismatchError.
func MakeMD5HashCheckerWith16k(expectedHash16k, expectedHash [md5.Size]byte, isReconstructedData bool) io.WriteCloser {
	return &md5HashCheckerWith16k{
		hp: md5HashProcessor{
			md5Hash: md5.New(),
			processHash16k: func(hash16k [md5.Size]byte) error {
				if hash16k != expectedHash16k {
					return HashMismatchError{expectedHash16k, hash16k, true, isReconstructedData}
				}
				return nil
			},
			processHash: func(hash [md5.Size]byte) error {
				if hash != expectedHash {
					return HashMismatchError{expectedHash, hash, false, isReconstructedData}
				}
				return nil
			},
		},
	}
}

// Write writes the given data to the underlying MD5 hash object.
// When 16k bytes have been written, it checks the hash of those 16k
// bytes against the expected ones, and returns an error if they don't
// match.
func (h *md5HashCheckerWith16k) Write(p []byte) (n int, err error) {
	return h.hp.Write(p)
}

// Close checks the computed hash against the expected 16k hash, if it
// hasn't been called already, which may return an error. If it
// doesn't, it also checks the computed hash against the expected full
// hash.
func (h *md5HashCheckerWith16k) Close() error {
	return h.hp.Close()
}

func checkReaderMD5Hashes(reader io.Reader, expectedHash16k, expectedHash [md5.Size]byte, isReconstructedData bool) error {
	checker := MakeMD5HashCheckerWith16k(expectedHash16k, expectedHash, isReconstructedData)
	_, err := io.Copy(checker, reader)
	if err != nil {
		return err
	}
	return checker.Close()
}

// CheckMD5Hashes calculates the MD5 hashes of the given data and
// compares them to the given expected ones. If the 16k hash
// comparison fails, then the full hash isn't done.
func CheckMD5Hashes(data []byte, expectedHash16k, expectedHash [md5.Size]byte, isReconstructedData bool) error {
	return checkReaderMD5Hashes(bytes.NewReader(data), expectedHash16k, expectedHash, isReconstructedData)
}
