package par1

import (
	"errors"
	"path"
	"path/filepath"

	"github.com/akalin/gopar/fs"
)

// NumParityFilesDefault is the default value used for
// CreateOptions.NumParityFiles if the latter is <= 0.
const NumParityFilesDefault = 3

// CreateDelegate extends EncoderDelegate with another delegate
// function.
type CreateDelegate interface {
	EncoderDelegate
	OnFilesNotAllInSameDir()
}

// DoNothingCreateDelegate is an implementation of CreateDelegate that
// does nothing for all methods.
type DoNothingCreateDelegate struct{}

// OnDataFileLoad implements the CreateDelegate interface.
func (DoNothingCreateDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {}

// OnVolumeFileWrite implements the CreateDelegate interface.
func (DoNothingCreateDelegate) OnVolumeFileWrite(i, n int, path string, dataByteCount, byteCount int, err error) {
}

// OnFilesNotAllInSameDir implements the CreateDelegate interface.
func (DoNothingCreateDelegate) OnFilesNotAllInSameDir() {}

// CreateOptions holds all the options for Create.
type CreateOptions struct {
	// The number of parity files to create. If <= 0,
	// NumParityFilesDefault is used.
	NumParityFiles int
	// The CreateDelegate to use. If nil, DoNothingCreateDelegate
	// is used.
	CreateDelegate CreateDelegate
}

// Create a par file for the given file paths at parPath with the
// given options.
func Create(parPath string, filePaths []string, options CreateOptions) error {
	return create(fs.MakeDefaultFS(), parPath, filePaths, options)
}

func checkExtension(parPath string) error {
	ext := path.Ext(parPath)
	if ext != ".par" {
		return errors.New("parPath must have a .par extension")
	}
	return nil
}

func create(fs fs.FS, parPath string, filePaths []string, options CreateOptions) (err error) {
	err = checkExtension(parPath)
	if err != nil {
		return err
	}

	if len(filePaths) == 0 {
		return errors.New("filePaths must not be empty")
	}

	numParityFiles := options.NumParityFiles
	if numParityFiles <= 0 {
		numParityFiles = NumParityFilesDefault
	}

	delegate := options.CreateDelegate
	if delegate == nil {
		delegate = DoNothingCreateDelegate{}
	}

	parDir := filepath.Dir(parPath)
	filesAllInSameDir := true
	for _, p := range filePaths {
		if filepath.Dir(p) != parDir {
			filesAllInSameDir = false
			break
		}
	}
	if !filesAllInSameDir {
		delegate.OnFilesNotAllInSameDir()
	}

	encoder, err := newEncoder(fs, delegate, filePaths, numParityFiles)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := encoder.Close()
		if err == nil {
			err = closeErr
		}
	}()

	err = encoder.LoadFileData()
	if err != nil {
		return err
	}

	err = encoder.ComputeParityData()
	if err != nil {
		return err
	}

	return encoder.Write(parPath)
}
