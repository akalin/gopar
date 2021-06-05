package fs

import (
	"fmt"
	"io"
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

type checkingCloser struct {
	closer    io.Closer
	openFiles map[string]bool
	path      string
	closed    bool
}

func (cc checkingCloser) verifyNotClosed() error {
	if cc.closed {
		return fmt.Errorf("%q is already closed", cc.path)
	}
	return nil
}

func (cc *checkingCloser) Close() error {
	if err := cc.verifyNotClosed(); err != nil {
		return err
	}
	cc.closed = true
	err := cc.closer.Close()
	delete(cc.openFiles, cc.path)
	return err
}

func makeCheckingCloser(closer io.Closer, openFiles map[string]bool, path string) *checkingCloser {
	return &checkingCloser{closer, openFiles, path, false}
}

type checkingReadStream struct {
	*checkingCloser
	readStream ReadStream
}

func makeCheckingReadStream(readStream ReadStream, openFiles map[string]bool, path string) checkingReadStream {
	return checkingReadStream{makeCheckingCloser(readStream, openFiles, path), readStream}
}

func (crs checkingReadStream) Read(p []byte) (int, error) {
	if err := crs.verifyNotClosed(); err != nil {
		return 0, err
	}
	return crs.readStream.Read(p)
}

func (crs checkingReadStream) ReadAt(p []byte, off int64) (int, error) {
	if err := crs.verifyNotClosed(); err != nil {
		return 0, err
	}
	return crs.readStream.ReadAt(p, off)
}

func (crs checkingReadStream) Offset() int64 {
	if err := crs.verifyNotClosed(); err != nil {
		panic(err)
	}
	return crs.readStream.Offset()
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

type checkingWriteStream struct {
	*checkingCloser
	writeStream WriteStream
}

func makeCheckingWriteStream(writeStream WriteStream, openFiles map[string]bool, path string) checkingWriteStream {
	return checkingWriteStream{makeCheckingCloser(writeStream, openFiles, path), writeStream}
}

func (crs checkingWriteStream) Write(p []byte) (int, error) {
	if err := crs.verifyNotClosed(); err != nil {
		return 0, err
	}
	return crs.writeStream.Write(p)
}

// GetWriteStream wraps the given WriteStream getter in order to
// enforce that a file is opened at most once at a time.
func (ofm OpenFileManager) GetWriteStream(path string, getWriteStream func(string) (WriteStream, error)) (WriteStream, error) {
	// We don't try to detect the case where multiple paths point
	// to the same file.
	if ofm.openFiles[path] {
		return nil, fmt.Errorf("%q is already open", path)
	}

	readStream, err := getWriteStream(path)
	if readStream != nil {
		ofm.openFiles[path] = true
		readStream = makeCheckingWriteStream(readStream, ofm.openFiles, path)
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
