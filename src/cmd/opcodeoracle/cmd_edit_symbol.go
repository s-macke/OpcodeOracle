package main

import (
	"fmt"
	"os"

	"opcodeoracle/internal/stateio"
	"opcodeoracle/internal/symbols"

	"github.com/urfave/cli/v2"
)

func editSymbolCommand() *cli.Command {
	return &cli.Command{
		Name:      "symbol",
		Usage:     "Add or remove symbol from state file",
		ArgsUsage: "<state-file>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "address", Aliases: []string{"a"}, Required: true, Usage: "target address (hex like $C000 or 0xC000)"},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Required: true, Usage: "symbol name"},
			&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Usage: "symbol type (subroutine, label, byte, word, entry)"},
			&cli.BoolFlag{Name: "remove", Aliases: []string{"r"}, Usage: "remove symbol instead of adding"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cli.Exit("error: requires <state-file> argument", ExitInvalidArgs)
			}
			return cmdEditSymbol(c, c.Args().Get(0))
		},
	}
}

func cmdEditSymbol(c *cli.Context, stateFile string) error {
	// Parse address
	addr, err := parseNumber(c.String("address"))
	if err != nil {
		return cli.Exit("error: invalid address: "+err.Error(), ExitInvalidArgs)
	}

	name := c.String("name")
	remove := c.Bool("remove")

	// Check if file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return cli.Exit("error: state file not found: "+stateFile, ExitFileNotFound)
	}

	// Load state
	s, err := stateio.Load(stateFile)
	if err != nil {
		return cli.Exit("error: loading state: "+err.Error(), ExitInvalidState)
	}

	if remove {
		// Remove symbol
		s.Symbols.Remove(addr, name)
		if err := stateio.Save(s, stateFile); err != nil {
			return cli.Exit("error: saving state: "+err.Error(), ExitIOError)
		}
		fmt.Printf("Removed symbol '%s' at $%04X\n", name, addr)
		return nil
	}

	// Adding symbol - type is required
	typeStr := c.String("type")
	if typeStr == "" {
		return cli.Exit("error: --type is required when adding a symbol", ExitInvalidArgs)
	}

	var symType symbols.SymbolType
	switch typeStr {
	case "subroutine":
		symType = symbols.SymbolSubroutine
	case "label":
		symType = symbols.SymbolLabel
	case "byte":
		symType = symbols.SymbolByte
	case "word":
		symType = symbols.SymbolWord
	case "entry":
		symType = symbols.SymbolEntry
	default:
		return cli.Exit("error: invalid type: must be subroutine, label, byte, word, or entry", ExitInvalidArgs)
	}

	// Add symbol
	s.Symbols.Add(addr, symbols.Symbol{
		Name:   name,
		Type:   symType,
		Source: symbols.SourceUser,
	})

	if err := stateio.Save(s, stateFile); err != nil {
		return cli.Exit("error: saving state: "+err.Error(), ExitIOError)
	}

	fmt.Printf("Added %s symbol '%s' at $%04X\n", typeStr, name, addr)
	return nil
}
