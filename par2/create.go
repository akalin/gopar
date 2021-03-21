package par2

import (
	"errors"
	"path"
	"path/filepath"

	"github.com/akalin/gopar/par2cmdline"
	"github.com/akalin/gopar/rsec16"
)

// SliceByteCountDefault is the default value used for
// CreateOptions.SliceByteCount if the latter is <= 0.
const SliceByteCountDefault = 2000

// NumParityShardsDefault is the default value used for
// CreateOptions.NumParityShards if the latter is <= 0.
const NumParityShardsDefault = 3

// NumGoroutinesDefault returns the default value used for
// CreateOptions.NumGoRoutines the latter is <= 0.
func NumGoroutinesDefault() int {
	return rsec16.DefaultNumGoroutines()
}

// CreateDelegate is just EncoderDelegate for now.
type CreateDelegate interface {
	EncoderDelegate
}

// DoNothingCreateDelegate is an implementation of CreateDelegate that
// does nothing for all methods.
type DoNothingCreateDelegate struct{}

// OnDataFileLoad implements the CreateDelegate interface.
func (DoNothingCreateDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {
}

// OnIndexFileWrite implements the CreateDelegate interface.
func (DoNothingCreateDelegate) OnIndexFileWrite(path string, byteCount int, err error) {
}

// OnRecoveryFileWrite implements the CreateDelegate interface.
func (DoNothingCreateDelegate) OnRecoveryFileWrite(start, count, total int, path string, dataByteCount, byteCount int, err error) {
}

// CreateOptions holds all the options for Create.
type CreateOptions struct {
	// How big each slice should be in bytes. If <= 0,
	// SliceByteCountDefault is used.
	SliceByteCount int
	// The number of parity shards to create. If <= 0,
	// NumParityShardsDefault is used.
	NumParityShards int
	// The number of goroutines to use while encoding. If <= 0,
	// NumGoroutinesDefault() is used.
	NumGoroutines int
	// The CreateDelegate to use. If nil, DoNothingCreateDelegate
	// is used.
	CreateDelegate CreateDelegate
}

// Create a par file for the given file paths at parPath with the
// given options.
func Create(parPath string, filePaths []string, options CreateOptions) error {
	return create(defaultFileIO{}, parPath, filePaths, options)
}

func create(fileIO fileIO, parPath string, filePaths []string, options CreateOptions) error {
	ext := path.Ext(parPath)
	if ext != ".par2" {
		return errors.New("parPath must have a .par2 extension")
	}

	if len(filePaths) == 0 {
		return errors.New("filePaths must not be empty")
	}

	sliceByteCount := options.SliceByteCount
	if sliceByteCount <= 0 {
		sliceByteCount = SliceByteCountDefault
	}

	numParityShards := options.NumParityShards
	if numParityShards <= 0 {
		numParityShards = NumParityShardsDefault
	}

	numGoroutines := options.NumGoroutines
	if numGoroutines <= 0 {
		numGoroutines = NumGoroutinesDefault()
	}

	delegate := options.CreateDelegate
	if delegate == nil {
		delegate = DoNothingCreateDelegate{}
	}

	absParPath, err := filepath.Abs(parPath)
	if err != nil {
		return err
	}
	basePath := filepath.Dir(absParPath)
	absFilePaths := make([]string, len(filePaths))
	for i, path := range filePaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		absFilePaths[i] = absPath
	}

	encoder, err := newEncoder(fileIO, delegate, basePath, absFilePaths, sliceByteCount, numParityShards, numGoroutines)
	if err != nil {
		return err
	}

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

// ExitCodeForCreateErrorPar2CmdLine returns the error code
// par2cmdline would have returned for the given error returned by
// Create.
func ExitCodeForCreateErrorPar2CmdLine(err error) int {
	if err != nil {
		// Map everything to eFileIOError for now.
		return par2cmdline.ExitFileIOError
	}
	return par2cmdline.ExitSuccess
}
