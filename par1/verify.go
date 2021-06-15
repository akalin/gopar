package par1

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
	// If VerifyAllData is true, then check whether all data and
	// parity files contain correct data even if no missing or
	// corrupt files are detected.
	VerifyAllData bool
	// The VerifyDelegate to use. If nil, DoNothingVerifyDelegate
	// is used.
	VerifyDelegate VerifyDelegate
}

// VerifyResult holds the result of a Verify call.
type VerifyResult struct {
	// FileCounts contains file counts which can be used to deduce
	// whether repair is necessary and/or possible.
	FileCounts FileCounts
	// AllDataOk contains the result of calling VerifyAllData(),
	// when VerifyAllData is set to true in VerifyOptions and
	// FileCounts.AllFilesUsable() returns true, and contains
	// false otherwise.
	AllDataOk bool
}

// Verify a par file at parPath with the given options. The returned
// VerifyResult is not filled in if an error is returned.
func Verify(parPath string, options VerifyOptions) (VerifyResult, error) {
	return verify(fs.MakeDefaultFS(), parPath, options)
}

func verify(filesystem fs.FS, parPath string, options VerifyOptions) (result VerifyResult, err error) {
	err = checkExtension(parPath)
	if err != nil {
		return VerifyResult{}, err
	}

	delegate := options.VerifyDelegate
	if delegate == nil {
		delegate = DoNothingVerifyDelegate{}
	}

	decoder := newDecoder(filesystem, delegate, parPath)
	defer fs.CloseCloser(decoder, &err)

	err = decoder.LoadIndexFile()
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

	fileCounts := decoder.FileCounts()

	allDataOk := false
	if options.VerifyAllData && fileCounts.AllFilesUsable() {
		allDataOk, err = decoder.VerifyAllData()
		if err != nil {
			return VerifyResult{}, err
		}
	}
	return VerifyResult{
		FileCounts: fileCounts,
		AllDataOk:  allDataOk,
	}, nil
}
