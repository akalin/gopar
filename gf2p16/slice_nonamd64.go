// +build !amd64

package gf2p16

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

func mulSlice(c T, in, out []T) {
	mulSliceGeneric(c, in, out)
}

func mulAndAddSlice(c T, in, out []T) {
	mulAndAddSliceGeneric(c, in, out)
}
