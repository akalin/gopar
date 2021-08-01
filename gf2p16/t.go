package gf2p16

import (
	"sync"
)

//go:generate go run ./internal/gen -generator logTable -out t_logtable_gen.go
//go:generate go run ./internal/gen -generator expTable -out t_exptable_gen.go

// T is an element of GF(2^16).
type T uint16

var mulTableCache [1 << 16]*mulTableEntry
var mulTableCacheMutex sync.RWMutex

func (t T) mulTableEntry() *mulTableEntry {
	if cachesLoaded {
		return mulTableCache[t]
	}
	mulTableCacheMutex.RLock()
	entry := mulTableCache[t]
	if entry != nil {
		mulTableCacheMutex.RUnlock()
		return entry
	}
	mulTableCacheMutex.RUnlock()
	mulTableCacheMutex.Lock()
	defer mulTableCacheMutex.Unlock()
	if mulTableCache[t] == nil {
		mulTableCache[t] = &mulTableEntry{}
		for j := 0; j < 256; j++ {
			mulTableCache[t].s0[j] = t.Times(T(j))
			mulTableCache[t].s8[j] = t.Times(T(j << 8))
		}
	}
	return mulTableCache[t]
}

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

type mulTableEntry struct {
	s0, s8 [1 << 8]T
}

var cachesLoaded bool

// LoadCaches preloads mulTableCache and mulTable64Cache
func LoadCaches() {
	if cachesLoaded {
		return
	}
	for i := range mulTableCache {
		T(i).mulTableEntry()
	}
	platformPreloadCaches()
	cachesLoaded = true
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

// Pow returns the t^p as an element of GF(2^16). T(0).Pow(0) returns 1.
func (t T) Pow(p uint32) T {
	if t == 0 {
		if p == 0 {
			return 1
		}
		return 0
	}

	logT := uint64(logTable[t-1])
	return expTable[(logT*uint64(p))%(order-1)]
}
