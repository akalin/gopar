package fs

import (
	"io/ioutil"
	"path/filepath"
)

// defaultFS is a thin wrapper around existing I/O functions intended
// to be the default implementation for the FS interface.
type defaultFS struct{}

// MakeDefaultFS returns an FS object that uses the underlying OS's
// I/O functions.
func MakeDefaultFS() FS { return defaultFS{} }

// ReadFile simply calls ioutil.ReadFile.
func (fs defaultFS) ReadFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

// FindWithPrefixAndSuffix uses filepath.Glob to find files with the
// given prefix and suffix.
func (fs defaultFS) FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error) {
	return filepath.Glob(prefix + "*" + suffix)
}

// WriteFile simply calls ioutil.WriteFile.
func (fs defaultFS) WriteFile(path string, data []byte) error {
	return ioutil.WriteFile(path, data, 0600)
}
