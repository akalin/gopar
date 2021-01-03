package par2

import (
	"errors"
	"testing"

	"github.com/akalin/gopar/errorcode"
	"github.com/stretchr/testify/require"
)

func TestDecodeASCIIString(t *testing.T) {
	strings := []string{
		"hello world",
		"hello\nworld",
		"\x01\x7f",
	}

	for _, s := range strings {
		require.Equal(t, s, decodeNullPaddedASCIIString([]byte(s)))
	}
}

func TestDecodeASCIIStringNullByte(t *testing.T) {
	s := "hello\x00world"
	require.Equal(t, "hello", decodeNullPaddedASCIIString([]byte(s)))
}

func TestDecodeNonASCIIString(t *testing.T) {
	s := "hello\x80world"
	require.Equal(t, "hello\uFFFDworld", decodeNullPaddedASCIIString([]byte(s)))
}

func TestEncodeASCIIString(t *testing.T) {
	strings := []string{
		"hello world",
		"hello\nworld",
		"\x01\x7f",
	}

	for _, s := range strings {
		bs, err := encodeASCIIString(s)
		require.NoError(t, err)
		require.Equal(t, []byte(s), bs)
	}
}

func TestEncodeNonASCIIString(t *testing.T) {
	s := "hello\x80world"
	_, err := encodeASCIIString(s)
	require.Equal(t, errorcode.InvalidASCII, err)
}
