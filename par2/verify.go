package par2

import "github.com/akalin/gopar/fs"

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
	// ShardCounts contains shard counts which can be used to deduce
	// whether repair is necessary and/or possible.
	ShardCounts ShardCounts
}

// Verify a par file at parPath with the given options. The returned
// VerifyResult is not filled in if an error is returned.
func Verify(parPath string, options VerifyOptions) (VerifyResult, error) {
	return verify(fs.MakeDefaultFS(), parPath, options)
}

func verify(fs fs.FS, parPath string, options VerifyOptions) (VerifyResult, error) {
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

	decoder, err := newDecoder(fs, delegate, parPath, numGoroutines)
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

	return VerifyResult{
		ShardCounts: decoder.ShardCounts(),
	}, nil
}
