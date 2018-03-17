package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime/pprof"
	"strings"

	"github.com/akalin/gopar/par1"
	"github.com/akalin/gopar/par2"
	"github.com/akalin/gopar/rsec16"
)

type par1LogEncoderDelegate struct{}

func (par1LogEncoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Loading data file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Loaded data file %q (%d bytes)\n", i, n, path, byteCount)
	}
}

func (par1LogEncoderDelegate) OnVolumeFileWrite(i, n int, path string, dataByteCount, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Writing volume file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Wrote volume file %q (%d data bytes, %d bytes)\n", i, n, path, dataByteCount, byteCount)
	}
}

type par1LogDecoderDelegate struct{}

func (par1LogDecoderDelegate) OnHeaderLoad(headerInfo string) {
	fmt.Printf("Loaded header: %s\n", headerInfo)
}

func (par1LogDecoderDelegate) OnFileEntryLoad(i, n int, filename, entryInfo string) {
	fmt.Printf("[%d/%d] Loaded entry for %q: %s\n", i, n, filename, entryInfo)
}

func (par1LogDecoderDelegate) OnCommentLoad(comment []byte) {
	fmt.Printf("Comment: %q\n", comment)
}

func (par1LogDecoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, corrupt bool, err error) {
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

func (par1LogDecoderDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Writing data file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Wrote data file %q (%d bytes)\n", i, n, path, byteCount)
	}
}

func (par1LogDecoderDelegate) OnVolumeFileLoad(i uint64, path string, storedSetHash, computedSetHash [16]byte, dataByteCount int, err error) {
	if os.IsNotExist(err) {
		// Do nothing.
	} else if err != nil {
		fmt.Printf("[%d] Loading volume file %q failed: %+v\n", i, path, err)
	} else {
		fmt.Printf("[%d] Loaded volume file %q (%d data bytes)\n", i, path, dataByteCount)
		if storedSetHash != computedSetHash {
			fmt.Printf("[%d] Warning: stored set hash in %q %x doesn't match computed set hash %x\n", i, path, storedSetHash, computedSetHash)
		}
	}
}

type par2LogEncoderDelegate struct{}

func (par2LogEncoderDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Loading data file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Loaded data file %q (%d bytes)\n", i, n, path, byteCount)
	}
}

func (par2LogEncoderDelegate) OnIndexFileWrite(path string, byteCount int, err error) {
	if err != nil {
		fmt.Printf("Writing index file %q failed: %+v\n", path, err)
	} else {
		fmt.Printf("Wrote index file %q (%d bytes)\n", path, byteCount)
	}
}

func (par2LogEncoderDelegate) OnRecoveryFileWrite(start, count, total int, path string, dataByteCount, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d+%d/%d] Writing recovery file %q failed: %+v\n", start, count, total, path, err)
	} else {
		fmt.Printf("[%d+%d/%d] Wrote recovery file %q (%d data bytes, %d bytes)\n", start, count, total, path, dataByteCount, byteCount)
	}
}

type par2LogDecoderDelegate struct{}

func (par2LogDecoderDelegate) OnCreatorPacketLoad(clientID string) {
	fmt.Printf("Loaded creator packet with client ID %q\n", clientID)
}

func (par2LogDecoderDelegate) OnMainPacketLoad(sliceByteCount, recoverySetCount, nonRecoverySetCount int) {
	fmt.Printf("Loaded main packet: slice byte count=%d, recovery set size=%d, non-recovery set size=%d\n", sliceByteCount, recoverySetCount, nonRecoverySetCount)
}

func (par2LogDecoderDelegate) OnFileDescriptionPacketLoad(fileID [16]byte, filename string, byteCount int) {
	fmt.Printf("Loaded file description packet for %q (ID=%x, %d bytes)\n", filename, fileID, byteCount)
}

func (par2LogDecoderDelegate) OnIFSCPacketLoad(fileID [16]byte) {
	fmt.Printf("Loaded checksums for file with ID %x\n", fileID)
}

func (par2LogDecoderDelegate) OnRecoveryPacketLoad(exponent uint16, byteCount int) {
	fmt.Printf("Loaded recovery packet: exponent=%d, byte count=%d\n", exponent, byteCount)
}

func (par2LogDecoderDelegate) OnUnknownPacketLoad(packetType [16]byte, byteCount int) {
	fmt.Printf("Loaded unknown packet of type %q and byte count %d\n", packetType, byteCount)
}

func (par2LogDecoderDelegate) OnOtherPacketSkip(setID [16]byte, packetType [16]byte, byteCount int) {
	fmt.Printf("Skipped packet with set ID %x of type %q and byte count %d\n", setID, packetType, byteCount)
}

