package hashutil

import (
	"io"

	"github.com/akalin/gopar/fs"
)

type teeReadStream struct {
	r fs.ReadStream
	w io.Writer
}

func (t teeReadStream) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		if n, err := t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return n, err
}

func (t teeReadStream) ReadAt(p []byte, off int64) (n int, err error) {
	return t.r.ReadAt(p, off)
}

func (t teeReadStream) Close() error {
	rErr := t.r.Close()
	var wErr error
	if writeCloser, ok := t.w.(io.Closer); ok {
		wErr = writeCloser.Close()
	}
	if rErr != nil {
		return rErr
	}
	return wErr
}

func (t teeReadStream) Offset() int64 {
	return t.r.Offset()
}

func (t teeReadStream) ByteCount() int64 {
	return t.r.ByteCount()
}

// TeeReadStream is like TeeReader but it takes an fs.ReadStream
// instead of an io.Reader. Note that only reads through Read are
// written to w, and not ReadAt.
func TeeReadStream(r fs.ReadStream, w io.Writer) fs.ReadStream {
	return teeReadStream{r, w}
}