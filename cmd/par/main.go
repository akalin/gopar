package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/akalin/gopar/par1"
)

type logEncoderDelegate struct{}

func (logEncoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Loading data file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Loaded data file %q (%d bytes)\n", i, n, path, byteCount)
	}
}

func (logEncoderDelegate) OnVolumeFileWrite(i, n int, path string, dataByteCount, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Writing volume file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Wrote volume file %q (%d data bytes, %d bytes)\n", i, n, path, dataByteCount, byteCount)
	}
}

type logDecoderDelegate struct{}

func (logDecoderDelegate) OnHeaderLoad(headerInfo string) {
	fmt.Printf("Loaded header: %s\n", headerInfo)
}

func (logDecoderDelegate) OnFileEntryLoad(i, n int, filename, entryInfo string) {
	fmt.Printf("[%d/%d] Loaded entry for %q: %s\n", i, n, filename, entryInfo)
}

func (logDecoderDelegate) OnCommentLoad(comment []byte) {
	fmt.Printf("Comment: %q\n", comment)
}

func (logDecoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, corrupt bool, err error) {
	if err != nil {
		if corrupt {
			fmt.Printf("[%d/%d] Loading data file %q failed; marking as corrupt and skipping: %+v\n", i, n, path, err)
		} else {
			fmt.Printf("[%d/%d] Loading data file %q failed: %+v\n", i, n, path, err)
		}
	} else {
		fmt.Printf("[%d/%d] Loaded data file %q (%d bytes)\n", i, n, path, byteCount)
	}
}

func (logDecoderDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Writing data file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Wrote data file %q (%d bytes)\n", i, n, path, byteCount)
	}
}

func (logDecoderDelegate) OnVolumeFileLoad(i uint64, path string, dataByteCount int, err error) {
	if os.IsNotExist(err) {
		// Do nothing.
	} else if err != nil {
		fmt.Printf("[%d] Loading volume file %q failed: %+v\n", i, path, err)
	} else {
		fmt.Printf("[%d] Loaded volume file %q (%d data bytes)\n", i, path, dataByteCount)
	}
}

func printUsageAndExit(name string, flagSet *flag.FlagSet) {
	name = filepath.Base(name)
	fmt.Printf(`
Usage:
  %s c(reate) [options] <PAR file> [files]
  %s v(erify) [options] <PAR file>
  %s r(epair) [options] <PAR file>

Options:
`, name, name, name)
	flagSet.PrintDefaults()
	fmt.Printf("\n")
	os.Exit(2)
}

func main() {
	name := os.Args[0]
	flagSet := flag.NewFlagSet(name, flag.ExitOnError)
	flagSet.SetOutput(os.Stdout)
	usage := flagSet.Bool("h", false, "print usage info")
	numParityShards := flagSet.Int("n", 3, "number of parity volumes to create")
	flagSet.Parse(os.Args[1:])

	if flagSet.NArg() < 2 || *usage {
		printUsageAndExit(name, flagSet)
	}

	cmd := flagSet.Arg(0)
	parFile := flagSet.Arg(1)

	switch strings.ToLower(cmd) {
	case "c":
		fallthrough
	case "create":
		if flagSet.NArg() == 2 {
			printUsageAndExit(name, flagSet)
		}

		encoder, err := par1.NewEncoder(logEncoderDelegate{}, flagSet.Args()[2:], *numParityShards)
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
		decoder, err := par1.NewDecoder(logDecoderDelegate{}, parFile)
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
		decoder, err := par1.NewDecoder(logDecoderDelegate{}, parFile)
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
		printUsageAndExit(name, flagSet)
	}
}
