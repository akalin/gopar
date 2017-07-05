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

// ilog2(n) returns floor(log2(n)), assuming n > 0.
func ilog2(n uint64) uint {
	var r uint
	for {
		n >>= 1
		if n == 0 {
			break
		}
		r++
	}
	return r
}

// Div performs Euclidean division using p and p2 (if non-zero) and
// returns the resulting quotient and remainder. That is, it returns q
// and r such that q.Times(p2).Plus(r) == p and either r == 0 or
// floor(log2(r)) < floor(log2(p2)). It panics if p2 == 0.
func (p Poly64) Div(p2 Poly64) (q, r Poly64) {
	if p2 == 0 {
		panic("division by zero")
	}

	q = 0
	r = p
	log2p2 := ilog2(uint64(p2))
	for r != 0 {
		log2r := ilog2(uint64(r))
		if log2r < log2p2 {
			break
		}
		dlog2 := log2r - log2p2
		q ^= (1 << dlog2)
		r ^= (p2 << dlog2)
	}

	return q, r
}
