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

// func standardToAltMapSSSE3Unsafe(in0, in1, outLow, outHigh *[16]byte)
TEXT ·standardToAltMapSSSE3Unsafe(SB), NOSPLIT, $0
	// X0 = *in0
	MOVQ  in0+0(FP), AX
	MOVOU (AX), X0

	// X1 = *in1
	MOVQ  in1+8(FP), AX
	MOVOU (AX), X1

	// All 128-bit words are written in big-endian form below.

	// Set X4 = 00ff00ff:00ff00ff:00ff00ff:00ff00ff, clobbering X5.
	MOVQ   $0xff, BX
	MOVQ   BX, X4
	PXOR   X5, X5
	PSHUFB X5, X4
	PSRLW  $8, X4

	// Letting each digit represent a single byte, if
	//
	//   *in0 = X0 = 1234:5678:9abc:defg
	//   *in1 = X1 = hijk:lmno:pqrs:tuvw,
	//
	// set
	//
	//   X0 = 0103:0507:090b:0d0f
	//   X2 = 0204:0608:0a0c:0e0g,
	MOVO  X0, X2
	PSRLW $8, X0
	PAND  X4, X2

	// and
	//
	//   X1 = 0h0j:0l0n:0p0r:0t0v
	//   X3 = 0i0k:0m0o:0q0s:0u0w.
	MOVO  X1, X3
	PSRLW $8, X1
	PAND  X4, X3

	// Then set X0 to the low bytes of each nibble of X1.X0,
	// and X2 to the low bytes of each nibble of X3.X2, where .
	// denotes concatenation, i.e.
	//
	//   X0 = hjln:prtv:1357:9bdf
	//   X2 = ikmo:qsuw:2468:aceg.
	PACKUSWB X1, X0
	PACKUSWB X3, X2

	// *outLow = X2
	MOVQ  outLow+16(FP), AX
	MOVOU X2, (AX)

	// *outHigh = X0
	MOVQ  outHigh+24(FP), AX
	MOVOU X0, (AX)

	RET
