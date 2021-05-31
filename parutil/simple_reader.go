package parutil

import (
	"errors"
	"io"
)

type eagerReader struct {
	reader io.Reader
}

func (er eagerReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, errors.New("0-length buffer passed to eagerReader.read")
	}

	for n < len(p) && err == nil {
		var readBytes int
		readBytes, err = er.reader.Read(p[n:])
		if readBytes == 0 && err == nil {
			err = errors.New("reader returned 0, nil")
		}
		n += readBytes
	}
	return n, err
}
