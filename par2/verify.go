package par2

import (
	"github.com/akalin/gopar/par2cmdline"
	"github.com/akalin/gopar/rsec16"
)

// VerifyDelegate is just DecoderDelegate for now.
type VerifyDelegate interface {
	DecoderDelegate
}

// DoNothingVerifyDelegate is an implementation of VerifyDelegate that
// does nothing for all methods.
type DoNothingVerifyDelegate struct {
	DoNothingDecoderDelegate
}

// VerifyOptions holds all the options for Verify.
type VerifyOptions struct {
	// The number of goroutines to use while encoding. If <= 0,
	// NumGoroutinesDefault() is used.
	NumGoroutines int
	// The VerifyDelegate to use. If nil, DoNothingVerifyDelegate
	// is used.
	VerifyDelegate VerifyDelegate
}

// VerifyResult holds the result of a Verify call.
type VerifyResult struct {
	// NeedsRepair holds whether the set of files needs repair.
	NeedsRepair bool
}

// Verify a par file at parPath with the given options. The returned
// VerifyResult may be partially or not filled in if an error is
// returned.
func Verify(parPath string, options VerifyOptions) (VerifyResult, error) {
	return verify(defaultFileIO{}, parPath, options)
}

func verify(fileIO fileIO, parPath string, options VerifyOptions) (VerifyResult, error) {
	err := checkExtension(parPath)
	if err != nil {
		return VerifyResult{}, err
	}

	delegate := options.VerifyDelegate
	if delegate == nil {
		delegate = DoNothingVerifyDelegate{}
	}

	numGoroutines := options.NumGoroutines
	if numGoroutines <= 0 {
		numGoroutines = NumGoroutinesDefault()
	}

	decoder, err := newDecoder(fileIO, delegate, parPath, numGoroutines)
	if err != nil {
		return VerifyResult{}, err
	}

	err = decoder.LoadFileData()
	if err != nil {
		return VerifyResult{}, err
	}

	err = decoder.LoadParityData()
	if err != nil {
		return VerifyResult{}, err
	}

	needsRepair, err := decoder.Verify()
	return VerifyResult{NeedsRepair: needsRepair}, err
}

// ExitCodeForVerifyErrorPar2CmdLine returns the error code
// par2cmdline would have returned for the given error returned by
// Verify.
func ExitCodeForVerifyErrorPar2CmdLine(err error) int {
	if err != nil {
		switch err.(type) {
		case rsec16.NotEnoughParityShardsError:
			return par2cmdline.ExitRepairNotPossible
		default:
			return par2cmdline.ExitLogicError
		}
	}
	return par2cmdline.ExitSuccess
}
