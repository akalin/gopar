package hashutil

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/akalin/gopar/fs"
	"github.com/akalin/gopar/memfs"
	"github.com/stretchr/testify/require"
)

type closeChecker struct {
	closed bool
	name   string
}

func (c closeChecker) errClosedTwice() error {
	return fmt.Errorf("closed twice (name=%s)", c.name)
}

func (c *closeChecker) Close() error {
	if c.closed {
		return c.errClosedTwice()
	}
	c.closed = true
	return nil
}

type testReadStream struct {
	fs.ReadStream
	closeChecker
}

func (trs *testReadStream) Close() error {
	return trs.closeChecker.Close()
}

func newTestReadStream(buf []byte) *testReadStream {
	return &testReadStream{memfs.MakeReadStream(buf), closeChecker{name: "testReadStream"}}
}

func TestTeeReadStream(t *testing.T) {
	src := []byte("hello, world")
	dst := make([]byte, len(src))
	trs := newTestReadStream(src)
	wb := new(bytes.Buffer)
	r := TeeReadStream(trs, wb)

	require.Equal(t, int64(len(src)), r.ByteCount())

	n, err := fs.ReadFullEOF(r, dst)
	require.NoError(t, err)
	require.Equal(t, len(src), n)
	require.Equal(t, src, dst)
	require.Equal(t, src, wb.Bytes())
	require.False(t, trs.closed)

	n, err = r.Read(dst)
	require.Equal(t, io.EOF, err)
	require.Equal(t, 0, n)
	require.False(t, trs.closed)

	require.NoError(t, r.Close())
	require.True(t, trs.closed)

	require.Equal(t, trs.errClosedTwice(), r.Close())
	require.True(t, trs.closed)
}

func TestTeeReadStreamWriterError(t *testing.T) {
	src := []byte("hello, world")
	dst := make([]byte, len(src))
	trs := newTestReadStream(src)
	pr, pw := io.Pipe()
	require.NoError(t, pr.Close())
	r := TeeReadStream(trs, pw)

	n, err := fs.ReadFullEOF(r, dst)
	require.Equal(t, io.ErrClosedPipe, err)
	require.Equal(t, 0, n)

	require.NoError(t, r.Close())
	require.True(t, trs.closed)
}

type testWriteCloser struct {
	io.Writer
	closeChecker
}

func newTestWriteCloser() testWriteCloser {
	return testWriteCloser{new(bytes.Buffer), closeChecker{name: "testWriteCloser"}}
}

func TestTeeReadStreamWriteCloser(t *testing.T) {
	src := []byte("hello, world")
	dst := make([]byte, len(src))
	trs := newTestReadStream(src)
	twc := newTestWriteCloser()
	r := TeeReadStream(trs, &twc)

	n, err := fs.ReadFullEOF(r, dst)
	require.NoError(t, err)
	require.Equal(t, len(src), n)
	require.Equal(t, src, dst)
	require.False(t, trs.closed)
	require.False(t, twc.closed)

	require.NoError(t, r.Close())
	require.True(t, trs.closed)
	require.True(t, twc.closed)

	require.Equal(t, trs.errClosedTwice(), r.Close())
	require.True(t, trs.closed)
	require.True(t, twc.closed)
}

func TestTeeReadStreamWriteCloserClose(t *testing.T) {
	src := []byte("hello, world")
	trs := newTestReadStream(src)
	twc := newTestWriteCloser()
	r := TeeReadStream(trs, &twc)

	trs.closed = true
	twc.closed = true
	require.Equal(t, trs.errClosedTwice(), r.Close())
	require.True(t, trs.closed)
	require.True(t, twc.closed)

	trs.closed = true
	twc.closed = false
	require.Equal(t, trs.errClosedTwice(), r.Close())
	require.True(t, trs.closed)
	require.True(t, twc.closed)

	trs.closed = false
	twc.closed = true
	require.Equal(t, twc.errClosedTwice(), r.Close())
	require.True(t, trs.closed)
	require.True(t, twc.closed)

	trs.closed = false
	twc.closed = false
	require.NoError(t, r.Close())
	require.True(t, trs.closed)
	require.True(t, twc.closed)
}
