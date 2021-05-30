package fs

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// DefaultFS is a thin wrapper around existing I/O functions intended
// to be the default implementation for the FS interface.
type DefaultFS struct{}

// ReadFile simply calls ioutil.ReadFile.
func (fs DefaultFS) ReadFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

type fileWithByteCount struct {
	*os.File
	ByteCountHolder
}

// closeOnError closes f if it and err is non-nil.
func closeOnError(f *os.File, err error) {
	if f != nil && err != nil {
		_ = f.Close()
	}
}

// GetReadStream calls os.Open and also uses (*File).Stat to get the
// byte count of the opened file.
//
// Exactly one of the returned ReadStream and error is non-nil.
func (fs DefaultFS) GetReadStream(path string) (ReadStream, error) {
	f, err := os.Open(path)
	defer closeOnError(f, err)
	if err != nil {
		return nil, err
	}
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return fileWithByteCount{f, ByteCountHolder{Count: s.Size()}}, err
}

// FindWithPrefixAndSuffix uses filepath.Glob to find files with the
// given prefix and suffix.
func (fs DefaultFS) FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error) {
	return filepath.Glob(prefix + "*" + suffix)
}

// WriteFile simply calls ioutil.WriteFile.
func (fs DefaultFS) WriteFile(path string, data []byte) error {
	return ioutil.WriteFile(path, data, 0600)
}

// GetWriteStream calls os.Create.
//
// Exactly one of the returned WriteStream and error is non-nil.
func (fs DefaultFS) GetWriteStream(path string) (WriteStream, error) {
	f, err := os.Create(path)
	defer closeOnError(f, err)
	return f, err
}
