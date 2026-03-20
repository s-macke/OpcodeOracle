package main

import (
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

func newCommand() *cli.Command {
	return &cli.Command{
		Name:  "new",
		Usage: "Create new project from binary",
		Subcommands: []*cli.Command{
			newBinCommand(),
			newPrgCommand(),
			newSidCommand(),
		},
	}
}

// outputFilename generates the output filename by replacing the extension with .opcodeoracle.json
func outputFilename(inputFile string) string {
	name := filepath.Base(inputFile)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return base + ".opcodeoracle.json"
}
