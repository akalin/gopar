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
	MOVWLZX (AX)(R11*2), R11    // R11 = cEntry.low[in[i]&0xff]
	SHRW    $8, R10
	MOVWLZX 512(AX)(R10*2), R10 // R10 = cEntry.high[in[i]>>8]
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
