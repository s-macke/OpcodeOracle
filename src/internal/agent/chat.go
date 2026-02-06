package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"opcodeoracle/internal/agent/tools"
	"opcodeoracle/internal/stateio"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// Chat runs an interactive REPL for conversing with the agent about a loaded binary.
func (a *Agent) Chat(ctx context.Context, input io.Reader) error {
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
		MaxIterations: 200,
	})
	if err != nil {
		return fmt.Errorf("failed to create ADK agent: %w", err)
	}

	// Build run options (enable prompt caching for Anthropic)
	var runOpts []adk.AgentRunOption
	if a.config.Provider == ProviderAnthropic {
		runOpts = append(runOpts, adk.WithChatModelOptions([]model.Option{
			claude.WithEnableAutoCache(true),
		}))
	}

	// Welcome banner
	fmt.Fprintf(a.output, "OpcodeOracle chat (%s/%s)\n", a.config.Provider, a.getModelName())
	fmt.Fprintf(a.output, "Binary: $%04X-$%04X\n", a.state.Binary.Start(), a.state.Binary.End())
	fmt.Fprintf(a.output, "Type your questions. Press Ctrl+D to exit.\n\n")

	// Session token totals
	var sessionPromptTokens, sessionCompletionTokens, sessionCachedTokens int

	// Conversation history
	var history []*schema.Message

	// REPL loop
	scanner := bufio.NewScanner(input)
	for {
		fmt.Fprint(a.output, "you> ")
		if !scanner.Scan() {
			break // EOF (Ctrl+D)
		}
		userInput := scanner.Text()
		if userInput == "" {
			continue
		}

		// Append user message to history
		history = append(history, schema.UserMessage(userInput))

		// Shallow-copy history to prevent ADK from mutating our backing array
		inputMessages := make([]*schema.Message, len(history))
		copy(inputMessages, history)

		agentInput := &adk.AgentInput{
			Messages: inputMessages,
		}

		// Run the agent for this turn
		var turnMessages []*schema.Message
		var turnPromptTokens, turnCompletionTokens, turnCachedTokens int

		iter := adkAgent.Run(ctx, agentInput, runOpts...)
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				fmt.Fprintf(a.output, "error: %v\n", event.Err)
				break
			}

			if event.Output != nil && event.Output.MessageOutput != nil {
				mv := event.Output.MessageOutput
				msg := mv.Message

				// Collect all messages for history
				if msg != nil {
					turnMessages = append(turnMessages, msg)
				}

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
					// Track token usage
					if msg != nil && msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
						turnPromptTokens += msg.ResponseMeta.Usage.PromptTokens
						turnCompletionTokens += msg.ResponseMeta.Usage.CompletionTokens
						turnCachedTokens += msg.ResponseMeta.Usage.PromptTokenDetails.CachedTokens
					}
				} else if mv.Role == schema.Tool {
					if a.config.Verbose && msg != nil {
						fmt.Fprintf(a.output, "  Result: %s\n", truncate(msg.Content, 200))
					}
				}
			}
		}

		// Append turn messages to history
		history = append(history, turnMessages...)

		// Print per-turn token usage
		sessionPromptTokens += turnPromptTokens
		sessionCompletionTokens += turnCompletionTokens
		sessionCachedTokens += turnCachedTokens
		fmt.Fprintf(a.output, "[tokens: %d read (%d cached), %d write]\n\n",
			turnPromptTokens, turnCachedTokens, turnCompletionTokens)

		// Save state after each turn (unless dry-run)
		if !a.config.DryRun && a.config.StatePath != "" {
			toolCtx.Mu.Lock()
			if err := stateio.Save(a.state, a.config.StatePath); err != nil && a.config.Verbose {
				fmt.Fprintf(a.output, "  Warning: save failed: %v\n", err)
			}
			toolCtx.Mu.Unlock()
		}
	}

	// Print session totals
	fmt.Fprintf(a.output, "\nSession token usage: %d read (%d cached), %d write, %d total\n",
		sessionPromptTokens, sessionCachedTokens, sessionCompletionTokens,
		sessionPromptTokens+sessionCompletionTokens)

	// Print dry-run summary if applicable
	if a.config.DryRun && len(toolCtx.Changes) > 0 {
		fmt.Fprintln(a.output, "\n--- Dry Run Summary ---")
		fmt.Fprintf(a.output, "Would make %d changes:\n", len(toolCtx.Changes))
		for _, change := range toolCtx.Changes {
			fmt.Fprintf(a.output, "  [%s] $%04X: %s\n", change.Type, change.Address, truncate(change.Value, 60))
		}
	}

	return nil
}
