package par2

import (
	"hash/crc32"
)

type crc32Window struct {
	windowSize              int
	crcOldLeaderMaskedTable [256]uint32
}

func newCRC32Window(windowSize int) *crc32Window {
	if windowSize < 4 {
		panic("window size too small")
	}
	a := make([]byte, windowSize+1)
	crc0 := crc32.ChecksumIEEE(a)

	// See comments in crc32Window.update below for why we need
	// crcWindowMask.
	a[0] = 0xff
	a[4] = 0xff
	crcWindowMask := crc32.ChecksumIEEE(a)

	// Set baseTable[i] to crc((1 << i)..) for 0 <= i < 8,
	// where .. denotes padding with trailing 0s to have length
	// windowSize+1.
	a[4] = 0
	var baseTable [8]uint32
	for i := uint(0); i < 8; i++ {
		a[0] = byte(1 << i)
		baseTable[i] = crc32.ChecksumIEEE(a)
	}

	var maskedTable [256]uint32
	// Set maskedTable[i] to crc(i..) ^ crcWindowMask for
	// 0 <= i < 256.
	maskedTable[0] = crc0 ^ crcWindowMask
	for i := 1; i < 256; i++ {
		// Compute crc(i..) using baseTable and the identities
		//
		//   crc(a_1 ^ ... ^ a_n) = crc(a_1) ^ ... ^ crc(a^n)
		//
		// for odd n, and
		//
		//   crc(a_1 ^ ... ^ a_n) =
		//     crc(a_1) ^ ... ^ crc(a^n) ^ crc(0..)
		//
		// for even n, which can be deduced from letting
		//
		//   crc(a) = ffffffff ^ crcz(ffffffff.. ^ a)
		//
		// and the identity
		//
		//   crcz(a ^ b) = crcz(a) ^ crcz(b).
		var crc uint32
		crcCount := 0
		for j := uint(0); j < 8; j++ {
			if i&(1<<j) != 0 {
				crc ^= baseTable[j]
				crcCount++
			}
		}
		if crcCount%2 == 0 {
			crc ^= crc0
		}
		maskedTable[i] = crc ^ crcWindowMask
	}

	return &crc32Window{
		windowSize:              windowSize,
		crcOldLeaderMaskedTable: maskedTable,
	}
}

// update returns the crc (crc32.ChecksumIEEE) of a[1:n+1], given the
// crc of a[0:n], oldLeader = a[0], and newTrailer = a[n], where n is
// the window size, and a[] is a "virtual" byte slice. It does so in
// constant time, i.e. independent of the window size.
func (w *crc32Window) update(crc uint32, oldLeader, newTrailer byte) uint32 {
	// Making update a function on *crc32Window gives a
	// substantial speedup.

	// Inline crc32.simpleUpdate to compute crcExtended.
	t := ^crc
	t = crc32.IEEETable[byte(t)^newTrailer] ^ (t >> 8)
	crcExtended := ^t

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
