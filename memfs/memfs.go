package memfs

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/akalin/gopar/fs"
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
	ofm        fs.OpenFileManager
}

// MakeMemFS makes a MemFS from the given working directory and file
// data.
func MakeMemFS(workingDir string, fileData map[string][]byte) MemFS {
	return MemFS{workingDir, fileDataToAbsPaths(workingDir, fileData), fs.MakeOpenFileManager(nil)}
}

// ReadFile returns the data of the file at the given path, which may
// be absolute or relative (to the working directory). If the file
// doesn't exist, os.ErrNotExist is returned.
func (mfs MemFS) ReadFile(path string) (data []byte, err error) {
	absPath := toAbsPath(mfs.workingDir, path)
	if data, ok := mfs.fileData[absPath]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

type readerCloser struct {
	*bytes.Reader
}

func (r readerCloser) Close() error { return nil }

// MakeReadStream returns a fs.ReadStream with the given data.
func MakeReadStream(data []byte) fs.ReadStream {
	return fs.ReadCloserToStream(readerCloser{bytes.NewReader(data)}, int64(len(data)))
}

func (mfs MemFS) getReadStream(path string) (fs.ReadStream, error) {
	absPath := toAbsPath(mfs.workingDir, path)
	if data, ok := mfs.fileData[absPath]; ok {
		return MakeReadStream(data), nil
	}
	return nil, os.ErrNotExist
}

// GetReadStream returns an fs.ReadStream for the file at the given
// path, which may be absolute or relative (to the working directory),
// as well as its size. If the file doesn't exist, os.ErrNotExist is
// returned.
//
// Exactly one of the returned ReadStream and error is non-nil.
func (mfs MemFS) GetReadStream(path string) (fs.ReadStream, error) {
	return mfs.ofm.GetReadStream(path, mfs.getReadStream)
}

// FindWithPrefixAndSuffix returns all files whose path matches the
// given prefix and suffix, in no particular order. The prefix may be
// absolute or relative (to the working directory).
func (mfs MemFS) FindWithPrefixAndSuffix(prefix, suffix string) ([]string, error) {
	absPrefix := toAbsPath(mfs.workingDir, prefix)
	var matches []string
	for _, filename := range mfs.Paths() {
		if len(filename) >= len(absPrefix)+len(suffix) && strings.HasPrefix(filename, absPrefix) && strings.HasSuffix(filename, suffix) {
			matches = append(matches, filename)
		}
	}
	return matches, nil
}

// WriteFile sets the data of the file at the given path, which may be
// absolute or relative (to the working directory). The file may or
// may not already exist.
func (mfs MemFS) WriteFile(path string, data []byte) error {
	absPath := toAbsPath(mfs.workingDir, path)
	mfs.fileData[absPath] = data
	return nil
}

type writeStream struct {
	*bytes.Buffer
	fileData map[string][]byte
	absPath  string
}

func (w writeStream) Close() error {
	w.fileData[w.absPath] = w.Buffer.Bytes()
	return nil
}

func (mfs MemFS) getWriteStream(path string) (fs.WriteStream, error) {
	absPath := toAbsPath(mfs.workingDir, path)
	// Make the data only visible on close.
	mfs.fileData[absPath] = nil
	// Use []byte{} instead of nil for consistency, and because
	// some tests expect it.
	return writeStream{bytes.NewBuffer([]byte{}), mfs.fileData, absPath}, nil
}

// GetWriteStream truncates the file at the given path, which may be
// absolute or relative (to the working directory), if it exists, and
// returns a fs.WriteStream for the file.
//
// Exactly one of the returned WriteStream and error is non-nil.
func (mfs MemFS) GetWriteStream(path string) (fs.WriteStream, error) {
	return mfs.ofm.GetWriteStream(path, mfs.getWriteStream)
}

// FileCount returns the total number of files.
func (mfs MemFS) FileCount() int {
	return len(mfs.fileData)
}

// Paths returns a list of absolute paths of files in fs in no particular order.
func (mfs MemFS) Paths() []string {
	var paths []string
	for path := range mfs.fileData {
		paths = append(paths, path)
	}
	return paths
}

// RemoveFile removes the file at the given path, which may be
// absolute or relative (to the working directory). The removed data
// is returned, or os.ErrNotExist if it doesn't exist.
func (mfs MemFS) RemoveFile(path string) ([]byte, error) {
	absPath := toAbsPath(mfs.workingDir, path)
	data, err := mfs.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	delete(mfs.fileData, absPath)
	return data, nil
}

// MoveFile moves the file at oldPath to newPath. oldPath and newPath
// may be either absolute or relative (to the working directory). If
// the file doesn't exist at oldPath, os.ErrNotExist is returned.
func (mfs MemFS) MoveFile(oldPath, newPath string) error {
	data, err := mfs.RemoveFile(oldPath)
	if err != nil {
		return err
	}
	// Shouldn't return an error.
	return mfs.WriteFile(newPath, data)
}
