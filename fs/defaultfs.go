package fs

import (
	"io/ioutil"
	"path/filepath"
)

// DefaultFS is a thin wrapper around existing I/O functions intended
// to be the default implementation for the FS interface.
type DefaultFS struct{}

// ReadFile simply calls ioutil.ReadFile.
func (fs DefaultFS) ReadFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
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
