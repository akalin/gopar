package fs

import (
	"fmt"
)

// Helperer is the interface that wraps the Helper method, usually
// implemented by *testing.T.
type Helperer interface {
	Helper()
}

type doNothingHelperer struct{}

func (doNothingHelperer) Helper() {}

// OpenFileManager holds the set of currently open files for a
// filesystem.
type OpenFileManager struct {
	h         Helperer
	openFiles map[string]bool
}

// MakeOpenFileManager returns an empty OpenFileManager.
func MakeOpenFileManager(h Helperer) OpenFileManager {
	if h == nil {
		h = doNothingHelperer{}
	}
	return OpenFileManager{h, make(map[string]bool)}
}

type checkingReadStream struct {
	h          Helperer
	readStream ReadStream
	openFiles  map[string]bool
	path       string
	closed     bool
}

func makeCheckingReadStream(h Helperer, readStream ReadStream, openFiles map[string]bool, path string) *checkingReadStream {
	return &checkingReadStream{h, readStream, openFiles, path, false}
}

func (crs checkingReadStream) verifyNotClosed() error {
	if crs.closed {
		return fmt.Errorf("%q is already closed", crs.path)
	}
	return nil
}

func (crs checkingReadStream) Read(p []byte) (int, error) {
	crs.h.Helper()
	if err := crs.verifyNotClosed(); err != nil {
		return 0, err
	}
	return crs.readStream.Read(p)
}

func (crs *checkingReadStream) Close() error {
	crs.h.Helper()
	if err := crs.verifyNotClosed(); err != nil {
		return err
	}
	crs.closed = true
	err := crs.readStream.Close()
	delete(crs.openFiles, crs.path)
	return err
}

func (crs checkingReadStream) ByteCount() int64 {
	crs.h.Helper()
	if err := crs.verifyNotClosed(); err != nil {
		panic(err)
	}
	return crs.readStream.ByteCount()
}

// GetReadStream wraps the given ReadStream getter in order to enforce
// that a file is opened at most once at a time.
func (ofm OpenFileManager) GetReadStream(path string, getReadStream func(string) (ReadStream, error)) (ReadStream, error) {
	ofm.h.Helper()
	// We don't try to detect the case where multiple paths point
	// to the same file.
	if ofm.openFiles[path] {
		return nil, fmt.Errorf("%q is already open", path)
	}

	readStream, err := getReadStream(path)
	if readStream != nil {
		ofm.openFiles[path] = true
		readStream = makeCheckingReadStream(ofm.h, readStream, ofm.openFiles, path)
	}
	return readStream, err
}

// GetOpenFilePaths returns a list of open files.
func (ofm OpenFileManager) GetOpenFilePaths() []string {
	var paths []string
	for path := range ofm.openFiles {
		paths = append(paths, path)
	}
	return paths
}
