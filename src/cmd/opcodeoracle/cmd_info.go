package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func infoCommand() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Usage:     "Display state file information",
		ArgsUsage: "<state-file>",
		Action:    cmdInfo,
	}
}

func cmdInfo(c *cli.Context) error {
	if c.NArg() < 1 {
		_ = cli.ShowCommandHelp(c, "info")
		return cli.Exit("error: missing state file argument", ExitInvalidArgs)
	}

	stateFile := c.Args().Get(0)
	fmt.Printf("info: file=%s\n", stateFile)
	return nil
}
