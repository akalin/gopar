package gf2p16

import (
	"sync"
)

type mulTable64Entry struct {
	s0Low, s4Low, s8Low, s12Low     [1 << 4]byte
	s0High, s4High, s8High, s12High [1 << 4]byte
}

func platformPreloadCaches() {
	for i := range mulTable64Cache {
		T(i).mulTable64Entry()
	}
}

var mulTable64Cache [1 << 16]*mulTable64Entry
var mulTableCache64Mutex sync.RWMutex

func (t T) mulTable64Entry() *mulTable64Entry {
	if cachesLoaded {
		return mulTable64Cache[t]
	}
	mulTableCache64Mutex.RLock()
	entry := mulTable64Cache[t]
	if entry != nil {
		mulTableCache64Mutex.RUnlock()
		return entry
	}
	mulTableCache64Mutex.RUnlock()
	mulTableCache64Mutex.Lock()
	defer mulTableCache64Mutex.Unlock()
	if mulTable64Cache[t] == nil {
		entry := mulTable64Entry{}
		for j := 0; j < len(entry.s0Low); j++ {
			tt := t.Times(T(j))
			entry.s0Low[j] = byte(tt)
			entry.s0High[j] = byte(tt >> 8)

			tt = t.Times(T(j << 4))
			entry.s4Low[j] = byte(tt)
			entry.s4High[j] = byte(tt >> 8)

			tt = t.Times(T(j << 8))
			entry.s8Low[j] = byte(tt)
			entry.s8High[j] = byte(tt >> 8)

			tt = t.Times(T(j << 12))
			entry.s12Low[j] = byte(tt)
			entry.s12High[j] = byte(tt >> 8)
		}
		mulTable64Cache[t] = &entry
	}
	return mulTable64Cache[t]
}
