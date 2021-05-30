package testfs

import (
	"testing"

	"github.com/akalin/gopar/fs"
)

type testFS struct {
	t  *testing.T
	fs fs.FS
}

// MakeTestFS returns a fs.FS implementation that wraps the existing
// implementation, logging everything to the given *testing.T.
func MakeTestFS(t *testing.T, fs fs.FS) fs.FS {
	return testFS{t, fs}
}

func (fs testFS) ReadFile(path string) (data []byte, err error) {
	fs.t.Helper()
	defer func() {
		fs.t.Helper()
		fs.t.Logf("ReadFile(%q) => (%d bytes, %v)", path, len(data), err)
	}()
	return fs.fs.ReadFile(path)
}

func (fs testFS) FindWithPrefixAndSuffix(prefix, suffix string) (matches []string, err error) {
	fs.t.Helper()
	defer func() {
		fs.t.Helper()
		fs.t.Logf("FindWithPrefixAndSuffix(%q, %q) => (%d files, %v)", prefix, suffix, len(matches), err)
	}()
	return fs.fs.FindWithPrefixAndSuffix(prefix, suffix)
}

func (fs testFS) WriteFile(path string, data []byte) (err error) {
	fs.t.Helper()
	defer func() {
		fs.t.Helper()
		fs.t.Logf("WriteFile(%q, %d bytes) => %v", path, len(data), err)
	}()
	return fs.fs.WriteFile(path, data)
}