func (par2LogDecoderDelegate) OnDataFileLoad(i, n int, path string, byteCount, hits, misses int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Loading data file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Loaded data file %q (%d bytes, %d hits, %d misses)\n", i, n, path, byteCount, hits, misses)
	}
}

func (par2LogDecoderDelegate) OnParityFileLoad(i int, path string, err error) {
	if err != nil {
		fmt.Printf("[%d] Loading volume file %q failed: %+v\n", i, path, err)
	} else {
		fmt.Printf("[%d] Loaded volume file %q\n", i, path)
	}
}

func (par2LogDecoderDelegate) OnDetectCorruptDataChunk(fileID [16]byte, filename string, startByteOffset, endByteOffset int) {
	fmt.Printf("Corrupt data chunk: %q (ID %x), bytes %d to %d\n", filename, fileID, startByteOffset, endByteOffset-1)
}

func (par2LogDecoderDelegate) OnDetectDataFileHashMismatch(fileID [16]byte, filename string) {
	fmt.Printf("Hash mismatch for %q (ID %x)\n", filename, fileID)
}

func (par2LogDecoderDelegate) OnDetectDataFileWrongByteCount(fileID [16]byte, filename string) {
	fmt.Printf("Wrong byte count for %q (ID %x)\n", filename, fileID)
}

func (par2LogDecoderDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Writing data file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Wrote data file %q (%d bytes)\n", i, n, path, byteCount)
	}
}

func printUsageAndExit(name string, flagSet *flag.FlagSet) {
	name = filepath.Base(name)
	fmt.Printf(`
Usage:
  %s [options] c(reate) <PAR file> [files]
  %s [options] v(erify) <PAR file>
  %s [options] r(epair) <PAR file>

Options:
`, name, name, name)
	flagSet.PrintDefaults()
	fmt.Printf("\n")
	os.Exit(2)
}

type encoder interface {
	LoadFileData() error
	ComputeParityData() error
	Write(string) error
}

type decoder interface {
	LoadFileData() error
	LoadParityData() error
	Verify(checkParity bool) (bool, error)
	Repair(checkParity bool) ([]string, error)
}

func newEncoder(parFile string, filePaths []string, sliceByteCount, numParityShards, numGoroutines int) (encoder, error) {
	// TODO: Detect file type more robustly.
	ext := path.Ext(parFile)
	if ext == ".par2" {
		return par2.NewEncoder(par2LogEncoderDelegate{}, filePaths, sliceByteCount, numParityShards, numGoroutines)
	}
	return par1.NewEncoder(par1LogEncoderDelegate{}, filePaths, numParityShards)
}

func newDecoder(parFile string, numGoroutines int) (decoder, error) {
	// TODO: Detect file type more robustly.
	ext := path.Ext(parFile)
	if ext == ".par2" {
		return par2.NewDecoder(par2LogDecoderDelegate{}, parFile, numGoroutines)
	}
	return par1.NewDecoder(par1LogDecoderDelegate{}, parFile)
}

func main() {
	name := os.Args[0]
	flagSet := flag.NewFlagSet(name, flag.ExitOnError)
	flagSet.SetOutput(os.Stdout)
	usage := flagSet.Bool("h", false, "print usage info")
	cpuprofile := flagSet.String("cpuprofile", "", "if non-empty, where to write the CPU profile")
	checkParity := flagSet.Bool("checkparity", false, "check parity when verifying or repairing")
	sliceByteCount := flagSet.Int("s", 2000, "block size in bytes (must be a multiple of 4)")
	numParityShards := flagSet.Int("c", 3, "number of recovery blocks to create (or files, for PAR1)")
	// TODO: Detect hyperthreading and use only number of physical cores.
	numGoroutines := flagSet.Int("g", rsec16.DefaultNumGoroutines(), "number of goroutines to use for encoding/decoding PAR2")
	flagSet.Parse(os.Args[1:])

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		err = pprof.StartCPUProfile(f)
		if err != nil {
			panic(err)
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			<-c
			pprof.StopCPUProfile()
			os.Exit(1)
		}()

		defer pprof.StopCPUProfile()
	}

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

		encoder, err := newEncoder(parFile, flagSet.Args()[2:], *sliceByteCount, *numParityShards, *numGoroutines)
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
		decoder, err := newDecoder(parFile, *numGoroutines)
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

		ok, err := decoder.Verify(*checkParity)
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
		decoder, err := newDecoder(parFile, *numGoroutines)
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

		repairedFiles, err := decoder.Repair(*checkParity)
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
