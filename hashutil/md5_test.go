package hashutil

import (
	"bytes"
	"crypto/md5"
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

func TestMD5Hash16k(t *testing.T) {
	for _, input := range testInputs {
		hash16k := MD5Hash16k(input)
		if len(input) < 16*1024 {
			require.Equal(t, md5.Sum(input), hash16k)
		} else {
			require.Equal(t, md5.Sum(input[:16*1024]), hash16k)
		}
	}
}

func TestMD5HashWith16k(t *testing.T) {
	for _, input := range testInputs {
		hash, hash16k := MD5HashWith16k(input)
		require.Equal(t, md5.Sum(input), hash)
		require.Equal(t, MD5Hash16k(input), hash16k)
	}
}
