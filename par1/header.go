package par1

type header struct {
	ID             [8]byte
	VersionNumber  uint64
	ControlHash    [16]byte
	SetHash        [16]byte
	VolumeNumber   uint64
	FileCount      uint64
	FileListOffset uint64
	FileListBytes  uint64
	DataOffset     uint64
	DataBytes      uint64
}
