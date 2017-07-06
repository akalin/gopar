package gf2p16

import "github.com/akalin/gopar/gf2"

// T is an element of GF(2^16).
type T uint16

// Plus returns the sum of t and u as elements of GF(2^16), which is
// just the bitwise xor of the two.
func (t T) Plus(u T) T {
	return t ^ u
}

// Minus returns the difference of t and u as elements of GF(2^16),
// which is just the bitwise xor of the two.
func (t T) Minus(u T) T {
	return t ^ u
}

const order = 1 << 16

var logTable [order - 1]uint16
var expTable [order - 1]T

func init() {
	// TODO: Generate tables at compile time.

	// m is the irreducible polynomial of degree 16 used to model
	// GF(2^16). m was chosen to match the PAR2 spec.
	const m gf2.Poly64 = 0x1100b

	// g is a generator of GF(2^16).
	const g T = 3

	x := T(1)
	for p := 0; p < order-1; p++ {
		if x == 1 && p != 0 {
			panic("repeated power (1)")
		} else if x != 1 && logTable[x-1] != 0 {
			panic("repeated power")
		}
		if expTable[p] != 0 {
			panic("repeated exponent")
		}

		logTable[x-1] = uint16(p)
		expTable[p] = x
		_, r := gf2.Poly64(x).Times(gf2.Poly64(g)).Div(m)
		x = T(r)
	}
}

// Times returns the product of t and u as elements of GF(2^16).
func (t T) Times(u T) T {
	if t == 0 || u == 0 {
		return 0
	}

	logT := int(logTable[t-1])
	logU := int(logTable[u-1])
	return expTable[(logT+logU)%(order-1)]
}

// Inverse returns the multiplicative inverse of t, if t != 0. It
// panics if t == 0.
func (t T) Inverse() T {
	if t == 0 {
		panic("zero has no inverse")
	}
	logT := int(logTable[t-1])
	return expTable[(-logT+(order-1))%(order-1)]
}

// Div returns the product of t and u^{-1} as elements of GF(2^16), if
// u != 0. It panics if u == 0.
func (t T) Div(u T) T {
	if u == 0 {
		panic("division by zero")
	}

	if t == 0 {
		return 0
	}

	logT := int(logTable[t-1])
	logU := int(logTable[u-1])
	return expTable[(logT-logU+(order-1))%(order-1)]
}
