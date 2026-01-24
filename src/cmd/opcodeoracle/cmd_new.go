package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func newCommand() *cli.Command {
	return &cli.Command{
		Name:      "new",
		Usage:     "Create new project from binary",
		ArgsUsage: "<type> <binary-file>",
		Description: `Creates a new state file from a binary and runs flow-following disassembly.

File Types:
  bin    Raw binary data (requires --skip, --entry, --origin)
  prg    C64 PRG file with load address (requires --entry)
  sid    SID music file (no additional options required)

Number Format:
  Decimal:   2048
  Hex ($):   $0800
  Hex (0x):  0x0800`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "skip",
				Aliases: []string{"s"},
				Usage:   "bytes to skip at start of file (bin only)",
			},
			&cli.StringFlag{
				Name:    "entry",
				Aliases: []string{"e"},
				Usage:   "entry point address (bin, prg)",
			},
			&cli.StringFlag{
				Name:    "origin",
				Aliases: []string{"o"},
				Usage:   "load address/origin (bin only)",
			},
		},
		Action: cmdNew,
	}
}

func cmdNew(c *cli.Context) error {
	if c.NArg() < 2 {
		_ = cli.ShowCommandHelp(c, "new")
		return cli.Exit("error: missing required arguments <type> <binary-file>", ExitInvalidArgs)
	}

	fileType := c.Args().Get(0)
	binaryFile := c.Args().Get(1)

	switch fileType {
	case "bin":
		return cmdNewBin(c, binaryFile)
	case "prg":
		return cmdNewPrg(c, binaryFile)
	case "sid":
		return cmdNewSid(binaryFile)
	default:
		return cli.Exit("error: invalid file type: "+fileType+" (must be bin, prg, or sid)", ExitInvalidArgs)
	}
}

func cmdNewBin(c *cli.Context, binaryFile string) error {
	skipStr := c.String("skip")
	entryStr := c.String("entry")
	originStr := c.String("origin")

	if skipStr == "" {
		return cli.Exit("error: --skip is required for bin type", ExitInvalidArgs)
	}
	if entryStr == "" {
		return cli.Exit("error: --entry is required for bin type", ExitInvalidArgs)
	}
	if originStr == "" {
		return cli.Exit("error: --origin is required for bin type", ExitInvalidArgs)
	}

	skipNum, err := parseNumber(skipStr)
	if err != nil {
		return cli.Exit("error: invalid skip value: "+err.Error(), ExitInvalidArgs)
	}

	entryNum, err := parseNumber(entryStr)
	if err != nil {
		return cli.Exit("error: invalid entry value: "+err.Error(), ExitInvalidArgs)
	}

	originNum, err := parseNumber(originStr)
	if err != nil {
		return cli.Exit("error: invalid origin value: "+err.Error(), ExitInvalidArgs)
	}

	fmt.Printf("new bin: file=%s skip=%d entry=$%04X origin=$%04X\n",
		binaryFile, skipNum, entryNum, originNum)
	return nil
}

func cmdNewPrg(c *cli.Context, binaryFile string) error {
	entryStr := c.String("entry")

	if entryStr == "" {
		return cli.Exit("error: --entry is required for prg type", ExitInvalidArgs)
	}

	entryNum, err := parseNumber(entryStr)
	if err != nil {
		return cli.Exit("error: invalid entry value: "+err.Error(), ExitInvalidArgs)
	}

	fmt.Printf("new prg: file=%s entry=$%04X\n", binaryFile, entryNum)
	return nil
}

func cmdNewSid(binaryFile string) error {
	fmt.Printf("new sid: file=%s\n", binaryFile)
	return nil
}
