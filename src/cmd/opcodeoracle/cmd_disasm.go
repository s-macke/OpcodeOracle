package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"

	"opcodeoracle/internal/disasm"
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
		parsed, err := parseAddress(startStr)
		if err != nil {
			return cli.Exit("error: invalid start address: "+err.Error(), ExitInvalidArgs)
		}
		start = parsed
	}

	if endStr := c.String("end"); endStr != "" {
		parsed, err := parseAddress(endStr)
		if err != nil {
			return cli.Exit("error: invalid end address: "+err.Error(), ExitInvalidArgs)
		}
		end = parsed
	}

	// Create disassembler with boundaries for mid-instruction checking
	d := disasm.NewDisassembler(s, analyzer)

	// Disassemble (end is exclusive in the API, so add 1)
	output, err := d.Disassemble(start, end+1)
	if err != nil {
		return cli.Exit("error: "+err.Error(), ExitDisasmError)
	}

	fmt.Print(output)
	return nil
}

// parseAddress parses an address string in various formats:
// $C000, 0xC000, C000, 49152
func parseAddress(s string) (uint16, error) {
	s = strings.TrimSpace(s)

	// Handle $ prefix (6502 convention)
	if strings.HasPrefix(s, "$") {
		s = s[1:]
		val, err := strconv.ParseUint(s, 16, 16)
		if err != nil {
			return 0, fmt.Errorf("invalid hex address: %s", s)
		}
		return uint16(val), nil
	}

	// Handle 0x prefix
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		val, err := strconv.ParseUint(s[2:], 16, 16)
		if err != nil {
			return 0, fmt.Errorf("invalid hex address: %s", s)
		}
		return uint16(val), nil
	}

	// Try hex first (if all hex digits), then decimal
	if isHexString(s) {
		val, err := strconv.ParseUint(s, 16, 16)
		if err == nil {
			return uint16(val), nil
		}
	}

	// Try decimal
	val, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid address: %s", s)
	}
	return uint16(val), nil
}

// isHexString returns true if s contains only hex digits and at least one a-f/A-F.
func isHexString(s string) bool {
	hasHexLetter := false
	for _, c := range s {
		if (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			hasHexLetter = true
		} else if c < '0' || c > '9' {
			return false
		}
	}
	return hasHexLetter
}
