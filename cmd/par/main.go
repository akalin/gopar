package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime/pprof"
	"strings"

	"github.com/akalin/gopar/par1"
	"github.com/akalin/gopar/par2"
	"github.com/akalin/gopar/par2cmdline"
	"github.com/akalin/gopar/rsec16"
)

type par1LogCreateDelegate struct{}

func (par1LogCreateDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Loading data file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Loaded data file %q (%d bytes)\n", i, n, path, byteCount)
	}
}

func (par1LogCreateDelegate) OnVolumeFileWrite(i, n int, path string, dataByteCount, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Writing volume file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Wrote volume file %q (%d data bytes, %d bytes)\n", i, n, path, dataByteCount, byteCount)
	}
}

func (par1LogCreateDelegate) OnFilesNotAllInSameDir() {
	fmt.Printf("Warning: PAR and data files not all in the same directory, which a decoder will expect\n")
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

type par1LogVerifyDelegate struct {
	par1LogDecoderDelegate
}

type par2LogCreateDelegate struct{}

func (par2LogCreateDelegate) OnDataFileLoad(i, n int, path string, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Loading data file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Loaded data file %q (%d bytes)\n", i, n, path, byteCount)
	}
}

func (par2LogCreateDelegate) OnIndexFileWrite(path string, byteCount int, err error) {
	if err != nil {
		fmt.Printf("Writing index file %q failed: %+v\n", path, err)
	} else {
		fmt.Printf("Wrote index file %q (%d bytes)\n", path, byteCount)
	}
}

func (par2LogCreateDelegate) OnRecoveryFileWrite(start, count, total int, path string, dataByteCount, byteCount int, err error) {
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

func (par2LogDecoderDelegate) OnDetectCorruptDataChunk(fileID [16]byte, path string, startByteOffset, endByteOffset int) {
	fmt.Printf("Corrupt data chunk: %q (ID %x), bytes %d to %d\n", path, fileID, startByteOffset, endByteOffset-1)
}

func (par2LogDecoderDelegate) OnDetectDataFileHashMismatch(fileID [16]byte, path string) {
	fmt.Printf("Hash mismatch for %q (ID %x)\n", path, fileID)
}

func (par2LogDecoderDelegate) OnDetectDataFileWrongByteCount(fileID [16]byte, path string) {
	fmt.Printf("Wrong byte count for %q (ID %x)\n", path, fileID)
}

func (par2LogDecoderDelegate) OnDataFileWrite(i, n int, path string, byteCount int, err error) {
	if err != nil {
		fmt.Printf("[%d/%d] Writing data file %q failed: %+v\n", i, n, path, err)
	} else {
		fmt.Printf("[%d/%d] Wrote data file %q (%d bytes)\n", i, n, path, byteCount)
	}
}

type par2LogVerifyDelegate struct {
	par2LogDecoderDelegate
}

func newFlagSet(name string) *flag.FlagSet {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	flagSet.SetOutput(ioutil.Discard)
	return flagSet
}

type globalFlags struct {
	usage         bool
	cpuProfile    string
	numGoroutines int
}

func getGlobalFlags(name string) (*flag.FlagSet, *globalFlags) {
	flagSet := newFlagSet(name)

	var flags globalFlags
	flagSet.BoolVar(&flags.usage, "h", false, "print usage info")
	flagSet.StringVar(&flags.cpuProfile, "cpuprofile", "", "if non-empty, where to write the CPU profile")
	// TODO: Detect hyperthreading and use only number of physical cores.
	flagSet.IntVar(&flags.numGoroutines, "g", rsec16.DefaultNumGoroutines(), "number of goroutines to use for encoding/decoding PAR2")

	return flagSet, &flags
}

type createFlags struct {
	sliceByteCount  int
	numParityShards int
}

func getCreateFlags(name string) (*flag.FlagSet, *createFlags) {
	flagSet := newFlagSet(name + " create")

	var flags createFlags
	flagSet.IntVar(&flags.sliceByteCount, "s", par2.SliceByteCountDefault, "block size in bytes (must be a multiple of 4) (PAR2 only)")
	// par1.NumParityFilesDefault == par2.NumParityShardsDefault
	flagSet.IntVar(&flags.numParityShards, "c", par2.NumParityShardsDefault, "number of recovery blocks to create (or files, for PAR1)")

	return flagSet, &flags
}

type verifyFlags struct {
	verifyAllData bool
}

func getVerifyFlags(name string) (*flag.FlagSet, *verifyFlags) {
	flagSet := newFlagSet(name + " verify")

	var flags verifyFlags
	// TODO: Implement this for PAR2 too.
	flagSet.BoolVar(&flags.verifyAllData, "a", false, "whether or not to do extra checking even if no missing or corrupt files are detected (PAR1 only)")
	return flagSet, &flags
}

type repairFlags struct {
	checkParity bool
}

func getRepairFlags(name string) (*flag.FlagSet, *repairFlags) {
	flagSet := newFlagSet(name + " repair")

	var flags repairFlags
	flagSet.BoolVar(&flags.checkParity, "checkparity", false, "check parity files before repairing")

	return flagSet, &flags
}

type commandMask int

const (
	createCommand commandMask = 1 << iota
	verifyCommand
	repairCommand
	allCommands = createCommand | verifyCommand | repairCommand
)

func printUsageAndExit(name string, mask commandMask, err error) {
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}

	fmt.Printf("\nUsage:\n")

	if mask&createCommand != 0 {
		fmt.Printf("  %s [global options] c(reate) [create options] <PAR file> <data files...>\n", name)
	}

	if mask&verifyCommand != 0 {
		fmt.Printf("  %s [global options] v(erify) [verify options] <PAR file>\n", name)
	}

	if mask&repairCommand != 0 {
		fmt.Printf("  %s [global options] f(epair) [repair options] <PAR file>\n", name)
	}

	fmt.Printf("\nGlobal options\n")
	globalFlagSet, _ := getGlobalFlags(name)
	globalFlagSet.SetOutput(os.Stdout)
	globalFlagSet.PrintDefaults()

	if mask&createCommand != 0 {
		fmt.Printf("\nCreate options\n")
		createFlagSet, _ := getCreateFlags(name)
		createFlagSet.SetOutput(os.Stdout)
		createFlagSet.PrintDefaults()
	}

	if mask&verifyCommand != 0 {
		fmt.Printf("\nVerify options\n")
		verifyFlagSet, _ := getVerifyFlags(name)
		verifyFlagSet.SetOutput(os.Stdout)
		verifyFlagSet.PrintDefaults()
	}

	if mask&repairCommand != 0 {
		fmt.Printf("\nRepair options\n")
		repairFlagSet, _ := getRepairFlags(name)
		repairFlagSet.SetOutput(os.Stdout)
		repairFlagSet.PrintDefaults()
	}

	fmt.Printf("\n")
	if err != nil {
		os.Exit(par2cmdline.ExitInvalidCommandLineArguments)
	}
	os.Exit(par2cmdline.ExitSuccess)
}

