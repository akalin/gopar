package par2cmdline

// Taken from https://github.com/Parchive/par2cmdline/blob/master/src/libpar2.h#L111 .
const (
	ExitSuccess                     = 0
	ExitRepairPossible              = 1
	ExitRepairNotPossible           = 2
	ExitInvalidCommandLineArguments = 3
	ExitInsufficientCriticalData    = 4
	ExitRepairFailed                = 5
	ExitFileIOError                 = 6
	ExitLogicError                  = 7
	ExitMemoryError                 = 8
)
