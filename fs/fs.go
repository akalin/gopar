package fs

import "io"

// ReadStream defines the interface for streaming file reads. This is
// usually implemented by *os.File, but there might be other
// implementations for testing.
type ReadStream interface {
	io.Reader
	io.Closer
	// TODO: Once streaming is used everywhere, evaluate whether
	// we still need this function.
	ByteCount() int64
}

// ByteCountHolder is a helper type that can be embedded that
// implements the ByteCount() part of ReadStream.
type ByteCountHolder struct {
	Count int64
}

// ByteCount returns the underlying byte count.
func (h ByteCountHolder) ByteCount() int64 {
	return h.Count
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
	// Implementations must guarantee that exactly one of the
	// returned ReadStream and error is non-nil.
	GetReadStream(path string) (ReadStream, error)
	// FindWithPrefixAndSuffix should behave like calling
	// filepath.Glob with prefix + "*" + suffix.
	FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error)
	// WriteFile should behave like ioutil.WriteFile.
	WriteFile(path string, data []byte) error
}