func printCreateErrorAndExit(err error, exitCode int) {
	fmt.Printf("Create error: %s\n", err)
	os.Exit(exitCode)
}

func printVerifyErrorAndExit(err error, exitCode int) {
	fmt.Printf("Verify error: %s\n", err)
	os.Exit(exitCode)
}

type decoder interface {
	LoadFileData() error
	LoadParityData() error
	Repair(checkParity bool) ([]string, error)
}

func newDecoder(parFile string, numGoroutines int) (decoder, error) {
	// TODO: Detect file type more robustly.
	ext := path.Ext(parFile)
	if ext == ".par2" {
		return par2.NewDecoder(par2LogDecoderDelegate{}, parFile, numGoroutines)
	}
	return par1.NewDecoder(par1LogDecoderDelegate{}, parFile)
}

type repairChecker interface {
	RepairNeeded() bool
	RepairPossible() bool
}

func processRepairChecker(repairChecker repairChecker) int {
	if repairChecker.RepairNeeded() {
		if repairChecker.RepairPossible() {
			fmt.Printf("Repair necessary and possible.\n")
			return par2cmdline.ExitRepairPossible
		}
		fmt.Printf("Repair necessary but not possible.\n")
		return par2cmdline.ExitRepairNotPossible
	}
	return par2cmdline.ExitSuccess
}

func processRepairError(err error) int {
	// Match exit codes to par2cmdline.
	if err != nil {
		switch err.(type) {
		case rsec16.NotEnoughParityShardsError:
			fmt.Printf("Repair necessary but not possible.\n")
			return par2cmdline.ExitRepairNotPossible
		default:
			fmt.Printf("Error encountered: %s\n", err)
			return par2cmdline.ExitLogicError
		}
	}
	return par2cmdline.ExitSuccess
}

