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

type readFullFunction func(io.Reader, []byte) (int, error)

func runWithReadFullFunctions(t *testing.T, testFn func(*testing.T, readFullFunction)) {
	readFullFns := []struct {
		name string
		fn   readFullFunction
	}{
		{"ReadFull", ReadFull},
		{"ReadFullEOF", ReadFullEOF},
	}
	for _, pair := range readFullFns {
		pair := pair
		t.Run(pair.name, func(t *testing.T) {
			testFn(t, pair.fn)
		})
	}
}

func testReadFullZeroBuf(t *testing.T, fn readFullFunction) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := []byte{}

	n, err := fn(r, buf)
	require.EqualError(t, err, "len(buf) == 0 unexpectedly in ReadFull")
	require.Equal(t, 0, n)
}

func TestReadFullZeroBuf(t *testing.T) {
	runWithReadFullFunctions(t, testReadFullZeroBuf)
}

func testReadFullUnexpectedEOF(t *testing.T, fn readFullFunction) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 4)

	n, err := fn(r, buf)
	require.Equal(t, io.ErrUnexpectedEOF, err)
	require.Equal(t, 3, n)
}

func TestReadFullUnexpectedEOF(t *testing.T) {
	runWithReadFullFunctions(t, testReadFullUnexpectedEOF)
}

// TODO: Remove this and use iotest.ErrReader instead once we stop
// supporting go 1.15 and earlier.
type errReader struct {
	err error
}

func (r errReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func testReadFullUnexpectedEOFWithError(t *testing.T, fn readFullFunction) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 4)

	expectedErr := errors.New("test error")
	n, err := fn(io.MultiReader(r, errReader{expectedErr}), buf)
	require.Equal(t, expectedErr, err)
	require.Equal(t, 3, n)
}

func TestReadFullUnexpectedEOFWithError(t *testing.T) {
	runWithReadFullFunctions(t, testReadFullUnexpectedEOFWithError)
}

func testReadFullImmediateEOF(t *testing.T, fn readFullFunction) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 3)

	// Normal behavior of r is to return 0, io.EOF from the first
	// Read call after the last piece of data is read.
	n, err := fn(iotest.DataErrReader(r), buf)
	require.NoError(t, err)
	require.Equal(t, 3, n)
}

func TestReadFullImmediateEOF(t *testing.T) {
	runWithReadFullFunctions(t, testReadFullImmediateEOF)
}

func TestReadFullImmediateEOFWithError(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 3)

	expectedErr := errors.New("test error")
	n, err := ReadFull(io.MultiReader(iotest.DataErrReader(r), errReader{expectedErr}), buf)
	// Shouldn't reach the errReader.
	require.NoError(t, err)
	require.Equal(t, 3, n)
}

func TestReadFullEOFImmediateEOFWithError(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 3)

	expectedErr := errors.New("test error")
	n, err := ReadFullEOF(io.MultiReader(iotest.DataErrReader(r), errReader{expectedErr}), buf)
	require.Equal(t, expectedErr, err)
	require.Equal(t, 3, n)
}

func testReadFullDelayedEOF(t *testing.T, fn readFullFunction) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 3)

	// Normal behavior of r is to return 0, io.EOF from the first
	// Read call after the last piece of data is read.
	n, err := fn(r, buf)
	require.NoError(t, err)
	require.Equal(t, 3, n)
}

func TestReadFullDelayedEOF(t *testing.T) {
	runWithReadFullFunctions(t, testReadFullDelayedEOF)
}

func TestReadFullDelayedEOFWithError(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 3)

	expectedErr := errors.New("test error")
	n, err := ReadFull(io.MultiReader(r, errReader{expectedErr}), buf)
	// Shouldn't reach the errReader.
	require.NoError(t, err)
	require.Equal(t, 3, n)
}

func TestReadFullEOFDelayedEOFWithError(t *testing.T) {
	r := bytes.NewReader([]byte{0x1, 0x2, 0x3})
	buf := make([]byte, 3)

	expectedErr := errors.New("test error")
	n, err := ReadFullEOF(io.MultiReader(r, errReader{expectedErr}), buf)
	require.Equal(t, expectedErr, err)
	require.Equal(t, 3, n)
}
