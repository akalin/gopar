package gf2p16

import (
	"reflect"
	"unsafe"
)

func castByteToTSlice(bs []byte) []T {
	h := *(*reflect.SliceHeader)(unsafe.Pointer(&bs))
	h.Len /= 2
	h.Cap /= 2
	return *(*[]T)(unsafe.Pointer(&h))
}

func mulByteSliceLEPlatformLE(c T, in, out []byte) {
	mulSlice(c, castByteToTSlice(in), castByteToTSlice(out))
}

func mulAndAddByteSliceLEPlatformLE(c T, in, out []byte) {
	mulAndAddSlice(c, castByteToTSlice(in), castByteToTSlice(out))
}
