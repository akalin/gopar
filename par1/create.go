package par1

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
)

// NumParityFilesDefault is the default value used for
// CreateOptions.NumParityFiles if the latter is <= 0.
const NumParityFilesDefault = 3

// DoNothingEncoderDelegate is an implementation of EncoderDelegate
// that does nothing for all methods.
type DoNothingEncoderDelegate struct{}

// OnDataFileLoad implements the EncoderDelegate interface.
func (DoNothingEncoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {}

// OnVolumeFileWrite implements the EncoderDelegate interface.
func (DoNothingEncoderDelegate) OnVolumeFileWrite(i, n int, path string, dataByteCount, byteCount int, err error) {
}

// CreateOptions holds all the options for Create.
type CreateOptions struct {
	// The number of parity files to create. If <= 0,
	// NumParityFilesDefault is used.
	NumParityFiles int
	// The EncoderDelegate to use. If nil,
	// DoNothingEncoderDelegate is used.
	EncoderDelegate EncoderDelegate
}

// Create a par file for the given file paths at parPath with the
// given options.
func Create(parPath string, filePaths []string, options CreateOptions) error {
	ext := path.Ext(parPath)
	if ext != ".par" {
		return errors.New("parPath must have a .par extension")
	}

	if len(filePaths) == 0 {
		return errors.New("filePaths must not be empty")
	}

	numParityFiles := options.NumParityFiles
	if numParityFiles <= 0 {
		numParityFiles = NumParityFilesDefault
	}

	delegate := options.EncoderDelegate
	if delegate == nil {
		delegate = DoNothingEncoderDelegate{}
	}

	parDir := filepath.Dir(parPath)
	allFilesInSameDir := true
	for _, p := range filePaths {
		if filepath.Dir(p) != parDir {
			allFilesInSameDir = false
			break
		}
	}
	// TODO: Use a delegate method for this.
	if !allFilesInSameDir {
		fmt.Printf("Warning: PAR and data files not all in the same directory, which a decoder will expect\n")
	}

	encoder, err := NewEncoder(delegate, filePaths, numParityFiles)
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
