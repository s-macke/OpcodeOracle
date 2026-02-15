package main

import (
	"fmt"
	"os"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/numparse"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/stateio"

	"github.com/urfave/cli/v2"
)

func newBinCommand() *cli.Command {
	return &cli.Command{
		Name:      "bin",
		Usage:     "Create project from raw binary",
		ArgsUsage: "<binary-file>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "skip", Aliases: []string{"s"}, Value: "0", Usage: "bytes to skip at start"},
			&cli.StringFlag{Name: "entry", Aliases: []string{"e"}, Required: true, Usage: "entry point address(es), comma-separated"},
			&cli.StringFlag{Name: "origin", Aliases: []string{"o"}, Value: "0", Usage: "load address"},
			&cli.StringFlag{Name: "description", Aliases: []string{"d"}, Usage: "project description"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cli.Exit("error: requires <binary-file> argument", ExitInvalidArgs)
			}
			return cmdNewBin(c, c.Args().Get(0))
		},
	}
}

func cmdNewBin(c *cli.Context, binaryFile string) error {
	skipNum, err := numparse.ParseUint16(c.String("skip"))
	if err != nil {
		return cli.Exit("error: invalid skip value: "+err.Error(), ExitInvalidArgs)
	}

	entryPoints, err := numparse.ParseUint16List(c.String("entry"))
	if err != nil {
		return cli.Exit("error: invalid entry value: "+err.Error(), ExitInvalidArgs)
	}

	originNum, err := numparse.ParseUint16(c.String("origin"))
	if err != nil {
		return cli.Exit("error: invalid origin value: "+err.Error(), ExitInvalidArgs)
	}

	// Read file
	fileData, err := os.ReadFile(binaryFile)
	if err != nil {
		return cli.Exit("error: reading file: "+err.Error(), ExitIOError)
	}

	// Validate skip doesn't exceed file length
	if int(skipNum) > len(fileData) {
		return cli.Exit("error: skip value exceeds file length", ExitInvalidArgs)
	}

	data := fileData[skipNum:]
	origin := uint16(originNum)

	s := state.NewState(data, origin, entryPoints, binaryFile)
	s.Metadata.Description = c.String("description")

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
