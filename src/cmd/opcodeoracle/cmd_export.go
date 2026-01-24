package main

import (
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"opcodeoracle/internal/export"
	"opcodeoracle/internal/stateio"
)

func exportCommand() *cli.Command {
	return &cli.Command{
		Name:      "export",
		Usage:     "Export state to assembly file",
		ArgsUsage: "<state-file>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file path",
			},
		},
		Action: cmdExport,
	}
}

func cmdExport(c *cli.Context) error {
	if c.NArg() < 1 {
		_ = cli.ShowCommandHelp(c, "export")
		return cli.Exit("error: missing state file argument", ExitInvalidArgs)
	}

	stateFile := c.Args().Get(0)

	s, err := stateio.Load(stateFile)
	if err != nil {
		return cli.Exit("error: "+err.Error(), ExitInvalidState)
	}

	outputPath := c.String("output")
	if outputPath == "" {
		// Derive from state file: foo.opcodeoracle.json -> foo.asm
		outputPath = deriveOutputPath(stateFile, ".asm")
	}

	exp := export.NewExporter(s)
	if err := exp.Export(outputPath); err != nil {
		return cli.Exit("error: "+err.Error(), ExitIOError)
	}

	return nil
}

// deriveOutputPath derives an output file path from the state file.
func deriveOutputPath(stateFile, ext string) string {
	base := filepath.Base(stateFile)
	// Remove .opcodeoracle.json suffix if present
	if strings.HasSuffix(base, ".opcodeoracle.json") {
		base = strings.TrimSuffix(base, ".opcodeoracle.json")
	} else {
		// Just remove extension
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return filepath.Join(filepath.Dir(stateFile), base+ext)
}
