package main

import (
	"fmt"

	"opcodeoracle/internal/author"
	"opcodeoracle/internal/numparse"

	"github.com/urfave/cli/v2"
)

func editAnnotationCommand() *cli.Command {
	return &cli.Command{
		Name:      "annotation",
		Usage:     "Set inline annotation in state file",
		ArgsUsage: "<state-file>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "address", Aliases: []string{"a"}, Required: true, Usage: "target address (hex like $C000 or 0xC000)"},
			&cli.StringFlag{Name: "comment", Aliases: []string{"c"}, Usage: "annotation text (empty to remove)"},
			&cli.StringFlag{Name: "author", Value: "user", Usage: "author (user or assistant)"},
			&cli.BoolFlag{Name: "extend", Aliases: []string{"e"}, Usage: "append to existing annotation instead of replacing"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cli.Exit("error: requires <state-file> argument", ExitInvalidArgs)
			}
			return cmdEditAnnotation(c, c.Args().Get(0))
		},
	}
}

func cmdEditAnnotation(c *cli.Context, stateFile string) error {
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

	// Set, extend, or remove annotation
	if comment == "" {
		s.Annotations.Remove(addr, a)
		fmt.Printf("Removed annotation at $%04X (author: %s)\n", addr, a)
	} else if c.Bool("extend") {
		s.Annotations.Extend(addr, comment, a)
		fmt.Printf("Extended annotation at $%04X (author: %s)\n", addr, a)
	} else {
		s.Annotations.Set(addr, comment, a)
		fmt.Printf("Set annotation at $%04X (author: %s)\n", addr, a)
	}

	return saveState(s, stateFile)
}
