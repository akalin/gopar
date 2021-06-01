package hashutil

import (
	"bytes"
	"crypto/md5"
	"fmt"
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
		require.Equal(t, expectedHash16k, hash16k)
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

func TestMD5HashWith16k(t *testing.T) {
	for _, input := range testInputs {
		hash, hash16k := MD5HashWith16k(input)
		require.Equal(t, md5.Sum(input), hash)
		expectedHash16k, _ := md5Hash16k(input)
		require.Equal(t, expectedHash16k, hash16k)
	}
}

func TestCheckMD5Hashes(t *testing.T) {
	input := bytes.Repeat([]byte{0x5}, 17*1024)
	hash, hash16k := md5HashWith16k(t, input)
	require.NoError(t, CheckMD5Hashes(input, hash16k, hash, false))
	require.NoError(t, CheckMD5Hashes(input, hash16k, hash, true))
	require.EqualError(t, CheckMD5Hashes(input, hash, hash, false), fmt.Sprintf("hash mismatch (16k): expected=%x, actual=%x", hash, hash16k))
	require.EqualError(t, CheckMD5Hashes(input, hash16k, hash16k, false), fmt.Sprintf("hash mismatch: expected=%x, actual=%x", hash16k, hash))
	require.EqualError(t, CheckMD5Hashes(input, hash, hash, true), fmt.Sprintf("hash mismatch (16k) in reconstructed data: expected=%x, actual=%x", hash, hash16k))
	require.EqualError(t, CheckMD5Hashes(input, hash16k, hash16k, true), fmt.Sprintf("hash mismatch in reconstructed data: expected=%x, actual=%x", hash16k, hash))
}
