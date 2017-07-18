package par2

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

func nullTerminate(bs []byte) []byte {
	// This emulates the direction in the spec to append a null
	// byte to turn a string into a null-terminated one.
	for i, b := range bs {
		if b == '\x00' {
			return bs[:i]
		}
	}

	return bs
}

func decodeNullPaddedASCIIString(bs []byte) string {
	// First, null-terminate if necessary.
	bs = nullTerminate(bs)

	var replaceBuf [4]byte
	n := utf8.EncodeRune(replaceBuf[:], unicode.ReplacementChar)

	// Replace all non-ASCII characters with the replacement character.
	var outBytes []byte
	for _, b := range bs {
		if b <= unicode.MaxASCII {
			outBytes = append(outBytes, b)
		} else {
			outBytes = append(outBytes, replaceBuf[:n]...)
		}
	}

	return string(outBytes)
}

func encodeASCIIString(s string) ([]byte, error) {
	var bs []byte
	for _, c := range s {
		if c > unicode.MaxASCII {
			return nil, errors.New("invalid ASCII character")
		}
		bs = append(bs, byte(c))
	}
	return bs, nil
}
