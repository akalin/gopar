package hashutil

import (
	"bytes"
	"crypto/md5"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

var testInputs = [][]byte{
	{},
	bytes.Repeat([]byte{0x1}, 100),
	bytes.Repeat([]byte{0x2}, 16*1024-1),
	bytes.Repeat([]byte{0x3}, 16*1024),
	bytes.Repeat([]byte{0x4}, 16*1024+1),
}

func md5HashWith16k(t *testing.T, data []byte) (hash [md5.Size]byte, hash16k [md5.Size]byte) {
	md5Hasher := MakeMD5HasherWith16k()
	_, err := md5Hasher.Write(data)
	require.NoError(t, err)
	return md5Hasher.Hashes()
}

func TestMD5HasherWith16k(t *testing.T) {
	for _, input := range testInputs {
		hash, hash16k := md5HashWith16k(t, input)
		require.Equal(t, md5.Sum(input), hash)
		expectedHash16k, _ := md5Hash16k(input)
		foo := input
		if len(foo) > 5 {
			foo = foo[:5]
		}
		require.Equal(t, expectedHash16k, hash16k, "input is %x %d", foo, len(input))
	}
}

func TestMD5Hash16k(t *testing.T) {
	for _, input := range testInputs {
		hash16k, h := md5Hash16k(input)
		if len(input) < 16*1024 {
			require.Equal(t, md5.Sum(input), hash16k)
		} else {
			require.Equal(t, md5.Sum(input[:16*1024]), hash16k)
		}
		require.NotNil(t, h)
	}
}

func TestCheckReaderMD5Hashes(t *testing.T) {
	input := bytes.Repeat([]byte{0x5}, 17*1024)
	hash, hash16k := md5HashWith16k(t, input)
	require.NoError(t, checkReaderMD5Hashes(bytes.NewReader(input), hash16k, hash, false))
	require.NoError(t, checkReaderMD5Hashes(bytes.NewReader(input), hash16k, hash, true))
	require.Equal(t, HashMismatchError{hash, hash16k, true, false}, checkReaderMD5Hashes(bytes.NewReader(input), hash, hash, false))
	require.Equal(t, HashMismatchError{hash16k, hash, false, false}, checkReaderMD5Hashes(bytes.NewReader(input), hash16k, hash16k, false))
	require.Equal(t, HashMismatchError{hash, hash16k, true, true}, checkReaderMD5Hashes(bytes.NewReader(input), hash, hash, true))
	require.Equal(t, HashMismatchError{hash16k, hash, false, true}, checkReaderMD5Hashes(bytes.NewReader(input), hash16k, hash16k, true))
}

// TODO: Remove this once we stop supporting go 1.15 and earlier.
type errReader struct {
	err error
}

func (r errReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func TestCheckReaderMD5HashesReaderFailure(t *testing.T) {
	input := bytes.Repeat([]byte{0x5}, 17*1024)
	hash, hash16k := md5HashWith16k(t, input)
	err := errors.New("test error")
	for _, isReconstructedData := range []bool{true, false} {
		require.Equal(t, err, checkReaderMD5Hashes(errReader{err}, hash16k, hash, isReconstructedData))
	}

	for _, isReconstructedData := range []bool{true, false} {
		require.Equal(t, err, checkReaderMD5Hashes(io.MultiReader(bytes.NewReader(input), errReader{err}), hash16k, hash, isReconstructedData))
	}
}

func TestCheckMD5Hashes(t *testing.T) {
	input := bytes.Repeat([]byte{0x5}, 17*1024)
	hash, hash16k := md5HashWith16k(t, input)
	require.NoError(t, CheckMD5Hashes(input, hash16k, hash, false))
	require.NoError(t, CheckMD5Hashes(input, hash16k, hash, true))
	require.Equal(t, HashMismatchError{hash, hash16k, true, false}, CheckMD5Hashes(input, hash, hash, false))
	require.Equal(t, HashMismatchError{hash16k, hash, false, false}, CheckMD5Hashes(input, hash16k, hash16k, false))
	require.Equal(t, HashMismatchError{hash, hash16k, true, true}, CheckMD5Hashes(input, hash, hash, true))
	require.Equal(t, HashMismatchError{hash16k, hash, false, true}, CheckMD5Hashes(input, hash16k, hash16k, true))
}
