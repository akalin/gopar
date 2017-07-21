package par2

import (
	"hash/crc32"
)

type crc32Window struct {
	windowSize    int
	crcWindowMask uint32
}

func newCRC32Window(windowSize int) crc32Window {
	if windowSize < 4 {
		panic("window size too small")
	}
	windowMask := make([]byte, windowSize+1)
	windowMask[0] = 0xff
	windowMask[4] = 0xff
	return crc32Window{
		windowSize:    windowSize,
		crcWindowMask: crc32.ChecksumIEEE(windowMask),
	}
}

// update returns the crc (crc32.ChecksumIEEE) of a[1:n+1], given the
// crc of a[0:n], oldLeader = a[0], and newTrailer = a[n], where n is
// the window size, and a[] is a "virtual" byte slice.
//
// TODO: Do so in constant time, i.e. independent of the window size.
func (w crc32Window) update(crc uint32, oldLeader, newTrailer byte) uint32 {
	crcExtended := crc32.Update(crc, crc32.IEEETable, []byte{newTrailer})

	oldLeaderPadded := make([]byte, w.windowSize+1)
	oldLeaderPadded[0] = oldLeader
	crcOldLeaderPadded := crc32.ChecksumIEEE(oldLeaderPadded)

	// TODO: Create a table of oldLeaderPadded ^ w.crcWindow for
	// each possible byte.

	// We'll show that that crc of a[1:n+1] is equal to
	//
	//   crc[0:n+1] ^ a[0].. ^ ff000000ff..,
	//
	// where .. denotes padding with trailing 0s to have length
	// matching the other operands.
	//
	// First, let
	//
	//   crc(a) = ffffffff ^ crcz(ffffffff.. ^ a).
	//
	// From the identity
	//
	//   crcz(a ^ b) = crcz(a) ^ crcz(b)
	//
	// we can deduce that
	//
	//   crc(a ^ b ^ c) = crc(a) ^ crc(b) ^ crc(c).
	//
	// Furthermore, recall that
	//
	//   crcz(00.a) = crc(a),
	//
	// where . denotes concatenation.
	//
	// Then,
	//
	//   crc(a[1:n+1]) = ffffffff ^ crcz(a[1:n+1] ^ ffffffff..)
	//                 = ffffffff ^ crcz(00.a[1:n+1] ^ 00ffffffff..)
	//                 = ffffffff ^ crcz((a[0:n+1] ^ a[0]..) ^
	//                                   (ff000000ff.. ^ ffffffff..))
	//                 = crc(a[0:n+1] ^ a[0].. ^ ff000000ff..)
	//                 = crc(a[0:n+1]) ^ crc(a[0]..) ^ crc(ff000000ff..).
	return crcExtended ^ crcOldLeaderPadded ^ w.crcWindowMask
}
