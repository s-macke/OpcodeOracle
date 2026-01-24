package main

import (
	"fmt"
	"os"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/stateio"

	"github.com/urfave/cli/v2"
)

func newSidCommand() *cli.Command {
	return &cli.Command{
		Name:      "sid",
		Usage:     "Create project from SID music file",
		ArgsUsage: "<sid-file>",
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cli.Exit("error: requires <sid-file> argument", ExitInvalidArgs)
			}
			return cmdNewSid(c.Args().Get(0))
		},
	}
}

func cmdNewSid(binaryFile string) error {
	// Read file
	fileData, err := os.ReadFile(binaryFile)
	if err != nil {
		return cli.Exit("error: reading file: "+err.Error(), ExitIOError)
	}

	// Validate minimum header size (at least 14 bytes for basic header fields)
	if len(fileData) < 14 {
		return cli.Exit("error: SID file too small", ExitInvalidArgs)
	}

	// Validate magic is "PSID" or "RSID"
	magic := string(fileData[0:4])
	if magic != "PSID" && magic != "RSID" {
		return cli.Exit("error: invalid SID magic (expected PSID or RSID)", ExitInvalidArgs)
	}

	// Parse header fields (all big-endian)
	dataOffset := uint16(fileData[6])<<8 | uint16(fileData[7])
	loadAddress := uint16(fileData[8])<<8 | uint16(fileData[9])
	initAddress := uint16(fileData[10])<<8 | uint16(fileData[11])
	playAddress := uint16(fileData[12])<<8 | uint16(fileData[13])

	// Validate dataOffset
	if int(dataOffset) > len(fileData) {
		return cli.Exit("error: SID dataOffset exceeds file length", ExitInvalidArgs)
	}

	var origin uint16
	var data []byte

	if loadAddress == 0 {
		// Load address is in first 2 bytes of data (little-endian)
		if int(dataOffset)+2 > len(fileData) {
			return cli.Exit("error: SID file too small for embedded load address", ExitInvalidArgs)
		}
		origin = uint16(fileData[dataOffset]) | uint16(fileData[dataOffset+1])<<8
		data = fileData[dataOffset+2:]
	} else {
		origin = loadAddress
		data = fileData[dataOffset:]
	}

	// Build entry points
	var entryPoints []uint16

	// If initAddress is 0, use origin as init
	if initAddress == 0 {
		entryPoints = append(entryPoints, origin)
	} else {
		entryPoints = append(entryPoints, initAddress)
	}

	// If playAddress is non-zero, add it as an entry point
	if playAddress != 0 {
		entryPoints = append(entryPoints, playAddress)
	}

	s := state.NewState(data, origin, entryPoints, binaryFile)

	// Run flow analysis
	fmt.Printf("Analyzing from %d entry point(s)...\n", len(entryPoints))
	analyzer := analysis.NewAnalyzer(s, analysis.UpdateAll)
	if err := analyzer.Analyze(); err != nil {
		return cli.Exit("error: analyzing: "+err.Error(), ExitAnalysisError)
	}

	outputFile := outputFilename(binaryFile)
	if err := stateio.Save(s, outputFile); err != nil {
		return cli.Exit("error: saving state: "+err.Error(), ExitIOError)
	}

	fmt.Printf("Created %s\n", outputFile)
	return nil
}
