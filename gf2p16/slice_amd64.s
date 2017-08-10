#include "textflag.h"

// func mulSliceUnsafe(cEntry *mulTableEntry, in, out []T)
TEXT ·mulSliceUnsafe(SB), NOSPLIT, $0
	MOVQ cEntry+0(FP), AX
	MOVQ in_len+16(FP), CX
	MOVQ out+32(FP), BX
	MOVQ in+8(FP), SI
	MOVQ $0, R8

loop:
	MOVWLZX (SI)(R8*2), R10     // R10 = in[i]
	MOVBLZX R10B, R11
	MOVWLZX (AX)(R11*2), R11    // R11 = cEntry.s0[in[i]&0xff]
	SHRW    $8, R10
	MOVWLZX 512(AX)(R10*2), R10 // R10 = cEntry.s8[in[i]>>8]
	XORL    R10, R11
	MOVW    R11, (BX)(R8*2)     // out[i] = R10 ^ R11
	INCQ    R8
	CMPQ    R8, CX
	JLT     $0, loop

	RET

// func mulAndAddSliceUnsafe(cEntry *mulTableEntry, in, out []T)
TEXT ·mulAndAddSliceUnsafe(SB), NOSPLIT, $0
	MOVQ cEntry+0(FP), AX
	MOVQ in_len+16(FP), CX
	MOVQ out+32(FP), BX
	MOVQ in+8(FP), SI
	MOVQ $0, R8

loop:
	MOVWLZX (BX)(R8*2), R9      // R9 = out[i]
	MOVWLZX (SI)(R8*2), R10     // R10 = in[i]
	MOVBLZX R10B, R11
	MOVWLZX (AX)(R11*2), R11    // R11 = cEntry.low[in[i]&0xff]
	SHRW    $8, R10
	MOVWLZX 512(AX)(R10*2), R10 // R10 = cEntry.high[in[i]>>8]
	XORL    R10, R11
	XORL    R11, R9
	MOVW    R9, (BX)(R8*2)      // out[i] = R9 ^ R10 ^ R11
	INCQ    R8
	CMPQ    R8, CX
	JLT     $0, loop

	RET

// All 128-bit words are written in big-endian form below.

// Sets out = 00ff00ff:00ff00ff:00ff00ff:00ff00ff, clobbering tmp and
// tmpx. out and tmpx should be 128-bit registers, i.e. beginning with X,
// and tmp should be a general purpose register, e.g. AX, BX.
#define SET_CONV_MASK_SSSE3(out, tmp, tmpx) \
	MOVQ   $0xff, tmp \
	MOVQ   tmp, out   \
	PXOR   tmpx, tmpx \
	PSHUFB tmpx, out  \
	PSRLW  $8, out

// All arguments should be 128-bit registers, i.e. beginning with X.
// convMask should be set to 00ff00ff:00ff00ff:00ff00ff:00ff00ff.
//
// Sets in0 to outHigh, and clobbers in1 and tmp.
//
// Letting each digit represent a single byte, if
//
//   in0 = 1234:5678:9abc:defg
//   in1 = hijk:lmno:pqrs:tuvw,
//
// set (first clause)
//
//   in0    = 0103:0507:090b:0d0f
//   outLow = 0204:0608:0a0c:0e0g,
//
// and (second clause)
//
//   in1 = 0h0j:0l0n:0p0r:0t0v
//   tmp = 0i0k:0m0o:0q0s:0u0w.
//
// Then set in0 to the low bytes of each nibble of in1.in0,
// and outLow to the low bytes of each nibble of tmp.outLow, where .
// denotes concatenation, i.e.
//
//   in0 = outHigh = hjln:prtv:1357:9bdf
//         outLow  = ikmo:qsuw:2468:aceg.
#define STANDARD_TO_ALT_MAP_SSSE3(in0, in1, convMask, outLow, tmp) \
	MOVO     in0, outLow      \
	PSRLW    $8, in0          \
	PAND     convMask, outLow \
	                          \
	MOVO     in1, tmp         \
	PSRLW    $8, in1          \
	PAND     convMask, tmp    \
	                          \
	PACKUSWB in1, in0         \
	PACKUSWB tmp, outLow

