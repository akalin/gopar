package memfs

import (
	"os"
	"path/filepath"
	"strings"
)

// RootDir returns a string representing a root directory. On
// Unix-like systems this is just /, but on Windows it may be C:\ or
// some other drive letter.
func RootDir() string {
	// This complexity is only for Windows, the only platform
	// which has the concept of a VolumeName, e.g. C:. We don't
	// care which drive the current working directory is on. On
	// all other platforms, volName is empty.
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	volName := filepath.VolumeName(filepath.Clean(wd))
	return volName + string(filepath.Separator)
}

func fileDataToAbsPaths(workingDir string, fileData map[string][]byte) map[string][]byte {
	newFileData := make(map[string][]byte)
	for path, data := range fileData {
		newFileData[toAbsPath(workingDir, path)] = data
	}
	return newFileData
}

func toAbsPath(workingDir, path string) string {
	if !filepath.IsAbs(workingDir) {
		panic("workingDir must be an absolute path")
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(workingDir, path)
}

// MemFS is a simple in-memory filesystem with a working
// directory. It's intended mainly for testing.
type MemFS struct {
	workingDir string
	fileData   map[string][]byte
}

// MakeMemFS makes a MemFS from the given working directory and file
// data.
func MakeMemFS(workingDir string, fileData map[string][]byte) MemFS {
	return MemFS{workingDir, fileDataToAbsPaths(workingDir, fileData)}
}

// ReadFile returns the data of the file at the given path, which may
// be absolute or relative (to the working directory). If the file
// doesn't exist, os.ErrNotExist is returned.
func (fs MemFS) ReadFile(path string) (data []byte, err error) {
	absPath := toAbsPath(fs.workingDir, path)
	if data, ok := fs.fileData[absPath]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

// FindWithPrefixAndSuffix returns all files whose path matches the
// given prefix and suffix, in no particular order. The prefix may be
// absolute or relative (to the working directory).
func (fs MemFS) FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error) {
	absPrefix := toAbsPath(fs.workingDir, prefix)
	var matches []string
	for _, filename := range fs.Paths() {
		if len(filename) >= len(absPrefix)+len(suffix) && strings.HasPrefix(filename, absPrefix) && strings.HasSuffix(filename, suffix) {
			matches = append(matches, filename)
		}
	}
	return matches, nil
}

// WriteFile sets the data of the file at the given path, which may be
// absolute or relative (to the working directory). The file may or
// may not already exist.
func (fs MemFS) WriteFile(path string, data []byte) error {
	absPath := toAbsPath(fs.workingDir, path)
	fs.fileData[absPath] = data
	return nil
}

// FileCount returns the total number of files.
func (fs MemFS) FileCount() int {
	return len(fs.fileData)
}

// Paths returns a list of absolute paths of files in fs in no particular order.
func (fs MemFS) Paths() []string {
	var paths []string
	for path := range fs.fileData {
		paths = append(paths, path)
	}
	return paths
}

// RemoveFile removes the file at the given path, which may be
// absolute or relative (to the working directory). The removed data
// is returned, or os.ErrNotExist if it doesn't exist.
func (fs MemFS) RemoveFile(path string) ([]byte, error) {
	absPath := toAbsPath(fs.workingDir, path)
	data, err := fs.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	delete(fs.fileData, absPath)
	return data, nil
}

// MoveFile moves the file at oldPath to newPath. oldPath and newPath
// may be either absolute or relative (to the working directory). If
// the file doesn't exist at oldPath, os.ErrNotExist is returned.
func (fs MemFS) MoveFile(oldPath, newPath string) error {
	data, err := fs.RemoveFile(oldPath)
	if err != nil {
		return err
	}
	// Shouldn't return an error.
	return fs.WriteFile(newPath, data)
}
