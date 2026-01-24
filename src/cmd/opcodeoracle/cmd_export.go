package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func exportCommand() *cli.Command {
	return &cli.Command{
		Name:      "export",
		Usage:     "Export state to assembly files",
		ArgsUsage: "<state-file>",
		Action:    cmdExport,
	}
}

func cmdExport(c *cli.Context) error {
	if c.NArg() < 1 {
		_ = cli.ShowCommandHelp(c, "export")
		return cli.Exit("error: missing state file argument", ExitInvalidArgs)
	}

	stateFile := c.Args().Get(0)
	fmt.Printf("export: file=%s\n", stateFile)
	return nil
}
