package hashutil

import (
	"bytes"
	"crypto/md5"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMD5Hash16k(t *testing.T) {
	inputs := [][]byte{
		{},
		bytes.Repeat([]byte{0x1}, 100),
		bytes.Repeat([]byte{0x2}, 16*1024-1),
		bytes.Repeat([]byte{0x3}, 16*1024),
		bytes.Repeat([]byte{0x4}, 16*1024+1),
	}
	for _, input := range inputs {
		hash16k := MD5Hash16k(input)
		if len(input) < 16*1024 {
			require.Equal(t, md5.Sum(input), hash16k)
		} else {
			require.Equal(t, md5.Sum(input[:16*1024]), hash16k)
		}
	}
}
