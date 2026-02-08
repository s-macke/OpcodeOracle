package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"opcodeoracle/internal/disasm"
	"opcodeoracle/internal/numparse"
)

func disasmCommand() *cli.Command {
	return &cli.Command{
		Name:      "disasm",
		Usage:     "Disassemble to stdout",
		ArgsUsage: "<state-file>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "start",
				Aliases: []string{"s"},
				Usage:   "Start address (e.g., $C000 or 0xC000)",
			},
			&cli.StringFlag{
				Name:    "end",
				Aliases: []string{"e"},
				Usage:   "End address inclusive (e.g., $C0FF)",
			},
		},
		Action: cmdDisasm,
	}
}

func cmdDisasm(c *cli.Context) error {
	if c.NArg() < 1 {
		_ = cli.ShowCommandHelp(c, "disasm")
		return cli.Exit("error: missing state file argument", ExitInvalidArgs)
	}

	stateFile := c.Args().Get(0)
	s, analyzer, err := loadAndAnalyze(stateFile)
	if err != nil {
		return err
	}

	// Determine address range (use binary bounds as defaults)
	start := s.Binary.Start()
	end := s.Binary.End()

	if startStr := c.String("start"); startStr != "" {
		parsed, err := numparse.ParseUint16(startStr)
		if err != nil {
			return cli.Exit("error: invalid start address: "+err.Error(), ExitInvalidArgs)
		}
		start = parsed
	}

	if endStr := c.String("end"); endStr != "" {
		parsed, err := numparse.ParseUint16(endStr)
		if err != nil {
			return cli.Exit("error: invalid end address: "+err.Error(), ExitInvalidArgs)
		}
		end = parsed
	}

	// Create disassembler with boundaries for mid-instruction checking
	d := disasm.NewDisassembler(s, analyzer)

	// Disassemble (end is inclusive in CLI, exclusive in API).
	output, err := d.Disassemble(start, inclusiveToExclusiveEnd(end))
	if err != nil {
		return cli.Exit("error: "+err.Error(), ExitDisasmError)
	}

	fmt.Print(output)
	return nil
}

func inclusiveToExclusiveEnd(end uint16) uint16 {
	if end == 0xFFFF {
		return 0xFFFF
	}
	return end + 1
}
