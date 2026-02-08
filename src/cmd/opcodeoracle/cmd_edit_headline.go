package main

import (
	"fmt"

	"opcodeoracle/internal/author"
	"opcodeoracle/internal/numparse"

	"github.com/urfave/cli/v2"
)

func editHeadlineCommand() *cli.Command {
	return &cli.Command{
		Name:      "headline",
		Usage:     "Set headline in state file",
		ArgsUsage: "<state-file>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "address", Aliases: []string{"a"}, Required: true, Usage: "target address (hex like $C000 or 0xC000)"},
			&cli.StringFlag{Name: "comment", Aliases: []string{"c"}, Usage: "headline text (empty to remove)"},
			&cli.StringFlag{Name: "author", Value: "user", Usage: "author (user or assistant)"},
			&cli.BoolFlag{Name: "extend", Aliases: []string{"e"}, Usage: "append to existing headline instead of replacing"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cli.Exit("error: requires <state-file> argument", ExitInvalidArgs)
			}
			return cmdEditHeadline(c, c.Args().Get(0))
		},
	}
}

func cmdEditHeadline(c *cli.Context, stateFile string) error {
	// Parse address
	addr, err := numparse.ParseUint16(c.String("address"))
	if err != nil {
		return cli.Exit("error: invalid address: "+err.Error(), ExitInvalidArgs)
	}

	comment := c.String("comment")

	// Parse and validate author
	authorStr := c.String("author")
	a, err := author.Parse(authorStr)
	if err != nil {
		return cli.Exit("error: invalid author: must be 'user' or 'assistant'", ExitInvalidArgs)
	}

	// Load state
	s, err := loadState(stateFile)
	if err != nil {
		return err
	}

	// Set, extend, or remove headline
	if comment == "" {
		s.Headlines.Remove(addr, a)
		fmt.Printf("Removed headline at $%04X (author: %s)\n", addr, a)
	} else if c.Bool("extend") {
		s.Headlines.Extend(addr, comment, a)
		fmt.Printf("Extended headline at $%04X (author: %s)\n", addr, a)
	} else {
		s.Headlines.Set(addr, comment, a)
		fmt.Printf("Set headline at $%04X (author: %s)\n", addr, a)
	}

	return saveState(s, stateFile)
}