// func standardToAltMapSSSE3Unsafe(in0, in1, outLow, outHigh *[16]byte)
TEXT ·standardToAltMapSSSE3Unsafe(SB), NOSPLIT, $0
	// X0 = *in0
	MOVQ  in0+0(FP), AX
	MOVOU (AX), X0

	// X1 = *in1
	MOVQ  in1+8(FP), AX
	MOVOU (AX), X1

	SET_CONV_MASK_SSSE3(X4, BX, X5)
	STANDARD_TO_ALT_MAP_SSSE3(X0, X1, X4, X2, X3)

	// *outLow = X2
	MOVQ  outLow+16(FP), AX
	MOVOU X2, (AX)

	// *outHigh = X0
	MOVQ  outHigh+24(FP), AX
	MOVOU X0, (AX)

	RET

// func standardToAltMapSliceSSSE3Unsafe(in, out []byte)
TEXT ·standardToAltMapSliceSSSE3Unsafe(SB), NOSPLIT, $0
	SET_CONV_MASK_SSSE3(X4, AX, X5)

	// AX = len(in)/32
	MOVQ in_len+8(FP), AX
	SHRQ $5, AX
	CMPQ AX, $0
	JEQ  done

	// BX, CX = inChunk, outChunk = in, out
	MOVQ in+0(FP), BX
	MOVQ out+24(FP), CX

loop:
	// X0, X1 = in0, in1 = inChunk[16:32], inChunk[0:16]
	MOVOU (BX), X1
	MOVOU 16(BX), X0

	STANDARD_TO_ALT_MAP_SSSE3(X0, X1, X4, X2, X3)

	// outChunk[16:32], outChunk[0:16] = outLow, outHigh = X2, X0
	MOVOU X0, (CX)
	MOVOU X2, 16(CX)

	// inChunk += 32, outChunk += 32
	ADDQ $32, BX
	ADDQ $32, CX

	SUBQ $1, AX
	JNZ  loop

done:
	RET

// All arguments should be 128-bit registers, i.e. beginning with X.
//
// Sets inLow to out0.
//
// Letting each digit represent a single byte, if
//
//   inLow  = ikmo:qsuw:2468:aceg
//   inHigh = hjln:prtv:1357:9bdf,
//
// set out1 to be the alternating bytes of the high halves of
// out1 = inLow and inHigh, and inLow to be the alternating bytes of
// the low halves of inLow and inHigh, i.e.
//
//           out1 = hijk:lmno:pqrs:tuvw
//   inLow = out0 = 1234:5678:9abc:defg.
#define ALT_TO_STANDARD_MAP_SSSE3(inLow, inHigh, out1) \
	MOVO      inLow, out1   \
	PUNPCKHBW inHigh, out1  \
	PUNPCKLBW inHigh, inLow

// func altToStandardMapSSSE3Unsafe(inLow, inHigh, out0, out1 *[16]byte)
TEXT ·altToStandardMapSSSE3Unsafe(SB), NOSPLIT, $0
	// X0 = *inLow
	MOVQ  inLow+0(FP), AX
	MOVOU (AX), X0

	// X1 = *inHigh
	MOVQ  inHigh+8(FP), AX
	MOVOU (AX), X1

	ALT_TO_STANDARD_MAP_SSSE3(X0, X1, X2)

	// *out0 = X0
	MOVQ  out0+16(FP), AX
	MOVOU X0, (AX)

	// *out1 = X2
	MOVQ  out1+24(FP), AX
	MOVOU X2, (AX)

	RET

// func altToStandardMapSliceSSSE3Unsafe(in, out []byte)
TEXT ·altToStandardMapSliceSSSE3Unsafe(SB), NOSPLIT, $0
	// AX = len(in)/32
	MOVQ in_len+8(FP), AX
	SHRQ $5, AX
	CMPQ AX, $0
	JEQ  done

	// BX, CX = inChunk, outChunk = in, out
	MOVQ in+0(FP), BX
	MOVQ out+24(FP), CX

