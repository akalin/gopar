package gf2p16

// MulSlice sets each out[i] to c.Times(in[i]), treating in and out like []T.
func MulSlice(c T, in, out []uint16) {
	for i, x := range in {
		out[i] = uint16(c.Times(T(x)))
	}
}

// MulAndAddSlice adds c.Times(in[i]) to out[i], for each i, treating
// in and out like []T.
func MulAndAddSlice(c T, in, out []uint16) {
	for i, x := range in {
		out[i] ^= uint16(c.Times(T(x)))
	}
}
