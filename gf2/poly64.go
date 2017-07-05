package gf2

// A Poly64 is a polynomial over GF(2) mod x^64.
type Poly64 uint64

// Plus returns the sum of p and q as polynomials over GF(2), which is
// just the bitwise xor of the two.
func (p Poly64) Plus(q Poly64) Poly64 {
	return p ^ q
}

// Minus returns the difference of p and q as polynomials over GF(2),
// which is just the bitwise xor of the two.
func (p Poly64) Minus(q Poly64) Poly64 {
	return p ^ q
}

// Times returns the product of p and q as polynomials over GF(2), mod
// x^64.
func (p Poly64) Times(q Poly64) Poly64 {
	var prod Poly64
	for p != 0 && q != 0 {
		if q&1 != 0 {
			prod ^= p
		}
		q >>= 1
		p <<= 1
	}
	return prod
}