loop:
	// X0, X1 = inLow, inHigh = inChunk[16:32], inChunk[0:16]
	MOVOU (BX), X1
	MOVOU 16(BX), X0

	ALT_TO_STANDARD_MAP_SSSE3(X0, X1, X2)

	// outChunk[16:32], outChunk[0:16] = out0, out1 = X0, X2
	MOVOU X2, (CX)
	MOVOU X0, 16(CX)

	// inChunk += 32, outChunk += 32
	ADDQ $32, BX
	ADDQ $32, CX

	SUBQ $1, AX
	JNZ  loop

done:
	RET

// Sets out = 0f0f0f0f:0f0f0f0f:0f0f0f0f:0f0f0f0f, clobbering tmp and
// tmpx. out and tmpx should be 128-bit registers, i.e. beginning with X,
// and tmp should be a general purpose register, e.g. AX, BX.
#define SET_MUL_MASK_SSSE3(out, tmp, tmpx) \
	MOVQ   $0xf, tmp  \
	MOVQ   tmp, out   \
	PXOR   tmpx, tmpx \
	PSHUFB tmpx, out

// All arguments should be 128-bit registers, i.e. beginning with X.
// mulMask should be set to 0f0f0f0f:0f0f0f0f:0f0f0f0f:0f0f0f0f.
//
// Letting a[i] mean the ith byte of a, for each i, sets
//
//   out[i] = s0[inLow[i] & 0f] ^ s4[(inLow[i] >> 4) & 0f] ^
//     s8[inHigh[i] & 0f] ^ s12[(inHigh[i] >> 4) & 0f],
//
// and clobbers tmp0 and tmp1.
#define MUL_ALT_MAP_SSSE3_BYTE(s0, s4, s8, s12, inLow, inHigh, mulMask, out, tmp0, tmp1) \
	MOVO   inLow, tmp0   \
	PAND   mulMask, tmp0 \
	MOVO   s0, out       \
	PSHUFB tmp0, out     \
	                     \
	MOVO   inLow, tmp0   \
	PSRLW  $4, tmp0      \
	PAND   mulMask, tmp0 \
	MOVO   s4, tmp1      \
	PSHUFB tmp0, tmp1    \
	PXOR   tmp1, out     \
	                     \
	MOVO   inHigh, tmp0  \
	PAND   mulMask, tmp0 \
	MOVO   s8, tmp1      \
	PSHUFB tmp0, tmp1    \
	PXOR   tmp1, out     \
	                     \
	MOVO   inHigh, tmp0  \
	PSRLW  $4, tmp0      \
	PAND   mulMask, tmp0 \
	MOVO   s12, tmp1     \
	PSHUFB tmp0, tmp1    \
	PXOR   tmp1, out

// All arguments should be 128-bit registers, i.e. beginning with X.
// mulMask should be set to 0f0f0f0f:0f0f0f0f:0f0f0f0f:0f0f0f0f.
//
// Letting a[i] mean the ith byte of a, sets outLow, outHigh such
// that the following equation holds for each i:
//
//   (outHigh[i] << 8) | outLow[i] == c.Times((inHigh[i] << 8) | inLow[i]),
//
// where s{0,4,8,12}{Low,High} are the fields of &mulTable64[c],
// and clobbers tmp0 and tmp1.
#define MUL_ALT_MAP_SSSE3(s0Low, s4Low, s8Low, s12Low, s0High, s4High, s8High, s12High, inLow, inHigh, mulMask, outLow, outHigh, tmp0, tmp1) \
	MUL_ALT_MAP_SSSE3_BYTE(s0Low, s4Low, s8Low, s12Low, inLow, inHigh, mulMask, outLow, tmp0, tmp1)      \
	MUL_ALT_MAP_SSSE3_BYTE(s0High, s4High, s8High, s12High, inLow, inHigh, mulMask, outHigh, tmp0, tmp1)

