package gf2p16

func mulSlice(c T, in, out []T) {
	if len(out) != len(in) {
		panic("size mismatch")
	}
	if len(in) == 0 {
		return
	}
	mulSliceUnsafe(&mulTable[c], in, out)
}

func mulAndAddSlice(c T, in, out []T) {
	if len(out) != len(in) {
		panic("size mismatch")
	}
	if len(in) == 0 {
		return
	}
	mulAndAddSliceUnsafe(&mulTable[c], in, out)
}

//go:noescape
func mulSliceUnsafe(cEntry *mulTableEntry, in, out []T)

//go:noescape
func mulAndAddSliceUnsafe(cEntry *mulTableEntry, in, out []T)
