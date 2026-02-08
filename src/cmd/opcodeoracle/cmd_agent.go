package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"opcodeoracle/internal/agent"
	"opcodeoracle/internal/numparse"
)

func agentCommand() *cli.Command {
	return &cli.Command{
		Name:      "agent",
		Usage:     "Run AI agent to analyze and annotate disassembly",
		ArgsUsage: "<state-file>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "provider",
				Aliases: []string{"p"},
				Usage:   "LLM provider: 'anthropic', 'openai', or 'gemini'",
				Value:   "anthropic",
			},
			&cli.StringFlag{
				Name:    "model",
				Aliases: []string{"m"},
				Usage:   "Model name (e.g., 'claude-sonnet-4-5-20250929', 'gpt-4.1', 'gemini-2.5-flash')",
			},
			&cli.StringFlag{
				Name:    "api-key",
				Aliases: []string{"k"},
				Usage:   "API key (defaults to ANTHROPIC_API_KEY, OPENAI_API_KEY, or GEMINI_API_KEY env var)",
				EnvVars: []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY"},
			},
			&cli.StringFlag{
				Name:  "base-url",
				Usage: "Base URL for OpenAI-compatible API endpoints",
			},
			&cli.StringFlag{
				Name:    "start",
				Aliases: []string{"s"},
				Usage:   "Start address for analysis (e.g., '$C000')",
			},
			&cli.StringFlag{
				Name:    "end",
				Aliases: []string{"e"},
				Usage:   "End address for analysis (e.g., '$D000')",
			},
			&cli.Float64Flag{
				Name:  "temperature",
				Usage: "Model temperature (0.0-1.0)",
				Value: 0.3,
			},
			&cli.IntFlag{
				Name:  "max-tokens",
				Usage: "Maximum tokens in response",
				Value: 16384,
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
		Action: cmdAgent,
	}
}

func cmdAgent(c *cli.Context) error {
	if c.NArg() < 1 {
		_ = cli.ShowCommandHelp(c, "agent")
		return cli.Exit("error: missing state file argument", ExitInvalidArgs)
	}

	stateFile := c.Args().Get(0)

	// Load and analyze state
	s, analyzer, err := loadAndAnalyze(stateFile)
	if err != nil {
		return err
	}

	// Build agent config
	cfg := agent.NewConfig()
	cfg.Provider = agent.Provider(c.String("provider"))
	cfg.Model = c.String("model")
	cfg.APIKey = c.String("api-key")
	cfg.BaseURL = c.String("base-url")
	cfg.Temperature = c.Float64("temperature")
	cfg.MaxTokens = c.Int("max-tokens")
	cfg.DryRun = c.Bool("dry-run")
	cfg.Verbose = c.Bool("verbose")
	cfg.StatePath = stateFile
	cfg.SaveInterval = 5

	// Parse address range
	if startStr := c.String("start"); startStr != "" {
		addr, err := numparse.ParseUint16(startStr)
		if err != nil {
			return cli.Exit("error: invalid start address: "+err.Error(), ExitInvalidArgs)
		}
		cfg.StartAddr = addr
	}

	if endStr := c.String("end"); endStr != "" {
		addr, err := numparse.ParseUint16(endStr)
		if err != nil {
			return cli.Exit("error: invalid end address: "+err.Error(), ExitInvalidArgs)
		}
		cfg.EndAddr = addr
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return cli.Exit("error: "+err.Error(), ExitInvalidArgs)
	}

	// Create and run agent
	ag := agent.New(cfg, s, analyzer, os.Stdout)

	ctx := context.Background()
	if err := ag.Run(ctx); err != nil {
		return cli.Exit("error: agent failed: "+err.Error(), ExitAnalysisError)
	}

	// Save state unless dry-run
	if !cfg.DryRun {
		if err := saveState(s, stateFile); err != nil {
			return err
		}
		fmt.Println("\nState saved to", stateFile)
	}

	return nil
}
