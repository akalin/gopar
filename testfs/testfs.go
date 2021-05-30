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

type testWriteStream struct {
	t    *testing.T
	path string
	w    fs.WriteStream
}

func (tw testWriteStream) Write(p []byte) (n int, err error) {
	tw.t.Helper()
	defer func() {
		tw.t.Helper()
		tw.t.Logf("Write(%q, %d bytes) => (%d bytes, %v)", tw.path, len(p), n, err)
	}()
	return tw.w.Write(p)
}

func (tw testWriteStream) Close() (err error) {
	tw.t.Helper()
	defer func() {
		tw.t.Helper()
		tw.t.Logf("Close(%q) => (%v)", tw.path, err)
	}()
	return tw.w.Close()
}

func (tw testWriteStream) String() string {
	return fmt.Sprintf("testWriteStream{%q}", tw.path)
}

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

// GetReadStream implements the fs.FS interface.
func (io TestFS) GetReadStream(path string) (readStream fs.ReadStream, err error) {
	io.T.Helper()
	defer func() {
		io.T.Helper()
		io.T.Logf("GetFileReadSeekCloser(%q) => (%v, %v)", path, readStream, err)
	}()
	readStream, err = io.FS.GetReadStream(path)
	if readStream != nil {
		readStream = testReadStream{io.T, path, readStream}
	}
	return readStream, err
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

// GetWriteStream implements the fs.FS interface.
func (io TestFS) GetWriteStream(path string) (writeStream fs.WriteStream, err error) {
	io.T.Helper()
	defer func() {
		io.T.Helper()
		io.T.Logf("GetFileWriteStream(%q) => (%v, %v)", path, writeStream, err)
	}()
	writeStream, err = io.FS.GetWriteStream(path)
	if writeStream != nil {
		writeStream = testWriteStream{io.T, path, writeStream}
	}
	return writeStream, err
}
