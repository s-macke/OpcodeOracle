package main

import (
	"context"

	"github.com/urfave/cli/v2"

	"opcodeoracle/internal/agent/tools"
	"opcodeoracle/internal/mcpserver"
)

func mcpCommand() *cli.Command {
	return &cli.Command{
		Name:      "mcp",
		Usage:     "Start MCP server exposing OpcodeOracle tools over stdio",
		ArgsUsage: "<state-file>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show changes without saving to state file",
			},
		},
		Action: cmdMCP,
	}
}

func cmdMCP(c *cli.Context) error {
	if c.NArg() < 1 {
		_ = cli.ShowCommandHelp(c, "mcp")
		return cli.Exit("error: missing state file argument", ExitInvalidArgs)
	}

	stateFile := c.Args().Get(0)

	// Load and analyze state
	s, analyzer, err := loadAndAnalyze(stateFile)
	if err != nil {
		return err
	}

	// Create tool context
	toolCtx := &tools.Context{
		State:    s,
		Analyzer: analyzer,
		DryRun:   c.Bool("dry-run"),
	}

	// Create and run MCP server
	cfg := &mcpserver.Config{
		StatePath: stateFile,
		ToolCtx:   toolCtx,
	}

	server := mcpserver.New(cfg)

	ctx := context.Background()
	if err := mcpserver.Run(ctx, server); err != nil {
		return cli.Exit("error: mcp server failed: "+err.Error(), ExitAnalysisError)
	}

	return nil
}