func main() {
	name := filepath.Base(os.Args[0])

	globalFlagSet, globalFlags := getGlobalFlags(name)
	err := globalFlagSet.Parse(os.Args[1:])
	if err == nil && globalFlagSet.NArg() == 0 {
		err = errors.New("no command specified")
	}
	if err != nil || globalFlags.usage {
		printUsageAndExit(name, allCommands, err)
	}

	if globalFlags.cpuProfile != "" {
		f, err := os.Create(globalFlags.cpuProfile)
		if err != nil {
			panic(err)
		}
		defer func() {
			err := f.Close()
			if err != nil {
				panic(err)
			}
		}()

		err = pprof.StartCPUProfile(f)
		if err != nil {
			panic(err)
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			<-c
			pprof.StopCPUProfile()
			os.Exit(par2cmdline.ExitLogicError)
		}()

		defer pprof.StopCPUProfile()
	}

	cmd := globalFlagSet.Arg(0)
	args := globalFlagSet.Args()[1:]

	switch strings.ToLower(cmd) {
	case "c":
		fallthrough
	case "create":
		createFlagSet, createFlags := getCreateFlags(name)
		err := createFlagSet.Parse(args)
		if err == nil {
			if createFlagSet.NArg() == 0 {
				err = errors.New("no PAR file specified")
			} else if createFlagSet.NArg() == 1 {
				err = errors.New("no data files specified")
			}
		}
		if err != nil {
			printUsageAndExit(name, createCommand, err)
		}

		allFiles := createFlagSet.Args()
		parFile, filePaths := allFiles[0], allFiles[1:]

		switch ext := path.Ext(parFile); ext {
		case ".par":
			err := par1.Create(parFile, filePaths, par1.CreateOptions{

				NumParityFiles: createFlags.numParityShards,
				CreateDelegate: par1LogCreateDelegate{},
			})
			if err != nil {
				printCreateErrorAndExit(err, par2cmdline.ExitLogicError)
			}
			os.Exit(par2cmdline.ExitSuccess)

		case ".par2":
			err := par2.Create(parFile, filePaths, par2.CreateOptions{
				SliceByteCount:  createFlags.sliceByteCount,
				NumParityShards: createFlags.numParityShards,
				NumGoroutines:   globalFlags.numGoroutines,
				CreateDelegate:  par2LogCreateDelegate{},
			})
			if err != nil {
				printCreateErrorAndExit(err, par2.ExitCodeForCreateErrorPar2CmdLine(err))
			}
			os.Exit(par2cmdline.ExitSuccess)

		default:
			printCreateErrorAndExit(fmt.Errorf("unknown extension %s", ext), par2cmdline.ExitLogicError)
		}

	case "v":
		fallthrough
	case "verify":
		verifyFlagSet, verifyFlags := getVerifyFlags(name)
		err := verifyFlagSet.Parse(args)
		if err == nil && verifyFlagSet.NArg() == 0 {
			err = errors.New("no PAR file specified")
		}
		if err != nil {
			printUsageAndExit(name, verifyCommand, err)
		}

		parFile := verifyFlagSet.Arg(0)

		switch ext := path.Ext(parFile); ext {
		case ".par":
			result, err := par1.Verify(parFile, par1.VerifyOptions{
				VerifyAllData:  verifyFlags.verifyAllData,
				VerifyDelegate: par1LogVerifyDelegate{},
			})
			if err != nil {
				printVerifyErrorAndExit(err, par2cmdline.ExitLogicError)
			}
			exitCode := processRepairChecker(result.FileCounts)
			os.Exit(exitCode)

		case ".par2":
			result, err := par2.Verify(parFile, par2.VerifyOptions{
				NumGoroutines:  globalFlags.numGoroutines,
				VerifyDelegate: par2LogVerifyDelegate{},
			})
			if err != nil {
				printVerifyErrorAndExit(err, par2cmdline.ExitLogicError)
			}
			exitCode := processRepairChecker(result.ShardCounts)
			os.Exit(exitCode)

		default:
			printVerifyErrorAndExit(fmt.Errorf("unknown extension %s", ext), par2cmdline.ExitLogicError)
		}

	case "r":
		fallthrough
	case "repair":
		repairFlagSet, repairFlags := getRepairFlags(name)
		err := repairFlagSet.Parse(args)
		if err == nil && repairFlagSet.NArg() == 0 {
			err = errors.New("no PAR file specified")
		}
		if err != nil {
			printUsageAndExit(name, repairCommand, err)
		}

		parFile := repairFlagSet.Arg(0)

		decoder, err := newDecoder(parFile, globalFlags.numGoroutines)
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

		repairedPaths, err := decoder.Repair(repairFlags.checkParity)
		fmt.Printf("Repaired files: %v\n", repairedPaths)
		exitCode := processRepairError(err)
		os.Exit(exitCode)

	default:
		err := fmt.Errorf("unknown command '%s'", cmd)
		printUsageAndExit(name, allCommands, err)
	}
}
