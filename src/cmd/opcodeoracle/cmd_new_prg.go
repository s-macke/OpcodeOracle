package main

import (
	"fmt"
	"os"

	"opcodeoracle/internal/state"
	"opcodeoracle/internal/stateio"

	"github.com/urfave/cli/v2"
)

func newPrgCommand() *cli.Command {
	return &cli.Command{
		Name:      "prg",
		Usage:     "Create project from C64 PRG file",
		ArgsUsage: "<prg-file>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "entry", Aliases: []string{"e"}, Required: true, Usage: "entry point address"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cli.Exit("error: requires <prg-file> argument", ExitInvalidArgs)
			}
			return cmdNewPrg(c, c.Args().Get(0))
		},
	}
}

func cmdNewPrg(c *cli.Context, binaryFile string) error {
	entryNum, err := parseNumber(c.String("entry"))
	if err != nil {
		return cli.Exit("error: invalid entry value: "+err.Error(), ExitInvalidArgs)
	}

	// Read file
	fileData, err := os.ReadFile(binaryFile)
	if err != nil {
		return cli.Exit("error: reading file: "+err.Error(), ExitIOError)
	}

	// Validate file has at least 3 bytes (2-byte header + 1 byte data)
	if len(fileData) < 3 {
		return cli.Exit("error: PRG file too small (minimum 3 bytes)", ExitInvalidArgs)
	}

	// First 2 bytes are little-endian load address
	origin := uint16(fileData[0]) | uint16(fileData[1])<<8
	data := fileData[2:]
	entryPoints := []uint16{uint16(entryNum)}

	s := state.NewState(data, origin, entryPoints, binaryFile)

	outputFile := outputFilename(binaryFile)
	if err := stateio.Save(s, outputFile); err != nil {
		return cli.Exit("error: saving state: "+err.Error(), ExitIOError)
	}

	fmt.Printf("Created %s\n", outputFile)
	return nil
}
