package fs

import (
	"fmt"
)

// OpenFileManager holds the set of currently open files for a
// filesystem.
type OpenFileManager struct {
	openFiles map[string]bool
}

// MakeOpenFileManager returns an empty OpenFileManager.
func MakeOpenFileManager() OpenFileManager {
	return OpenFileManager{make(map[string]bool)}
}

type checkingReadStream struct {
	readStream ReadStream
	openFiles  map[string]bool
	path       string
	closed     bool
}

func makeCheckingReadStream(readStream ReadStream, openFiles map[string]bool, path string) *checkingReadStream {
	return &checkingReadStream{readStream, openFiles, path, false}
}

func (crs checkingReadStream) verifyNotClosed() error {
	if crs.closed {
		return fmt.Errorf("%q is already closed", crs.path)
	}
	return nil
}

func (crs checkingReadStream) Read(p []byte) (int, error) {
	if err := crs.verifyNotClosed(); err != nil {
		return 0, err
	}
	return crs.readStream.Read(p)
}

func (crs *checkingReadStream) Close() error {
	if err := crs.verifyNotClosed(); err != nil {
		return err
	}
	crs.closed = true
	err := crs.readStream.Close()
	delete(crs.openFiles, crs.path)
	return err
}

func (crs checkingReadStream) ByteCount() int64 {
	if err := crs.verifyNotClosed(); err != nil {
		panic(err)
	}
	return crs.readStream.ByteCount()
}

// GetReadStream wraps the given ReadStream getter in order to enforce
// that a file is opened at most once at a time.
func (ofm OpenFileManager) GetReadStream(path string, getReadStream func(string) (ReadStream, error)) (ReadStream, error) {
	// We don't try to detect the case where multiple paths point
	// to the same file.
	if ofm.openFiles[path] {
		return nil, fmt.Errorf("%q is already open", path)
	}

	readStream, err := getReadStream(path)
	if readStream != nil {
		ofm.openFiles[path] = true
		readStream = makeCheckingReadStream(readStream, ofm.openFiles, path)
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
