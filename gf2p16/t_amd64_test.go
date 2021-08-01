package gf2p16

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMulTable64(t *testing.T) {
	rand := rand.New(rand.NewSource(1))

	x := T(rand.Int())
	c := T(rand.Int())
	expectedCX := c.Times(x)

	cEntry := c.mulTable64Entry()
	cxLow := cEntry.s0Low[x&0x0f] ^ cEntry.s4Low[(x>>4)&0x0f] ^ cEntry.s8Low[(x>>8)&0x0f] ^ cEntry.s12Low[(x>>12)&0x0f]
	cxHigh := cEntry.s0High[x&0x0f] ^ cEntry.s4High[(x>>4)&0x0f] ^ cEntry.s8High[(x>>8)&0x0f] ^ cEntry.s12High[(x>>12)&0x0f]
	cx := T(cxLow) | (T(cxHigh) << 8)

	require.Equal(t, expectedCX, cx)
}
