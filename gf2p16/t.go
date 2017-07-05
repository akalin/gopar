package gf2p16

import "github.com/akalin/gopar/gf2"

// T is an element of GF(2^16).
type T uint16

// m is the generator, i.e. the irreducible polynomial of degree 16,
// used to generate GF(2^16). m was chosen to match the PAR2 spec.
const m gf2.Poly64 = 0x1100b

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

// Times returns the product of t and u as elements of GF(2^16).
//
// TODO: Use tables.
func (t T) Times(u T) T {
	_, prod := gf2.Poly64(t).Times(gf2.Poly64(u)).Div(m)
	return T(prod)
}

func (t T) inv() T {
	for i := 0; i < (1 << 16); i++ {
		if p := t.Times(T(i)); p == 1 {
			return T(i)
		}
	}

	panic("could not find inverse")
}

// Div returns the product of t and u^{-1}, if u != 0. It panics if u == 0.
//
// TODO: Use tables.
func (t T) Div(u T) T {
	if u == 0 {
		panic("division by zero")
	}

	_, quot := gf2.Poly64(t).Times(gf2.Poly64(u.inv())).Div(m)
	return T(quot)
}
