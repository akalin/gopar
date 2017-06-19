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

	err = decoder.LoadFileData()
	if err != nil {
		panic(err)
	}

	err = decoder.LoadParityData()
	if err != nil {
		panic(err)
	}

	ok, err := decoder.Verify()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Verify result: %t\n", ok)

	repairedFiles, err := decoder.Repair()
	if err != nil {
		fmt.Printf("Repair error: %s\n", err)
	}

	fmt.Printf("Repaired files: %v\n", repairedFiles)
}
