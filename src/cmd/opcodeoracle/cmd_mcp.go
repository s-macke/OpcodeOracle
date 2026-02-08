package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"opcodeoracle/internal/agent/tools"
	"opcodeoracle/internal/mcpserver"
)

func mcpCommand() *cli.Command {
	return &cli.Command{
		Name:      "mcp",
		Usage:     "Start MCP server exposing OpcodeOracle tools over stdio or streamable HTTP",
		ArgsUsage: "<state-file>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "transport",
				Usage: "MCP transport: 'stdio' or 'http'",
				Value: string(mcpserver.TransportStdio),
			},
			&cli.StringFlag{
				Name:  "listen",
				Usage: "Listen address for HTTP transport (e.g. 127.0.0.1:8080)",
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "HTTP endpoint path for streamable MCP transport",
				Value: "/mcp",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show changes without saving to state file",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable verbose output",
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
	runOpts, err := buildMCPRunOptions(c.String("transport"), c.String("listen"), c.String("path"))
	if err != nil {
		return cli.Exit("error: "+err.Error(), ExitInvalidArgs)
	}

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
		Output:    os.Stderr,
		Verbose:   c.Bool("verbose"),
	}

	server := mcpserver.New(cfg)

	ctx := context.Background()
	if err := mcpserver.Run(ctx, server, cfg, runOpts); err != nil {
		return cli.Exit("error: mcp server failed: "+err.Error(), ExitAnalysisError)
	}

	return nil
}

func buildMCPRunOptions(transport, listen, path string) (mcpserver.RunOptions, error) {
	t := mcpserver.Transport(strings.ToLower(strings.TrimSpace(transport)))

	switch t {
	case "", mcpserver.TransportStdio:
		return mcpserver.RunOptions{
			Transport: mcpserver.TransportStdio,
		}, nil
	case mcpserver.TransportHTTP:
		listen = strings.TrimSpace(listen)
		if listen == "" {
			return mcpserver.RunOptions{}, fmt.Errorf("--transport http requires --listen")
		}
		path = strings.TrimSpace(path)
		if path == "" {
			return mcpserver.RunOptions{}, fmt.Errorf("--path cannot be empty")
		}
		if !strings.HasPrefix(path, "/") {
			return mcpserver.RunOptions{}, fmt.Errorf("--path must start with '/'")
		}
		return mcpserver.RunOptions{
			Transport:  mcpserver.TransportHTTP,
			ListenAddr: listen,
			Path:       path,
		}, nil
	default:
		return mcpserver.RunOptions{}, fmt.Errorf("invalid transport %q: must be 'stdio' or 'http'", transport)
	}
}
