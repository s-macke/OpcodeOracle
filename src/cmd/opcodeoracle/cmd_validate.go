package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"opcodeoracle/internal/validate"
)

func validateCommand() *cli.Command {
	return &cli.Command{
		Name:      "validate",
		Usage:     "Validate state file for potential issues",
		ArgsUsage: "<state-file>",
		Action:    cmdValidate,
	}
}

func cmdValidate(c *cli.Context) error {
	if c.NArg() < 1 {
		_ = cli.ShowCommandHelp(c, "validate")
		return cli.Exit("error: missing state file argument", ExitInvalidArgs)
	}

	stateFile := c.Args().Get(0)
	s, analyzer, err := loadAndAnalyze(stateFile)
	if err != nil {
		return err
	}

	// Run validation
	issues := validate.Validate(s, analyzer)

	if len(issues) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	for _, issue := range issues {
		fmt.Println(issue)
	}

	return nil
}
