package main

import (
	"os"

	"github.com/akalin/gopar/libgopar"
)

func main() {
	os.Exit(libgopar.Par(os.Args,false))
}