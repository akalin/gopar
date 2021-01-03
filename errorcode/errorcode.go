package errorcode

type Errorcode int

const (
	Success						Errorcode = 0
	RepairPossible				Errorcode = 1
	RepairNotPossible			Errorcode = 2
	InvalidCommandLineArguments	Errorcode = 3
	InsufficientCriticalData	Errorcode = 4
	RepairFailed				Errorcode = 5
	FileIOError					Errorcode = 6
	LogicError					Errorcode = 7
	MemoryError					Errorcode = 8
)
