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
