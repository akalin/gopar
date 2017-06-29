package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func printUsageAndExit(name string) {
	name = filepath.Base(name)
	fmt.Printf(`
Usage:
  %s c(reate) <PAR file> [files]
  %s v(erify) <PAR file>
  %s r(epair) <PAR file>

`, name, name, name)
	os.Exit(-1)
}

func main() {
	name := os.Args[0]
	if len(os.Args) <= 2 {
		printUsageAndExit(name)
	}

	cmd := os.Args[1]
	parFile := os.Args[2]

	switch strings.ToLower(cmd) {
	case "c":
		fallthrough
	case "create":
		if len(os.Args) == 2 {
			printUsageAndExit(name)
		}

		// TODO: Add option to set number of parity volumes.
		encoder, err := par1.NewEncoder(os.Args[3:], 3)
		if err != nil {
			panic(err)
		}

		err = encoder.LoadFileData()
		if err != nil {
			panic(err)
		}

		err = encoder.ComputeParityData()
		if err != nil {
			panic(err)
		}

		err = encoder.Write(parFile)
		if err != nil {
			fmt.Printf("Write parity error: %s", err)
			os.Exit(-1)
		}

	case "v":
		fallthrough
	case "verify":
		decoder, err := par1.NewDecoder(logDelegate{}, parFile)
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
		if !ok {
			os.Exit(-1)
		}

	case "r":
		fallthrough
	case "repair":
		decoder, err := par1.NewDecoder(logDelegate{}, parFile)
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

		repairedFiles, err := decoder.Repair()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Repair error: %s\n", err)
			os.Exit(-1)
		}

		fmt.Printf("Repaired files: %v\n", repairedFiles)
		if err != nil {
			os.Exit(-1)
		}

	default:
		printUsageAndExit(name)
	}
}
