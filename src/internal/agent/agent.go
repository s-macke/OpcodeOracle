package agent

import (
	"context"
	"fmt"
	"io"
	"strings"

	"opcodeoracle/internal/agent/tools"
	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/stateio"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// Agent orchestrates the AI-powered analysis of MOS6502 disassembly.
type Agent struct {
	config   *Config
	state    *state.State
	analyzer *analysis.Analyzer
	output   io.Writer
}

// New creates a new Agent with the given configuration.
func New(cfg *Config, s *state.State, analyzer *analysis.Analyzer, output io.Writer) *Agent {
	return &Agent{
		config:   cfg,
		state:    s,
		analyzer: analyzer,
		output:   output,
	}
}

// Run executes the agent analysis loop.
func (a *Agent) Run(ctx context.Context) error {
	// Create the chat model
	chatModel, err := NewChatModel(ctx, a.config)
	if err != nil {
		return fmt.Errorf("failed to create chat model: %w", err)
	}

	// Create tool context
	toolCtx := &tools.Context{
		State:    a.state,
		Analyzer: a.analyzer,
		DryRun:   a.config.DryRun,
		Verbose:  a.config.Verbose,
	}

	// Get all tools
	allTools := tools.AllTools(toolCtx)

	// Create ADK agent
	adkAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "opcodeoracle",
		Description: "MOS6502 reverse engineering agent",
		Instruction: SystemPrompt,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: allTools,
			},
		},
		MaxIterations: 10000,
	})
	if err != nil {
		return fmt.Errorf("failed to create ADK agent: %w", err)
	}

	// Build input with the task prompt
	taskPrompt := a.buildTaskPrompt()
	input := &adk.AgentInput{
		Messages: []*schema.Message{
			schema.UserMessage(taskPrompt),
		},
	}

	if a.config.Verbose {
		fmt.Fprintf(a.output, "Starting analysis with %s (%s)...\n", a.config.Provider, a.getModelName())
		fmt.Fprintf(a.output, "Binary range: $%04X-$%04X\n", a.state.Binary.Start(), a.state.Binary.End())
		if a.config.StartAddr != 0 || a.config.EndAddr != 0 {
			fmt.Fprintf(a.output, "Analysis range: $%04X-$%04X\n", a.config.StartAddr, a.config.EndAddr)
		}
		fmt.Fprintln(a.output, "")
	}

	// Build run options (enable prompt caching for Anthropic)
	var runOpts []adk.AgentRunOption
	if a.config.Provider == ProviderAnthropic {
		runOpts = append(runOpts, adk.WithChatModelOptions([]model.Option{
			claude.WithEnableAutoCache(true),
		}))
	}

	// Run the agent and iterate events
	var toolCallCount int
	var lastSavedMutationCount uint64
	var totalPromptTokens, totalCompletionTokens, totalCachedTokens int
	iter := adkAgent.Run(ctx, input, runOpts...)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			if shouldAttemptSaveOnAgentError(
				a.config.DryRun,
				a.config.StatePath,
				a.state.Metadata.ArchiveOnSave,
				a.config.SaveInterval,
				toolCtx.MutationCount,
				lastSavedMutationCount,
			) {
				toolCtx.Mu.Lock()
				if a.state.Metadata.ArchiveOnSave {
					if toolCtx.MutationCount > lastSavedMutationCount {
						_ = stateio.Save(a.state, a.config.StatePath)
					}
				} else if a.config.SaveInterval > 0 {
					_ = stateio.Save(a.state, a.config.StatePath)
				}
				toolCtx.Mu.Unlock()
			}
			return fmt.Errorf("agent error: %w", event.Err)
		}

		// Handle output events
		if event.Output != nil && event.Output.MessageOutput != nil {
			mv := event.Output.MessageOutput
			msg := mv.Message

			if mv.Role == schema.Assistant {
				// Print assistant's text response
				if msg != nil && msg.Content != "" {
					fmt.Fprintln(a.output, msg.Content)
				}
				// Print tool calls in verbose mode
				if a.config.Verbose && msg != nil && len(msg.ToolCalls) > 0 {
					for _, tc := range msg.ToolCalls {
						fmt.Fprintf(a.output, "  Tool: %s(%s)\n", tc.Function.Name, truncate(tc.Function.Arguments, 100))
					}
				}
				if msg != nil && msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
					totalPromptTokens += msg.ResponseMeta.Usage.PromptTokens
					totalCompletionTokens += msg.ResponseMeta.Usage.CompletionTokens
					totalCachedTokens += msg.ResponseMeta.Usage.PromptTokenDetails.CachedTokens
				}
			} else if mv.Role == schema.Tool {
				// Tool response
				if a.config.Verbose && msg != nil {
					fmt.Fprintf(a.output, "  Result: %s\n", truncate(msg.Content, 200))
				}
				if shouldSaveImmediatelyOnMutation(
					a.config.DryRun,
					a.config.StatePath,
					a.state.Metadata.ArchiveOnSave,
					toolCtx.MutationCount,
					lastSavedMutationCount,
				) {
					toolCtx.Mu.Lock()
					if toolCtx.MutationCount > lastSavedMutationCount {
						if err := stateio.Save(a.state, a.config.StatePath); err != nil {
							if a.config.Verbose {
								fmt.Fprintf(a.output, "  Warning: save failed: %v\n", err)
							}
						} else {
							lastSavedMutationCount = toolCtx.MutationCount
							if a.config.Verbose {
								fmt.Fprintf(a.output, "  [auto-saved state]\n")
							}
						}
					}
					toolCtx.Mu.Unlock()
				} else {
					toolCallCount++
					if shouldAttemptPeriodicSave(
						a.config.DryRun,
						a.config.StatePath,
						a.state.Metadata.ArchiveOnSave,
						a.config.SaveInterval,
						toolCallCount,
					) {
						toolCtx.Mu.Lock()
						if err := stateio.Save(a.state, a.config.StatePath); err != nil && a.config.Verbose {
							fmt.Fprintf(a.output, "  Warning: periodic save failed: %v\n", err)
						}
						toolCtx.Mu.Unlock()
						toolCallCount = 0
						if a.config.Verbose {
							fmt.Fprintf(a.output, "  [auto-saved state]\n")
						}
						fmt.Fprintf(a.output, "  [tokens: %d read (%d cached), %d write, %d total]\n",
							totalPromptTokens, totalCachedTokens, totalCompletionTokens,
							totalPromptTokens+totalCompletionTokens)
					}
				}
			}
		}
	}

	fmt.Fprintf(a.output, "\nToken usage: %d read (%d cached), %d write, %d total\n",
		totalPromptTokens, totalCachedTokens, totalCompletionTokens,
		totalPromptTokens+totalCompletionTokens)

	// Print summary of changes in dry-run mode
	if a.config.DryRun && len(toolCtx.Changes) > 0 {
		fmt.Fprintln(a.output, "\n--- Dry Run Summary ---")
		fmt.Fprintf(a.output, "Would make %d changes:\n", len(toolCtx.Changes))
		for _, change := range toolCtx.Changes {
			fmt.Fprintf(a.output, "  [%s] $%04X: %s\n", change.Type, change.Address, truncate(change.Value, 60))
		}
	}

	return nil
}

// buildTaskPrompt creates the task prompt for the user message.
func (a *Agent) buildTaskPrompt() string {
	startAddr := a.config.StartAddr
	endAddr := a.config.EndAddr
	if startAddr == 0 {
		startAddr = a.state.Binary.Start()
	}
	if endAddr == 0 {
		endAddr = a.state.Binary.End()
	}
	return TaskPrompt(startAddr, endAddr, a.state.Metadata.Description)
}

// getModelName returns the model name being used.
func (a *Agent) getModelName() string {
	if a.config.Model != "" {
		return a.config.Model
	}
	switch a.config.Provider {
	case ProviderAnthropic:
		return DefaultAnthropicModel
	case ProviderOpenAI:
		return DefaultOpenAIModel
	case ProviderGemini:
		return DefaultGeminiModel
	default:
		return "unknown"
	}
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	// Remove newlines for display
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
