package testfs

import (
	"fmt"
	"testing"

	"github.com/akalin/gopar/fs"
	"github.com/stretchr/testify/require"
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

func (trs testReadStream) ReadAt(p []byte, off int64) (n int, err error) {
	trs.t.Helper()
	defer func() {
		trs.t.Helper()
		trs.t.Logf("ReadAt(%q, %d bytes, %d) => (%d bytes, %v)", trs.path, len(p), off, n, err)
	}()
	return trs.rs.ReadAt(p, off)
}

func (trs testReadStream) Close() (err error) {
	trs.t.Helper()
	defer func() {
		trs.t.Helper()
		trs.t.Logf("Close(%q) => (%v)", trs.path, err)
	}()
	return trs.rs.Close()
}

func (trs testReadStream) Offset() (offset int64) {
	trs.t.Helper()
	defer func() {
		trs.t.Helper()
		trs.t.Logf("Offset(%q) => (%d)", trs.path, offset)
	}()
	return trs.rs.Offset()
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

type testWriteStream struct {
	t    *testing.T
	path string
	w    fs.WriteStream
}

func (tws testWriteStream) Write(p []byte) (n int, err error) {
	tws.t.Helper()
	defer func() {
		tws.t.Helper()
		tws.t.Logf("Write(%q, %d bytes) => (%d bytes, %v)", tws.path, len(p), n, err)
	}()
	return tws.w.Write(p)
}

func (tws testWriteStream) Close() (err error) {
	tws.t.Helper()
	defer func() {
		tws.t.Helper()
		tws.t.Logf("Close(%q) => (%v)", tws.path, err)
	}()
	return tws.w.Close()
}

func (tws testWriteStream) String() string {
	return fmt.Sprintf("testWriteStream{%q}", tws.path)
}

type testFS struct {
	t   *testing.T
	fs  fs.FS
	ofm fs.OpenFileManager
}

// MakeTestFS returns a fs.FS implementation that wraps the existing
// implementation, logging everything to the given *testing.T.
func MakeTestFS(t *testing.T, delegateFS fs.FS) fs.FS {
	testFS := testFS{t, delegateFS, fs.MakeOpenFileManager()}
	t.Cleanup(testFS.requireNoOpenFiles)
	return testFS
}

func (fs testFS) getReadStream(path string) (readStream fs.ReadStream, err error) {
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

func (fs testFS) GetReadStream(path string) (readStream fs.ReadStream, err error) {
	fs.t.Helper()
	return fs.ofm.GetReadStream(path, fs.getReadStream)
}

func (fs testFS) FindWithPrefixAndSuffix(prefix, suffix string) (matches []string, err error) {
	fs.t.Helper()
	defer func() {
		fs.t.Helper()
		fs.t.Logf("FindWithPrefixAndSuffix(%q, %q) => (%d files, %v)", prefix, suffix, len(matches), err)
	}()
	return fs.fs.FindWithPrefixAndSuffix(prefix, suffix)
}

func (fs testFS) getWriteStream(path string) (writeStream fs.WriteStream, err error) {
	fs.t.Helper()
	defer func() {
		fs.t.Helper()
		fs.t.Logf("GetWriteStream(%q) => (%v, %v)", path, writeStream, err)
	}()
	writeStream, err = fs.fs.GetWriteStream(path)
	if writeStream != nil {
		writeStream = testWriteStream{fs.t, path, writeStream}
	}
	return writeStream, err
}

func (fs testFS) GetWriteStream(path string) (writeStream fs.WriteStream, err error) {
	fs.t.Helper()
	return fs.ofm.GetWriteStream(path, fs.getWriteStream)
}

func (fs testFS) requireNoOpenFiles() {
	for _, path := range fs.ofm.GetOpenFilePaths() {
		require.Failf(fs.t, "open file detected", "%q is still open", path)
	}
}
