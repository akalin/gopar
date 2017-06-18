package main

import (
	"fmt"
	"os"

	"github.com/akalin/gopar/par1"
)

func main() {
	decoder, err := par1.NewDecoder(os.Args[1])
	if err != nil {
		panic(err)
	}

	fmt.Printf("decoder = %+v\n", decoder)
}
