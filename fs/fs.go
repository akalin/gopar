package fs

import (
	"errors"
	"io"
)

// ReadStream defines the interface for streaming file reads. This is
// usually implemented by *os.File, but there might be other
// implementations for testing.
type ReadStream interface {
	// This object must not be used once it is
	// closed. Implementations should return an error in that
	// case, or panic if that isn't possible.
	io.ReadCloser
	// TODO: Once streaming is used everywhere, evaluate whether
	// we still need this function.
	ByteCount() int64
}

type readCloserStream struct {
	io.ReadCloser
	byteCount int64
}

func (rcs readCloserStream) ByteCount() int64 {
	return rcs.byteCount
}

// ReadCloserToStream makes and returns a ReadStream out of the given
// ReadCloser and byte count.
func ReadCloserToStream(readCloser io.ReadCloser, byteCount int64) ReadStream {
	return readCloserStream{readCloser, byteCount}
}

// WriteStream defines the interface for streaming file writes. This
// is usually implemented by *os.File, but there might be other
// implementations for testing.
type WriteStream interface {
	io.Writer
	io.Closer
}

// FS is the interface used by the par1 and par2 packages to the
// filesystem. Most code uses DefaultFS, but tests may use other
// implementations.
type FS interface {
	// ReadFile should behave like ioutil.ReadFile.
	ReadFile(path string) ([]byte, error)
	// GetReadStream returns a ReadStream to read the file at the
	// given path.
	//
	// Only one ReadStream may be open per (normalized)
	// path. Implementations should return an error in case
	// GetReadStream is called on a path with an open ReadStream.
	//
	// Implementations must guarantee that exactly one of the
	// returned ReadStream and error is non-nil.
	GetReadStream(path string) (ReadStream, error)
	// FindWithPrefixAndSuffix should behave like calling
	// filepath.Glob with prefix + "*" + suffix.
	FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error)
	// WriteFile should behave like ioutil.WriteFile.
	WriteFile(path string, data []byte) error
	// GetFileReadSeekCloser returns a WriteStream to write to the
	// file at the given path.
	//
	// Implementations must guarantee that exactly one of the
	// returned WriteStream and error is non-nil.
	GetWriteStream(path string) (WriteStream, error)
}

// readStrict checks that len(buf) != 0, calls r.Read(buf), and checks
// that the return value isn't 0, nil.
func readStrict(r io.Reader, buf []byte) (n int, err error) {
	if len(buf) == 0 {
		return 0, errors.New("len(buf) == 0 unexpectedly in readStrict")
	}
	n, err = r.Read(buf)
	if n == 0 && err == nil {
		return n, errors.New("r.Read() returned 0, nil")
	}
	return n, err
}

// readFullEOF is like io.ReadFull, except that it:
//
//   - requires len(buf) to be non-zero,
//   - calls readStrict instead,
//   - doesn't drop the error even if the buffer is completely filled,
//     except when the error is EOF,
//   - if the buffer is completely filled, checks that the next read from
//     the reader triggers an EOF.
func readFullEOF(r io.Reader, buf []byte) (n int, err error) {
	if len(buf) == 0 {
		return 0, errors.New("len(buf) == 0 unexpectedly in readFullEOF")
	}
	for n < len(buf) && err == nil {
		var nn int
		nn, err = readStrict(r, buf[n:])
		n += nn
	}
	if n < len(buf) {
		// Loop termination condition guarantees err != nil.
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return n, err
	}
	// Now we know that n >= len(buf) (really n == len(buf)), so
	// we just have to examine err.
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		return n, err
	}

	// Now we know that we filled up buf with no error (or EOF),
	// so now we just make sure that we're actually at EOF.
	var singleByte [1]byte
	_, err = readStrict(r, singleByte[:])
	if err == io.EOF {
		err = nil
	}
	return n, err
}

// ReadAndClose reads all the data in the given io.ReadCloser into a
// buffer and returns it, closing it in all cases.
//
// If err == nil, the returned buffer will never be nil, even if it
// has length 0.
//
// TODO: Make this function unnecessary.
func ReadAndClose(readStream ReadStream) (data []byte, err error) {
	defer func() {
		closeErr := readStream.Close()
		if err == nil {
			err = closeErr
		}
	}()
	byteCount := readStream.ByteCount()
	if int64(int(byteCount)) != byteCount {
		return nil, errors.New("file too big to read into memory")
	}
	data = make([]byte, byteCount)
	if len(data) > 0 {
		_, err = readFullEOF(readStream, data)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}
