package fs

import (
	"os"
	"path/filepath"
)

// defaultFS is a thin wrapper around existing I/O functions intended
// to be the default implementation for the FS interface.
type defaultFS struct {
	ofm OpenFileManager
}

// MakeDefaultFS returns an FS object that uses the underlying OS's
// I/O functions.
func MakeDefaultFS() FS { return defaultFS{MakeOpenFileManager(nil)} }

// closeOnError closes f if it and err is non-nil.
func closeOnError(f *os.File, err error) {
	if f != nil && err != nil {
		_ = f.Close()
	}
}

func (fs defaultFS) getReadStream(path string) (ReadStream, error) {
	f, err := os.Open(path)
	defer closeOnError(f, err)
	if err != nil {
		return nil, err
	}
	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return ReadCloserToStream(f, s.Size()), nil
}

// GetReadStream calls os.Open and also uses (*File).Stat to get the
// byte count of the opened file.
//
// Exactly one of the returned ReadStream and error is non-nil.
func (fs defaultFS) GetReadStream(path string) (ReadStream, error) {
	return fs.ofm.GetReadStream(path, fs.getReadStream)
}

// FindWithPrefixAndSuffix uses filepath.Glob to find files with the
// given prefix and suffix.
func (fs defaultFS) FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error) {
	return filepath.Glob(prefix + "*" + suffix)
}

func (fs defaultFS) getWriteStream(path string) (WriteStream, error) {
	f, err := os.Create(path)
	defer closeOnError(f, err)
	return f, err
}

// GetWriteStream calls os.Create.
//
// Exactly one of the returned WriteStream and error is non-nil.
func (fs defaultFS) GetWriteStream(path string) (WriteStream, error) {
	return fs.ofm.GetWriteStream(path, fs.getWriteStream)
}
