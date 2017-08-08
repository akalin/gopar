package gf2p16

func mulSlice(c T, in, out []T) {
	if len(out) != len(in) {
		panic("size mismatch")
	}
	if len(in) == 0 {
		return
	}
	mulSliceUnsafe(&mulTable[c], in, out)
}

func mulAndAddSlice(c T, in, out []T) {
	if len(out) != len(in) {
		panic("size mismatch")
	}
	if len(in) == 0 {
		return
	}
	mulAndAddSliceUnsafe(&mulTable[c], in, out)
}

//go:noescape
func mulSliceUnsafe(cEntry *mulTableEntry, in, out []T)

//go:noescape
func mulAndAddSliceUnsafe(cEntry *mulTableEntry, in, out []T)

// standardToAltMapSSSE3Unsafe sets
//
//   *outLow  = in0[0,2,4,6,8,10,12,14] . in1[0,2,4,6,8,10,12,14]
//   *outHigh = in0[1,3,5,7,9,11,13,15] . in1[1,3,5,7,9,11,13,15],
//
// where . denotes concatenation.
//
//go:noescape
func standardToAltMapSSSE3Unsafe(in0, in1, outLow, outHigh *[16]byte)
