package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"opcodeoracle/internal/agent/tools"
	"opcodeoracle/internal/stateio"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config holds configuration for the MCP server.
type Config struct {
	StatePath string
	ToolCtx   *tools.Context
}

// New creates a new MCP server with all OpcodeOracle tools registered.
func New(cfg *Config) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "opcodeoracle",
			Version: "0.1.0",
		},
		nil,
	)

	registerTools(server, cfg)
	return server
}

// Run starts the MCP server on stdio transport, blocking until the context is cancelled.
func Run(ctx context.Context, server *mcp.Server) error {
	return server.Run(ctx, &mcp.StdioTransport{})
}

// registerTools adds all 8 OpcodeOracle tools to the MCP server.
func registerTools(server *mcp.Server, cfg *Config) {
	viewDisassembly := tools.NewViewDisassemblyTool(cfg.ToolCtx)
	addAnnotation := tools.NewAddAnnotationTool(cfg.ToolCtx)
	addHeadline := tools.NewAddHeadlineTool(cfg.ToolCtx)
	addSymbol := tools.NewAddSymbolTool(cfg.ToolCtx)
	querySymbols := tools.NewQuerySymbolsTool(cfg.ToolCtx)
	queryXRefs := tools.NewQueryXRefsTool(cfg.ToolCtx)
	listSubroutines := tools.NewListSubroutinesTool(cfg.ToolCtx)
	getSubroutineContext := tools.NewGetSubroutineContextTool(cfg.ToolCtx)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "view_disassembly",
		Description: "View disassembled code at an address range. Shows instructions, existing annotations, headlines, and symbols. Use hex addresses like '$C000' or '0xC000'.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ViewDisassemblyArgs) (*mcp.CallToolResult, any, error) {
		return delegate(ctx, viewDisassembly, args)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_annotation",
		Description: "Add an inline comment (annotation) at a specific address. Annotations appear on the same line as instructions. Use to explain WHY code does something, not just WHAT it does.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args AddAnnotationArgs) (*mcp.CallToolResult, any, error) {
		return delegateAndSave(ctx, addAnnotation, args, cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_headline",
		Description: "Add a block comment (headline) above an address. Headlines appear as separate comment lines before the instruction. Use for subroutine descriptions, section headers, and major code block explanations.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args AddHeadlineArgs) (*mcp.CallToolResult, any, error) {
		return delegateAndSave(ctx, addHeadline, args, cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_symbol",
		Description: "Add or rename a symbol at an address. Symbols provide meaningful names for subroutines, labels, and data locations. Prefer descriptive names over generic ones.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args AddSymbolArgs) (*mcp.CallToolResult, any, error) {
		return delegateAndSave(ctx, addSymbol, args, cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_symbols",
		Description: "Query the symbol table to find existing symbols. Can filter by address range, type, or name pattern.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args QuerySymbolsArgs) (*mcp.CallToolResult, any, error) {
		return delegate(ctx, querySymbols, args)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_xrefs",
		Description: "Query cross-references to or from an address. Shows calls, jumps, branches, and data accesses. Use to understand how code is connected.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args QueryXRefsArgs) (*mcp.CallToolResult, any, error) {
		return delegate(ctx, queryXRefs, args)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_subroutines",
		Description: "List all subroutines (and entry points) in the binary or within an address range. Shows subroutine addresses and names. Use to get an overview of the code structure.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ListSubroutinesArgs) (*mcp.CallToolResult, any, error) {
		return delegate(ctx, listSubroutines, args)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_subroutine_context",
		Description: "Get detailed context for a subroutine including its code, callers, and callees. Useful for understanding a subroutine's purpose before adding documentation.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args GetSubroutineContextArgs) (*mcp.CallToolResult, any, error) {
		return delegate(ctx, getSubroutineContext, args)
	})
}

// invokable is the interface satisfied by all OpcodeOracle tools.
type invokable interface {
	InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einotool.Option) (string, error)
}

// delegate marshals args to JSON, calls InvokableRun, and wraps the result.
func delegate(ctx context.Context, tool invokable, args any) (*mcp.CallToolResult, any, error) {
	jsonBytes, err := json.Marshal(args)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal args: %w", err)
	}

	result, err := tool.InvokableRun(ctx, string(jsonBytes))
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

// delegateAndSave calls delegate then auto-saves state (unless dry-run).
func delegateAndSave(ctx context.Context, tool invokable, args any, cfg *Config) (*mcp.CallToolResult, any, error) {
	result, out, err := delegate(ctx, tool, args)
	if err != nil {
		return result, out, err
	}

	autoSave(cfg)
	return result, out, nil
}

// autoSave persists state to disk after write operations.
func autoSave(cfg *Config) {
	if cfg.ToolCtx.DryRun || cfg.StatePath == "" {
		return
	}
	cfg.ToolCtx.Mu.Lock()
	defer cfg.ToolCtx.Mu.Unlock()
	// Best-effort save; errors are not surfaced to the MCP client.
	_ = stateio.Save(cfg.ToolCtx.State, cfg.StatePath)
}
