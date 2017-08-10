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

// standardToAltMapSliceSSSE3Unsafe behaves like calling
//
//   standardToAltMapSSSE3Unsafe(
//     inChunk[16:32],  inChunk[0:16],
//     outChunk[16:32], outChunk[0:16],
//   )
//
// on each subsequent pair of 32-byte chunks of in and out.
//
//go:noescape
func standardToAltMapSliceSSSE3Unsafe(in, out []byte)

// altToStandardMapSSSE3Unsafe is the inverse of
// standardToAltMapSSSE3Unsafe. That is, it sets
//
//   *out0 = inLow[0]  . inHigh[0]  . inLow[1]  . inHigh[1] . ...
//         . inLow[6]  . inHigh[6]  . inLow[7]  . inHigh[7],
//   *out1 = inLow[8]  . inHigh[8]  . inLow[9]  . inHigh[9] . ...
//         . inLow[14] . inHigh[14] . inLow[15] . inHigh[15].
//
//go:noescape
func altToStandardMapSSSE3Unsafe(inLow, inHigh, out0, out1 *[16]byte)

// altToStandardMapSliceSSSE3Unsafe behaves like calling
//
//   altToStandardMapSSSE3Unsafe(
//     inChunk[16:32],  inChunk[0:16],
//     outChunk[16:32], outChunk[0:16],
//   )
//
// on each subsequent pair of 32-byte chunks of in and out.
//
//go:noescape
func altToStandardMapSliceSSSE3Unsafe(in, out []byte)

// mulAltMapSSSE3Unsafe sets outLow and outHigh such that the
// following equation holds for each i:
//
//   (outHigh[i] << 8) | outLow[i] == c.Times((inHigh[i] << 8) | inLow[i]),
//
// where cEntry is &mulTable64[c].
//
//go:noescape
func mulAltMapSSSE3Unsafe(cEntry *mulTable64Entry, inLow, inHigh, outLow, outHigh *[16]byte)

// mulSliceAltMapSSSE3Unsafe behaves like calling
//
//   mulAltMapSSSE3Unsafe(
//     cEntry,
//     inChunk[16:32],  inChunk[0:16],
//     outChunk[16:32], outChunk[0:16],
//   )
//
// on each subsequent pair of 32-byte chunks of in and out.
//
// go:noescape
func mulSliceAltMapSSSE3Unsafe(cEntry *mulTable64Entry, in, out []byte)

// mulSSSE3Unsafe sets out0 and out1 such that the following equations
// hold for each i:
//
//   out0[2*i] | (out0[2*i+1] << 8) == c.Times(in0[2*i] | in0[2*i+1] << 8)
//   out1[2*i] | (out1[2*i+1] << 8) == c.Times(in1[2*i] | in1[2*i+1] << 8),
//
// where cEntry is &mulTable64[c].
//
//go:noescape
func mulSSSE3Unsafe(cEntry *mulTable64Entry, in0, in1, out0, out1 *[16]byte)
