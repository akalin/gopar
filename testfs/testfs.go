package testfs

import (
	"testing"

	"github.com/akalin/gopar/fs"
)

// TestFS is an implementation of fs.FS that wraps an existing
// implementation and logs it.
type TestFS struct {
	T  *testing.T
	FS fs.FS
}

// ReadFile implements the fs.FS interface.
func (io TestFS) ReadFile(path string) (data []byte, err error) {
	io.T.Helper()
	defer func() {
		io.T.Helper()
		io.T.Logf("ReadFile(%q) => (%d bytes, %v)", path, len(data), err)
	}()
	return io.FS.ReadFile(path)
}

// FindWithPrefixAndSuffix implements the fs.FS interface.
func (io TestFS) FindWithPrefixAndSuffix(prefix, suffix string) (matches []string, err error) {
	io.T.Helper()
	defer func() {
		io.T.Helper()
		io.T.Logf("FindWithPrefixAndSuffix(%q, %q) => (%d files, %v)", prefix, suffix, len(matches), err)
	}()
	return io.FS.FindWithPrefixAndSuffix(prefix, suffix)
}

// WriteFile implements the fs.FS interface.
func (io TestFS) WriteFile(path string, data []byte) (err error) {
	io.T.Helper()
	defer func() {
		io.T.Helper()
		io.T.Logf("WriteFile(%q, %d bytes) => %v", path, len(data), err)
	}()
	return io.FS.WriteFile(path, data)
}
