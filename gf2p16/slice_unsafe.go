package gf2p16

import (
	"reflect"
	"unsafe"
)

func castByteToTSlice(bs []byte) []T {
	bsHdr := (*reflect.SliceHeader)(unsafe.Pointer(&bs))
	p := unsafe.Pointer(bsHdr.Data)

	var ts []T
	tsHdr := (*reflect.SliceHeader)(unsafe.Pointer(&ts))
	tsHdr.Data = uintptr(p)
	tsHdr.Len = bsHdr.Len / 2
	tsHdr.Cap = bsHdr.Cap / 2
	return ts
}

func mulByteSliceLEPlatformLE(c T, in, out []byte) {
	mulSlice(c, castByteToTSlice(in), castByteToTSlice(out))
}

func mulAndAddByteSliceLEPlatformLE(c T, in, out []byte) {
	mulAndAddSlice(c, castByteToTSlice(in), castByteToTSlice(out))
}
