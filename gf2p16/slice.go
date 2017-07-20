package gf2p16

// MulByteSliceLE treats in and out as arrays of Ts stored in
// little-endian format, and sets each out<T>[i] to c.Times(in<T>[i]).
func MulByteSliceLE(c T, in, out []byte) {
	for i := 0; i < len(in); i += 2 {
		x := T(in[i]) + (T(in[i+1]) << 8)
		cx := c.Times(x)
		out[i] = byte(cx)
		out[i+1] = byte(cx >> 8)
	}
}

// MulAndAddByteSliceLE treats in and out as arrays of Ts stored in
// little-endian format, and adds c.Times(in<T>[i]) to out<T>[i], for
// each i.
func MulAndAddByteSliceLE(c T, in, out []byte) {
	for i := 0; i < len(in); i += 2 {
		x := T(in[i]) + (T(in[i+1]) << 8)
		cx := c.Times(x)
		out[i] ^= byte(cx)
		out[i+1] ^= byte(cx >> 8)
	}
}
