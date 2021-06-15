package fs

import "io"

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
}
