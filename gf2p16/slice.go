package gf2p16

func mulByteSliceLEGeneric(c T, in, out []byte) {
	cEntry := &mulTable[c]
	for i := 0; i < len(in); i += 2 {
		cx := cEntry.s0[in[i]] ^ cEntry.s8[in[i+1]]
		out[i] = byte(cx)
		out[i+1] = byte(cx >> 8)
	}
}

func mulAndAddByteSliceLEGeneric(c T, in, out []byte) {
	cEntry := &mulTable[c]
	for i := 0; i < len(in); i += 2 {
		cx := cEntry.s0[in[i]] ^ cEntry.s8[in[i+1]]
		out[i] ^= byte(cx)
		out[i+1] ^= byte(cx >> 8)
	}
}

// MulByteSliceLE treats in and out as arrays of Ts stored in
// little-endian format, and sets each out<T>[i] to c.Times(in<T>[i]).
func MulByteSliceLE(c T, in, out []byte) {
	if platformLittleEndian {
		mulByteSliceLEPlatformLE(c, in, out)
	} else {
		mulByteSliceLEGeneric(c, in, out)
	}
}

// MulAndAddByteSliceLE treats in and out as arrays of Ts stored in
// little-endian format, and adds c.Times(in<T>[i]) to out<T>[i], for
// each i.
func MulAndAddByteSliceLE(c T, in, out []byte) {
	if platformLittleEndian {
		mulAndAddByteSliceLEPlatformLE(c, in, out)
	} else {
		mulAndAddByteSliceLEGeneric(c, in, out)
	}
}

// mulSliceGeneric sets each out[i] to c.Times(in[i]).
func mulSliceGeneric(c T, in, out []T) {
	cEntry := &mulTable[c]
	for i := 0; i < len(in); i++ {
		out[i] = cEntry.s0[in[i]&0xff] ^ cEntry.s8[in[i]>>8]
	}
}

// mulAndAddSliceGeneric adds c.Times(in[i]) to out[i], for each i.
func mulAndAddSliceGeneric(c T, in, out []T) {
	cEntry := &mulTable[c]
	for i := 0; i < len(in); i++ {
		out[i] ^= cEntry.s0[in[i]&0xff] ^ cEntry.s8[in[i]>>8]
	}
}