// func mulAltMapSSSE3Unsafe(cEntry *mulTable64Entry, inLow, inHigh, outLow, outHigh *[16]byte)
TEXT ·mulAltMapSSSE3Unsafe(SB), NOSPLIT, $0
	// Set X8 - X15 to input tables.
	MOVQ  cEntry+0(FP), AX
	MOVOU (AX), X8         // X8  = cEntry.s0Low
	MOVOU 16(AX), X9       // X9  = cEntry.s4Low
	MOVOU 32(AX), X10      // X10 = cEntry.s8Low
	MOVOU 48(AX), X11      // X11 = cEntry.s12Low
	MOVOU 64(AX), X12      // X12 = cEntry.s0High
	MOVOU 80(AX), X13      // X13 = cEntry.s4High
	MOVOU 96(AX), X14      // X14 = cEntry.s8High
	MOVOU 112(AX), X15     // X15 = cEntry.s12High

	// X0 = *inLow
	MOVQ  inLow+8(FP), AX
	MOVOU (AX), X0

	// X1 = *inHigh
	MOVQ  inHigh+16(FP), AX
	MOVOU (AX), X1

	SET_MUL_MASK_SSSE3(X7, AX, X2)
	MUL_ALT_MAP_SSSE3(X8, X9, X10, X11, X12, X13, X14, X15, X0, X1, X7, X2, X3, X4, X5)

	// *outLow = X2
	MOVQ  outLow+24(FP), AX
	MOVOU X2, (AX)

	// *outHigh = X3
	MOVQ  outHigh+32(FP), AX
	MOVOU X3, (AX)

	RET

// func mulSliceAltMapSSSE3Unsafe(cEntry *mulTable64Entry, in, out []byte)
TEXT ·mulSliceAltMapSSSE3Unsafe(SB), NOSPLIT, $0
	// Set X8 - X15 to input tables.
	MOVQ  cEntry+0(FP), AX
	MOVOU (AX), X8         // X8  = cEntry.s0Low
	MOVOU 16(AX), X9       // X9  = cEntry.s4Low
	MOVOU 32(AX), X10      // X10 = cEntry.s8Low
	MOVOU 48(AX), X11      // X11 = cEntry.s12Low
	MOVOU 64(AX), X12      // X12 = cEntry.s0High
	MOVOU 80(AX), X13      // X13 = cEntry.s4High
	MOVOU 96(AX), X14      // X14 = cEntry.s8High
	MOVOU 112(AX), X15     // X15 = cEntry.s12High

	SET_MUL_MASK_SSSE3(X7, AX, X2)

	// AX = len(in)/32
	MOVQ in_len+16(FP), AX
	SHRQ $5, AX
	CMPQ AX, $0
	JEQ  done

	// BX, CX = inChunk, outChunk = in, out
	MOVQ in+8(FP), BX
	MOVQ out+32(FP), CX

loop:
	// X0, X1 = inLow, inHigh = inChunk[16:32], inChunk[0:16]
	MOVOU (BX), X1
	MOVOU 16(BX), X0

	MUL_ALT_MAP_SSSE3(X8, X9, X10, X11, X12, X13, X14, X15, X0, X1, X7, X2, X3, X4, X5)

	// outChunk[16:32], outChunk[0:16] = outLow, outHigh = X2, X3
	MOVOU X3, (CX)
	MOVOU X2, 16(CX)

	// inChunk += 32, outChunk += 32
	ADDQ $32, BX
	ADDQ $32, CX

	SUBQ $1, AX
	JNZ  loop

done:
	RET

// All arguments should be 128-bit registers, i.e. beginning with X.
// convMask should be set to 00ff00ff:00ff00ff:00ff00ff:00ff00ff, and
// mulMask should be set to 0f0f0f0f:0f0f0f0f:0f0f0f0f:0f0f0f0f.
//
// Letting a[i] mean the ith byte of a, sets in1, in0 to out0, out1 such
// that the following equations hold for each i:
//
//   out0[2*i] | (out0[2*i+1] << 8) == c.Times(in0[2*i] | in0[2*i+1] << 8)
//   out1[2*i] | (out1[2*i+1] << 8) == c.Times(in1[2*i] | in1[2*i+1] << 8),
//
// and clobbers tmp0, tmp1, tmp2, tmp3.
#define MUL_STANDARD_MAP_SSSE3(s0Low, s4Low, s8Low, s12Low, s0High, s4High, s8High, s12High, in0, in1, convMask, mulMask, tmp0, tmp1, tmp2, tmp3) \
	STANDARD_TO_ALT_MAP_SSSE3(in0, in1, convMask, tmp0, tmp1)                                                                  \
	MUL_ALT_MAP_SSSE3(s0Low, s4Low, s8Low, s12Low, s0High, s4High, s8High, s12High, tmp0, in0, mulMask, in1, tmp1, tmp2, tmp3) \
	ALT_TO_STANDARD_MAP_SSSE3(in1, tmp1, in0)

