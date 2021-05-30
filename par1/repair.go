package par1

import (
	"github.com/akalin/gopar/fs"
	"github.com/klauspost/reedsolomon"
)

// RepairDelegate is just DecoderDelegate for now.
type RepairDelegate interface {
	DecoderDelegate
}

// DoNothingRepairDelegate is an implementation of RepairDelegate that
// does nothing for all methods.
type DoNothingRepairDelegate struct {
	DoNothingDecoderDelegate
}

// RepairOptions holds all the options for Repair.
type RepairOptions struct {
	// If DoubleCheck is true, then extra checking is done after
	// the repair to verify that the repaired shards are correct.
	DoubleCheck bool
	// The RepairDelegate to use. If nil, DoNothingRepairDelegate
	// is used.
	RepairDelegate RepairDelegate
}

// RepairResult holds the result of a Repair call.
type RepairResult struct {
	// RepairedPaths contains the paths of the files that were
	// repaired.
	RepairedPaths []string
}

// Repair a par file at parPath with the given options. The returned
// RepairResult may be partially or not filled in if an error is
// returned.
func Repair(parPath string, options RepairOptions) (RepairResult, error) {
	return repair(fs.MakeDefaultFS(), parPath, options)
}

func repair(fs fs.FS, parPath string, options RepairOptions) (RepairResult, error) {
	err := checkExtension(parPath)
	if err != nil {
		return RepairResult{}, err
	}

	delegate := options.RepairDelegate
	if delegate == nil {
		delegate = DoNothingRepairDelegate{}
	}

	decoder, err := newDecoder(fs, delegate, parPath)
	if err != nil {
		return RepairResult{}, err
	}

	err = decoder.LoadFileData()
	if err != nil {
		return RepairResult{}, err
	}

	err = decoder.LoadParityData()
	if err != nil {
		return RepairResult{}, err
	}

	repairedPaths, err := decoder.Repair(options.DoubleCheck)
	return RepairResult{
		RepairedPaths: repairedPaths,
	}, err
}

// RepairErrorMeansRepairNecessaryButNotPossible returns true if the
// error returned by Repair means that repair is necessary but not
// possible.
func RepairErrorMeansRepairNecessaryButNotPossible(err error) bool {
	return err == reedsolomon.ErrTooFewShards
}
