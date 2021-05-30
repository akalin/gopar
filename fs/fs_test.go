package fs

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

func TestReadStrict(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2})
	buf := []byte{}

	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = readStrict(r, buf)
	require.EqualError(t, err, "len(buf) == 0 unexpectedly in readStrict")
	require.Equal(t, 0, n)
}

func TestReadFullEOFZeroBuf(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := []byte{}

	n, err := readFullEOF(r, buf)
	require.EqualError(t, err, "len(buf) == 0 unexpectedly in readFullEOF")
	require.Equal(t, 0, n)
}

func TestReadFullEOFUnexpectedEOF(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 4)

	n, err := readFullEOF(r, buf)
	require.Equal(t, io.ErrUnexpectedEOF, err)
	require.Equal(t, 3, n)
}

// TODO: Remove this and use iotest.ErrReader instead once we stop
// supporting go 1.15 and earlier.
type errReader struct {
	err error
}

func (r errReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func TestReadFullEOFUnexpectedEOFWithError(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 4)

	expectedErr := errors.New("test error")
	n, err := readFullEOF(io.MultiReader(r, errReader{expectedErr}), buf)
	require.Equal(t, expectedErr, err)
	require.Equal(t, 3, n)
}

func TestReadFullEOFImmediateEOF(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 3)

	// Normal behavior of r is to return 0, io.EOF from the first
	// Read call after the last piece of data is read.
	n, err := readFullEOF(iotest.DataErrReader(r), buf)
	require.NoError(t, err)
	require.Equal(t, 3, n)
}

func TestReadFullEOFImmediateEOFWithError(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 3)

	expectedErr := errors.New("test error")
	n, err := readFullEOF(io.MultiReader(iotest.DataErrReader(r), errReader{expectedErr}), buf)
	require.Equal(t, expectedErr, err)
	require.Equal(t, 3, n)
}

func TestReadFullEOFDelayedEOF(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 3)

	// Normal behavior of r is to return 0, io.EOF from the first
	// Read call after the last piece of data is read.
	n, err := readFullEOF(r, buf)
	require.NoError(t, err)
	require.Equal(t, 3, n)
}

func TestReadFullEOFDelayedEOFWithError(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 3)

	expectedErr := errors.New("test error")
	n, err := readFullEOF(io.MultiReader(r, errReader{expectedErr}), buf)
	require.Equal(t, expectedErr, err)
	require.Equal(t, 3, n)
}
