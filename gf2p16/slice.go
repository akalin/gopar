package gf2p16

func mulByteSliceLEGeneric(c T, in, out []byte) {
	cEntry := c.mulTableEntry()
	for i := 0; i < len(in); i += 2 {
		cx := cEntry.s0[in[i]] ^ cEntry.s8[in[i+1]]
		out[i] = byte(cx)
		out[i+1] = byte(cx >> 8)
	}
}

func mulAndAddByteSliceLEGeneric(c T, in, out []byte) {
	cEntry := c.mulTableEntry()
	for i := 0; i < len(in); i += 2 {
		cx := cEntry.s0[in[i]] ^ cEntry.s8[in[i+1]]
		out[i] ^= byte(cx)
		out[i+1] ^= byte(cx >> 8)
	}
}

// mulSliceGeneric sets each out[i] to c.Times(in[i]).
func mulSliceGeneric(c T, in, out []T) {
	cEntry := c.mulTableEntry()
	for i := 0; i < len(in); i++ {
		out[i] = cEntry.s0[in[i]&0xff] ^ cEntry.s8[in[i]>>8]
	}
}

// mulAndAddSliceGeneric adds c.Times(in[i]) to out[i], for each i.
func mulAndAddSliceGeneric(c T, in, out []T) {
	cEntry := c.mulTableEntry()
	for i := 0; i < len(in); i++ {
		out[i] ^= cEntry.s0[in[i]&0xff] ^ cEntry.s8[in[i]>>8]
	}
}