// func mulSSSE3Unsafe(cEntry *mulTable64Entry, in0, in1, out0, out1 *[16]byte)
TEXT ·mulSSSE3Unsafe(SB), NOSPLIT, $0
	// Set X8 - X15 to input tables.
	MOVQ  cEntry+0(FP), AX
	MOVOU (AX), X8         // X8  = cEntry.s0Low
	MOVOU 16(AX), X9       // X9  = cEntry.s4Low
	MOVOU 32(AX), X10      // X10 = cEntry.s8Low
	MOVOU 48(AX), X11      // X11 = cEntry.s12Low
	MOVOU 64(AX), X12      // X12 = cEntry.s0High
	MOVOU 80(AX), X13      // X13 = cEntry.s4High
	MOVOU 96(AX), X14      // X14 = cEntry.s8High
	MOVOU 112(AX), X15     // X15 = cEntry.s12High

	// X0 = *in0
	MOVQ  in0+8(FP), AX
	MOVOU (AX), X0

	// X1 = *in1
	MOVQ  in1+16(FP), AX
	MOVOU (AX), X1

	SET_MUL_MASK_SSSE3(X7, AX, X2)
	SET_CONV_MASK_SSSE3(X6, AX, X2)

	MUL_STANDARD_MAP_SSSE3(X8, X9, X10, X11, X12, X13, X14, X15, X0, X1, X6, X7, X2, X3, X4, X5)

	// *out0 = X1
	MOVQ  out0+24(FP), AX
	MOVOU X1, (AX)

	// *out1 = X0
	MOVQ  out1+32(FP), AX
	MOVOU X0, (AX)

	RET

// func mulSliceSSSE3Unsafe(cEntry *mulTable64Entry, in, out []byte)
TEXT ·mulSliceSSSE3Unsafe(SB), NOSPLIT, $0
	// Set X8 - X15 to input tables.
	MOVQ  cEntry+0(FP), AX
	MOVOU (AX), X8         // X8  = cEntry.s0Low
	MOVOU 16(AX), X9       // X9  = cEntry.s4Low
	MOVOU 32(AX), X10      // X10 = cEntry.s8Low
	MOVOU 48(AX), X11      // X11 = cEntry.s12Low
	MOVOU 64(AX), X12      // X12 = cEntry.s0High
	MOVOU 80(AX), X13      // X13 = cEntry.s4High
	MOVOU 96(AX), X14      // X14 = cEntry.s8High
	MOVOU 112(AX), X15     // X15 = cEntry.s12High

	SET_MUL_MASK_SSSE3(X7, AX, X2)
	SET_CONV_MASK_SSSE3(X6, AX, X2)

	// AX = len(in)/32
	MOVQ in_len+16(FP), AX
	SHRQ $5, AX
	CMPQ AX, $0
	JEQ  done

	// BX, CX = inChunk, outChunk = in, out
	MOVQ in+8(FP), BX
	MOVQ out+32(FP), CX

loop:
	// X0, X1 = in0, in1 = inChunk[16:32], inChunk[0:16]
	MOVOU (BX), X1
	MOVOU 16(BX), X0

	MUL_STANDARD_MAP_SSSE3(X8, X9, X10, X11, X12, X13, X14, X15, X0, X1, X6, X7, X2, X3, X4, X5)

	// outChunk[16:32], outChunk[0:16] = out0, out1 = X1, X0
	MOVOU X0, (CX)
	MOVOU X1, 16(CX)

	// inChunk += 32, outChunk += 32
	ADDQ $32, BX
	ADDQ $32, CX

	SUBQ $1, AX
	JNZ  loop

done:
	RET
