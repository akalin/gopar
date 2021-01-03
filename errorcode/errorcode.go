package errorcode

import (
	"errors"
)

type Errorcode int
const (
	Success						Errorcode = 0
	RepairPossible				Errorcode = 1
	RepairNotPossible			Errorcode = 2
	InvalidCommandLineArguments	Errorcode = 3
	InsufficientCriticalData	Errorcode = 4
//	RepairFailedCode			Errorcode = 5
	FileIOError					Errorcode = 6
	LogicError					Errorcode = 7
	MemoryError					Errorcode = 8
)

var NoCommandSpecified						error = errors.New("no command specified")
var NoParFileSpecified						error = errors.New("no PAR file specified")
var NoDataFilesSpecified					error = errors.New("no data files specified")
var SingularMatrix							error = errors.New("singular matrix")
var UnexpectedSetHash						error = errors.New("unexpected set hash for parity volume")
var InvalidEntryByteCount					error = errors.New("invalid entry byte count")
var ByteCountMismatch						error = errors.New("byte count mismatch")
var UnexpectedIDString						error = errors.New("unexpected ID string")
var UnexpectedVersion						error = errors.New("unexpected version")
var UnexpectedFileListOffset				error = errors.New("unexpected file list offset")
var InvalidControlHash						error = errors.New("invalid control hash")
var ExpectedVolumeNumberIndex				error = errors.New("expected volume number 0 for index volume")
var HashMismatch16k							error = errors.New("hash mismatch (16k)")
var HashMismatch							error = errors.New("hash mismatch")
var NoFileDataFound							error = errors.New("no file data found")
var UnexpectedVolumeNumber					error = errors.New("unexpected volume number for parity volume")
var NoParityDataInVolume					error = errors.New("no parity data in volume")
var MismatchedParityDataByteCounts			error = errors.New("mismatched parity data byte counts")
var RepairFailed							error = errors.New("repair failed")
var HashMismatch16kInReconstructedData		error = errors.New("hash mismatch (16k) in reconstructed data")
var HashMismatchInReconstructedData			error = errors.New("hash mismatch in reconstructed data")
var InvalidSliceByteCount					error = errors.New("invalid slice byte count")
var NoPacketsFound							error = errors.New("no packets found")
var NoCreatorPacketFound					error = errors.New("no creator packet found")
var RecoveryPacketDuplicateExpDiffContent	error = errors.New("recovery packet with duplicate exponent but differing contents")
var EmptyClientID							error = errors.New("empty client ID")
var NoMainPacket							error = errors.New("no main packet")
var NoFileDescriptionPacket					error = errors.New("could not find file description packet")
var NoInputFileSliceChecksumPacket			error = errors.New("could not find input file slice checksum packet")
var AbsPathNotAllowed						error = errors.New("absolute paths not allowed")
var OutsideCurDirNotAllowed					error = errors.New("traversing outside of the current directory is not allowed")
var FileIDMismatch							error = errors.New("file ID mismatch")
var EmptyFilesNotAllowed					error = errors.New("empty files not allowed")
var FileLengthTooBig						error = errors.New("file length too big")
var InvalidByteCount						error = errors.New("invalid byte count")
var InvalidSize								error = errors.New("invalid size")
var NoChecksumPairsToWrite					error = errors.New("no checksum pairs to write")
var RecoverySetIDsNotSorted					error = errors.New("recovery set IDs not sorted")
var NonRecoverySetIDsNotSorted				error = errors.New("non-recovery set IDs not sorted")
var InvalidSliceSize						error = errors.New("invalid slice size")
var EmptyRecoverySet						error = errors.New("empty recovery set")
var NotEnoughFileIDs						error = errors.New("not enough file IDs")
var RecoverySetTooBig						error = errors.New("recovery set too big")
var UnexpectedMagicString					error = errors.New("unexpected magic string")
var InvalidLength							error = errors.New("invalid length")
var CouldNotReadBody						error = errors.New("could not read body")
var InvalidRecoveryDataByteCount			error = errors.New("invalid recovery data byte count")
var ExponentOutOfRange						error = errors.New("exponent out of range")
var InvalidASCII							error = errors.New("invalid ASCII character")
var RecoveryPacketsFoundInIndexFile			error = errors.New("recovery packets found in index file")
var SliceByteCountMismatch					error = errors.New("slice byte count mismatch")
var RecoverySetMismatch						error = errors.New("recovery set mismatch")
var NonRecoverySetMismatch					error = errors.New("non-recovery set mismatch")
var NoFileIntegrityInfo						error = errors.New("no file integrity info")
var NoParityData							error = errors.New("no parity data")
var NoParityShards							error = errors.New("no parity shards")
var NotEnoughParityShards					error = errors.New("not enough parity shards")
var TooManyShards							error = errors.New("too many shards")
var TooManyDataShards						error = errors.New("too many data shards")
var TooManyParityShards						error = errors.New("too many parity shards")
