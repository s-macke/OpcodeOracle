package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

// Exit codes
const (
	ExitSuccess       = 0
	ExitInvalidArgs   = 1
	ExitFileNotFound  = 2
	ExitIOError       = 3
	ExitInvalidState  = 4
	ExitDisasmError   = 5
	ExitAnalysisError = 6
)

// Version information
const Version = "0.1.0"

func main() {
	app := &cli.App{
		Name:    "opcodeoracle",
		Usage:   "MOS6502 Disassembler",
		Version: Version,
		Commands: []*cli.Command{
			newCommand(),
			infoCommand(),
			exportCommand(),
			editCommand(),
			disasmCommand(),
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowAppHelp(c)
				return cli.Exit("", ExitInvalidArgs)
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		if exitErr, ok := err.(cli.ExitCoder); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(ExitInvalidArgs)
	}
}
