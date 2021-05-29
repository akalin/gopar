package testfs

import (
	"fmt"
	"testing"

	"github.com/akalin/gopar/fs"
)

type testReadStream struct {
	t    *testing.T
	path string
	rs   fs.ReadStream
}

func (trs testReadStream) Read(p []byte) (n int, err error) {
	trs.t.Helper()
	defer func() {
		trs.t.Helper()
		trs.t.Logf("Read(%q, %d bytes) => (%d bytes, %v)", trs.path, len(p), n, err)
	}()
	return trs.rs.Read(p)
}

func (trs testReadStream) Close() (err error) {
	trs.t.Helper()
	defer func() {
		trs.t.Helper()
		trs.t.Logf("Close(%q) => (%v)", trs.path, err)
	}()
	return trs.rs.Close()
}

func (trs testReadStream) ByteCount() (byteCount int64) {
	trs.t.Helper()
	defer func() {
		trs.t.Helper()
		trs.t.Logf("ByteCount(%q) => (%d)", trs.path, byteCount)
	}()
	return trs.rs.ByteCount()
}

func (trs testReadStream) String() string {
	return fmt.Sprintf("testReadStream{%q}", trs.path)
}

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

func (fs testFS) GetReadStream(path string) (readStream fs.ReadStream, err error) {
	fs.t.Helper()
	defer func() {
		fs.t.Helper()
		fs.t.Logf("GetReadStream(%q) => (%v, %v)", path, readStream, err)
	}()

	readStream, err = fs.fs.GetReadStream(path)
	if readStream != nil {
		readStream = testReadStream{fs.t, path, readStream}
	}
	return readStream, err
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
