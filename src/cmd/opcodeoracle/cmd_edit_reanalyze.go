package main

import (
	"fmt"

	"opcodeoracle/internal/analysis"

	"github.com/urfave/cli/v2"
)

func editReanalyzeCommand() *cli.Command {
	return &cli.Command{
		Name:      "reanalyze",
		Usage:     "Rebuild auto regions, symbols, and xrefs from the current entry points",
		ArgsUsage: "<state-file>",
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cli.Exit("error: requires <state-file> argument", ExitInvalidArgs)
			}
			return cmdEditReanalyze(c.Args().Get(0))
		},
	}
}

func cmdEditReanalyze(stateFile string) error {
	s, err := loadState(stateFile)
	if err != nil {
		return err
	}

	analyzer, err := analysis.ReanalyzeFromEntryPoints(s)
	if err != nil {
		return cli.Exit("error: reanalysis failed: "+err.Error(), ExitAnalysisError)
	}

	if err := saveState(s, stateFile); err != nil {
		return err
	}

	fmt.Printf("Reanalyzed from %d entry point(s); rebuilt auto regions with %d instructions\n",
		len(s.EntryPoints), len(analyzer.InstructionAddresses()))
	return nil
}
