package main

import (
	"fmt"
	"os"

	"opcodeoracle/internal/annotations"
	"opcodeoracle/internal/stateio"

	"github.com/urfave/cli/v2"
)

func editAnnotationCommand() *cli.Command {
	return &cli.Command{
		Name:      "annotation",
		Usage:     "Set annotation in state file",
		ArgsUsage: "<state-file>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "address", Aliases: []string{"a"}, Required: true, Usage: "target address (hex like $C000 or 0xC000)"},
			&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Required: true, Usage: "annotation type (inline or headline)"},
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
	addr, err := parseNumber(c.String("address"))
	if err != nil {
		return cli.Exit("error: invalid address: "+err.Error(), ExitInvalidArgs)
	}

	// Parse and validate type
	typeStr := c.String("type")
	var annType annotations.AnnotationType
	switch typeStr {
	case "inline":
		annType = annotations.AnnotationInline
	case "headline":
		annType = annotations.AnnotationHeadline
	default:
		return cli.Exit("error: invalid type: must be 'inline' or 'headline'", ExitInvalidArgs)
	}

	comment := c.String("comment")

	// Parse and validate author
	authorStr := c.String("author")
	author, err := annotations.ParseAuthor(authorStr)
	if err != nil {
		return cli.Exit("error: invalid author: must be 'user' or 'assistant'", ExitInvalidArgs)
	}

	// Check if file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return cli.Exit("error: state file not found: "+stateFile, ExitFileNotFound)
	}

	// Load state
	s, err := stateio.Load(stateFile)
	if err != nil {
		return cli.Exit("error: loading state: "+err.Error(), ExitInvalidState)
	}

	// Set, extend, or remove annotation
	if comment == "" {
		s.Annotations.Remove(addr, author)
		fmt.Printf("Removed annotation at $%04X (author: %s)\n", addr, author)
	} else if c.Bool("extend") {
		s.Annotations.Extend(addr, annType, comment, author)
		fmt.Printf("Extended %s annotation at $%04X (author: %s)\n", typeStr, addr, author)
	} else {
		s.Annotations.Set(addr, annType, comment, author)
		fmt.Printf("Set %s annotation at $%04X (author: %s)\n", typeStr, addr, author)
	}

	// Save state
	if err := stateio.Save(s, stateFile); err != nil {
		return cli.Exit("error: saving state: "+err.Error(), ExitIOError)
	}

	return nil
}
