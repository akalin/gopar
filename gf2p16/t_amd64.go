package gf2p16

type mulTable64Entry struct {
	s0Low, s4Low, s8Low, s12Low     [1 << 4]byte
	s0High, s4High, s8High, s12High [1 << 4]byte
}

var mulTable64 [1 << 16]mulTable64Entry

func platformInit() {
	for i := 0; i < len(mulTable64); i++ {
		for j := 0; j < len(mulTable64[i].s0Low); j++ {
			t0 := T(i).Times(T(j))
			mulTable64[i].s0Low[j] = byte(t0)
			mulTable64[i].s0High[j] = byte(t0 >> 8)

			t1 := T(i).Times(T(j << 4))
			mulTable64[i].s4Low[j] = byte(t1)
			mulTable64[i].s4High[j] = byte(t1 >> 8)

			t2 := T(i).Times(T(j << 8))
			mulTable64[i].s8Low[j] = byte(t2)
			mulTable64[i].s8High[j] = byte(t2 >> 8)

			t3 := T(i).Times(T(j << 12))
			mulTable64[i].s12Low[j] = byte(t3)
			mulTable64[i].s12High[j] = byte(t3 >> 8)
		}
	}
}
