package main

import "github.com/urfave/cli/v2"

func editCommand() *cli.Command {
	return &cli.Command{
		Name:  "edit",
		Usage: "Edit state file",
		Subcommands: []*cli.Command{
			editAnnotationCommand(),
			editHeadlineCommand(),
			editReanalyzeCommand(),
			editReinterpretCommand(),
			editSymbolCommand(),
		},
	}
}
