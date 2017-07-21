package par2

import (
	"hash/crc32"
)

type crc32Window struct {
	windowSize              int
	crcOldLeaderMaskedTable [256]uint32
}

func newCRC32Window(windowSize int) crc32Window {
	if windowSize < 4 {
		panic("window size too small")
	}
	windowMask := make([]byte, windowSize+1)
	windowMask[0] = 0xff
	windowMask[4] = 0xff
	crcWindowMask := crc32.ChecksumIEEE(windowMask)

	var crcOldLeaderMaskedTable [256]uint32
	oldLeaderPadded := make([]byte, windowSize+1)
	for i := 0; i < 256; i++ {
		oldLeaderPadded[0] = byte(i)
		crcOldLeaderPadded := crc32.ChecksumIEEE(oldLeaderPadded)
		crcOldLeaderMaskedTable[i] = crcOldLeaderPadded ^ crcWindowMask
	}

	return crc32Window{
		windowSize:              windowSize,
		crcOldLeaderMaskedTable: crcOldLeaderMaskedTable,
	}
}

// update returns the crc (crc32.ChecksumIEEE) of a[1:n+1], given the
// crc of a[0:n], oldLeader = a[0], and newTrailer = a[n], where n is
// the window size, and a[] is a "virtual" byte slice. It does so in
// constant time, i.e. independent of the window size.
func (w crc32Window) update(crc uint32, oldLeader, newTrailer byte) uint32 {
	crcExtended := crc32.Update(crc, crc32.IEEETable, []byte{newTrailer})
	crcOldLeaderMasked := w.crcOldLeaderMaskedTable[oldLeader]

	// We'll show that
	//
	//   crc(a[1:n+1]) =
	//     crc(a[0:n+1]) ^ crc(a[0]..) ^ crc(ff000000ff..),
	//
	// where .. denotes padding with trailing 0s to have length n+1.
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
	//
	// Finally, we can precompute the latter two terms and build a
	// table indexed by a[0], which is precisely crcOldLeaderMasked.
	return crcExtended ^ crcOldLeaderMasked
}
