package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"

	"github.com/akalin/gopar/gf2"
)

const order = 1 << 16

var (
	logTable [order - 1]uint16
	expTable [order - 1]uint16
)

func buildTables() {
	// m is the irreducible polynomial of degree 16 used to model
	// GF(2^16). m was chosen to match the PAR2 spec.
	const m gf2.Poly64 = 0x1100b

	// g is a generator of GF(2^16).
	const g = 3

	x := uint16(1)
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
		x = uint16(r)
	}
}

func main() {
	outFile := flag.String("out", "", "output file")
	gen := flag.String("generator", "", "generator (logTable or expTable)")
	flag.Parse()
	if outFile == nil {
		log.Fatal("-out is required")
	}
	if gen == nil {
		log.Fatal("-generator is required")
	}
	buildTables()
	var buf bytes.Buffer
	switch *gen {
	case "logTable":
		genLogTable(&buf)
	case "expTable":
		genExpTable(&buf)
	default:
		log.Fatal("unknown generator:", *gen)
	}
	b, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(*outFile, b, 0600)
	if err != nil {
		log.Fatal(err)
	}
}

func mustFprintf(w io.Writer, format string, a ...interface{}) {
	_, err := fmt.Fprintf(w, format, a...)
	if err != nil {
		panic(err)
	}
}

func genExpTable(w io.Writer) {
	mustFprintf(w, `// Code generated by gf2p16/internal/gen. DO NOT EDIT.

package gf2p16

var expTable = [order - 1]T {`)

	for i, u := range expTable {
		if i%64 == 0 {
			mustFprintf(w, "\n")
		}
		mustFprintf(w, "%d, ", u)
	}
	mustFprintf(w, "\n}\n")
}

func genLogTable(w io.Writer) {
	mustFprintf(w, `// Code generated by gf2p16/internal/gen. DO NOT EDIT.

package gf2p16

var logTable = [order - 1]uint16 {`)
	for i, u := range logTable {
		if i%64 == 0 {
			mustFprintf(w, "\n")
		}
		mustFprintf(w, "%d, ", u)
	}
	mustFprintf(w, "\n}\n")
}