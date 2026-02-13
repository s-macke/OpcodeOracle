package main

import (
	"fmt"

	"opcodeoracle/internal/numparse"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/reinterpret"

	"github.com/urfave/cli/v2"
)

func editReinterpretCommand() *cli.Command {
	return &cli.Command{
		Name:      "reinterpret",
		Usage:     "Force reinterpretation as code (single address) or data (range), then rerun analysis from scratch",
		ArgsUsage: "<state-file>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "code-address", Usage: "single address to force as code seed (hex like $C000 or 0xC000)"},
			&cli.StringFlag{Name: "data-start", Usage: "start address of range to force as data"},
			&cli.StringFlag{Name: "data-end", Usage: "end address of range to force as data"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cli.Exit("error: requires <state-file> argument", ExitInvalidArgs)
			}
			return cmdEditReinterpret(c, c.Args().Get(0))
		},
	}
}

func cmdEditReinterpret(c *cli.Context, stateFile string) error {
	codeAddrStr := c.String("code-address")
	dataStartStr := c.String("data-start")
	dataEndStr := c.String("data-end")

	codeMode := codeAddrStr != ""
	dataMode := dataStartStr != "" || dataEndStr != ""
	if codeMode == dataMode {
		return cli.Exit("error: specify either --code-address OR both --data-start and --data-end", ExitInvalidArgs)
	}
	if dataMode && (dataStartStr == "" || dataEndStr == "") {
		return cli.Exit("error: --data-start and --data-end must be provided together", ExitInvalidArgs)
	}

	s, err := loadState(stateFile)
	if err != nil {
		return err
	}

	if codeMode {
		addr, err := numparse.ParseUint16(codeAddrStr)
		if err != nil {
			return cli.Exit("error: invalid code address: "+err.Error(), ExitInvalidArgs)
		}
		analyzer, err := reinterpret.AsCode(s, addr, regions.RegionSourceUser)
		if err != nil {
			return cli.Exit("error: reinterpretation failed: "+err.Error(), ExitAnalysisError)
		}
		if err := saveState(s, stateFile); err != nil {
			return err
		}
		fmt.Printf("Reinterpreted $%04X as code; rebuilt CFG with %d instructions\n",
			addr, len(analyzer.InstructionAddresses()))
		return nil
	}

	start, err := numparse.ParseUint16(dataStartStr)
	if err != nil {
		return cli.Exit("error: invalid data start: "+err.Error(), ExitInvalidArgs)
	}
	end, err := numparse.ParseUint16(dataEndStr)
	if err != nil {
		return cli.Exit("error: invalid data end: "+err.Error(), ExitInvalidArgs)
	}
	analyzer, err := reinterpret.AsData(s, start, end, regions.RegionSourceUser)
	if err != nil {
		return cli.Exit("error: reinterpretation failed: "+err.Error(), ExitAnalysisError)
	}
	if err := saveState(s, stateFile); err != nil {
		return err
	}
	fmt.Printf("Reinterpreted $%04X-$%04X as hard-locked data; rebuilt CFG with %d instructions\n",
		start, end, len(analyzer.InstructionAddresses()))
	return nil
}
