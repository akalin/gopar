package main

import (
	"fmt"
	"os"

	"github.com/akalin/gopar/par1"
)

type logDelegate struct{}

func (logDelegate) OnDataFileLoad(path string, err error) {
	if err != nil {
		fmt.Printf("Loading data file %q failed: %+v\n", path, err)
	} else {
		fmt.Printf("Loaded data file %q\n", path)
	}
}

func (logDelegate) OnVolumeFileLoad(path string, err error) {
	if os.IsNotExist(err) {
		// Do nothing.
	} else if err != nil {
		fmt.Printf("Loading volume file %q failed: %+v\n", path, err)
	} else {
		fmt.Printf("Loaded volume file %q\n", path)
	}
}

func main() {
	decoder, err := par1.NewDecoder(logDelegate{}, os.Args[1])
	if err != nil {
		panic(err)
	}

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
