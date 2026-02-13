package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"opcodeoracle/internal/agent/tools"
	"opcodeoracle/internal/stateio"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config holds configuration for the MCP server.
type Config struct {
	StatePath string
	ToolCtx   *tools.Context
	Output    io.Writer
	Verbose   bool

	logMu sync.Mutex
}

// Transport selects the MCP transport implementation.
type Transport string

const (
	TransportStdio Transport = "stdio"
	TransportHTTP  Transport = "http"
)

// RunOptions configures how the MCP server is exposed.
type RunOptions struct {
	Transport  Transport
	ListenAddr string
	Path       string
}

// New creates a new MCP server with all OpcodeOracle tools registered.
func New(cfg *Config) *mcp.Server {
	initConfig(cfg)

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

// Run starts the MCP server using the configured transport.
func Run(ctx context.Context, server *mcp.Server, cfg *Config, opts RunOptions) error {
	initConfig(cfg)

	switch opts.Transport {
	case "", TransportStdio:
		logInfo(cfg, "starting MCP server (transport=stdio)")
		err := server.Run(ctx, &mcp.StdioTransport{})
		if err != nil {
			logWarn(cfg, "MCP server exited with error: %v", err)
			return err
		}
		logInfo(cfg, "MCP server stopped")
		return nil
	case TransportHTTP:
		if strings.TrimSpace(opts.ListenAddr) == "" {
			return errors.New("http transport requires a non-empty listen address")
		}
		if opts.Path == "" {
			return errors.New("http transport requires a non-empty path")
		}
		if !strings.HasPrefix(opts.Path, "/") {
			return errors.New("http transport path must start with '/'")
		}

		mux := http.NewServeMux()
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		mux.Handle(opts.Path, handler)

		httpServer := &http.Server{
			Addr:    opts.ListenAddr,
			Handler: mux,
		}

		stopShutdown := context.AfterFunc(ctx, func() {
			logInfo(cfg, "shutdown requested; stopping HTTP server")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = httpServer.Shutdown(shutdownCtx)
		})
		defer stopShutdown()

		logInfo(cfg, "starting MCP server (transport=http listen=%s path=%s)", opts.ListenAddr, opts.Path)
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) && ctx.Err() != nil {
			logInfo(cfg, "HTTP server stopped")
			return nil
		}
		if err != nil {
			logWarn(cfg, "HTTP server exited with error: %v", err)
		}
		return err
	default:
		logWarn(cfg, "invalid MCP transport: %q", opts.Transport)
		return fmt.Errorf("unknown transport %q", opts.Transport)
	}
}

// registerTools adds all OpcodeOracle tools to the MCP server.
func registerTools(server *mcp.Server, cfg *Config) {
	viewDisassembly := tools.NewViewDisassemblyTool(cfg.ToolCtx)
	searchDisassembly := tools.NewSearchDisassemblyTool(cfg.ToolCtx)
	addAnnotation := tools.NewAddAnnotationTool(cfg.ToolCtx)
	addHeadline := tools.NewAddHeadlineTool(cfg.ToolCtx)
	addSymbol := tools.NewAddSymbolTool(cfg.ToolCtx)
	querySymbols := tools.NewQuerySymbolsTool(cfg.ToolCtx)
	queryXRefs := tools.NewQueryXRefsTool(cfg.ToolCtx)
	reinterpretAsCode := tools.NewReinterpretAsCodeTool(cfg.ToolCtx)
	reinterpretAsData := tools.NewReinterpretAsDataTool(cfg.ToolCtx)
	listSubroutinesAndDataSegments := tools.NewListSubroutinesAndDataSegmentsTool(cfg.ToolCtx)
	getSubroutineContext := tools.NewGetSubroutineContextTool(cfg.ToolCtx)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "view_disassembly",
		Description: "View disassembled code at an address range. Shows instructions, existing annotations, headlines, and symbols. Use hex addresses like '$C000' or '0xC000'.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ViewDisassemblyArgs) (*mcp.CallToolResult, any, error) {
		return delegate(ctx, viewDisassembly, args, "view_disassembly", cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_disassembly",
		Description: "Search disassembly text for a query string. Returns match addresses, matching lines, and nearby context. Useful for finding opcodes, symbols, comments, and patterns quickly.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args SearchDisassemblyArgs) (*mcp.CallToolResult, any, error) {
		return delegate(ctx, searchDisassembly, args, "search_disassembly", cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_annotation",
		Description: "Add an inline comment (annotation) at a specific address. Annotations appear on the same line as instructions. Use to explain WHY code does something, not just WHAT it does.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args AddAnnotationArgs) (*mcp.CallToolResult, any, error) {
		return delegateAndSave(ctx, addAnnotation, args, "add_annotation", cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_headline",
		Description: "Add a block comment (headline) above an address. Headlines appear as separate comment lines before the instruction. Use for subroutine descriptions, section headers, and major code block explanations.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args AddHeadlineArgs) (*mcp.CallToolResult, any, error) {
		return delegateAndSave(ctx, addHeadline, args, "add_headline", cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_symbol",
		Description: "Add or rename a symbol at an address. Symbols provide meaningful names for subroutines, labels, and data locations. Prefer descriptive names over generic ones.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args AddSymbolArgs) (*mcp.CallToolResult, any, error) {
		return delegateAndSave(ctx, addSymbol, args, "add_symbol", cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_symbols",
		Description: "Query the symbol table to find existing symbols. Can filter by address range, type, or name pattern.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args QuerySymbolsArgs) (*mcp.CallToolResult, any, error) {
		return delegate(ctx, querySymbols, args, "query_symbols", cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_xrefs",
		Description: "Query cross-references to or from an address. Shows calls, jumps, branches, and data accesses. Use to understand how code is connected.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args QueryXRefsArgs) (*mcp.CallToolResult, any, error) {
		return delegate(ctx, queryXRefs, args, "query_xrefs", cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "reinterpret_as_code",
		Description: "Force reinterpretation of one address as code seed, then rerun analysis from scratch to rebuild regions, symbols, xrefs, and CFG.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ReinterpretAsCodeArgs) (*mcp.CallToolResult, any, error) {
		return delegateAndSave(ctx, reinterpretAsCode, args, "reinterpret_as_code", cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "reinterpret_as_data",
		Description: "Force reinterpretation of a range as hard-locked data, then rerun analysis from scratch to rebuild regions, symbols, xrefs, and CFG.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ReinterpretAsDataArgs) (*mcp.CallToolResult, any, error) {
		return delegateAndSave(ctx, reinterpretAsData, args, "reinterpret_as_data", cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_subroutines_and_data_segments",
		Description: "List subroutine/entry, code, and data segments in the binary or within an address range. Subroutine rows include names and caller counts for quick code-structure overview.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ListSubroutinesAndDataSegmentsArgs) (*mcp.CallToolResult, any, error) {
		return delegate(ctx, listSubroutinesAndDataSegments, args, "list_subroutines_and_data_segments", cfg)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_subroutine_context",
		Description: "Get detailed context for a subroutine including its code, callers, and callees. Useful for understanding a subroutine's purpose before adding documentation.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args GetSubroutineContextArgs) (*mcp.CallToolResult, any, error) {
		return delegate(ctx, getSubroutineContext, args, "get_subroutine_context", cfg)
	})
}

// invokable is the interface satisfied by all OpcodeOracle tools.
type invokable interface {
	InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einotool.Option) (string, error)
}

// delegate marshals args to JSON, calls InvokableRun, and wraps the result.
func delegate(ctx context.Context, tool invokable, args any, toolName string, cfg *Config) (*mcp.CallToolResult, any, error) {
	start := time.Now()

	jsonBytes, err := json.Marshal(args)
	if err != nil {
		logWarn(cfg, "tool %s failed to marshal args: %v", toolName, err)
		return nil, nil, fmt.Errorf("marshal args: %w", err)
	}
	logVerbose(cfg, "tool %s args=%s", toolName, truncateOneLine(string(jsonBytes), 300))

	result, err := tool.InvokableRun(ctx, string(jsonBytes))
	if err != nil {
		logWarn(cfg, "tool %s failed (%s): %v", toolName, time.Since(start).Round(time.Millisecond), err)
		return nil, nil, err
	}
	logInfo(cfg, "tool %s ok (%s)", toolName, time.Since(start).Round(time.Millisecond))
	logVerbose(cfg, "tool %s result=%s", toolName, truncateOneLine(result, 300))

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: result}},
	}, nil, nil
}

// delegateAndSave calls delegate then auto-saves state (unless dry-run).
func delegateAndSave(ctx context.Context, tool invokable, args any, toolName string, cfg *Config) (*mcp.CallToolResult, any, error) {
	result, out, err := delegate(ctx, tool, args, toolName, cfg)
	if err != nil {
		return result, out, err
	}

	autoSaveStatus, autoSaveErr := autoSave(cfg)
	switch autoSaveStatus {
	case autoSaveSkippedDryRun:
		logVerbose(cfg, "auto-save skipped after %s: dry-run enabled", toolName)
	case autoSaveSkippedNoPath:
		logVerbose(cfg, "auto-save skipped after %s: empty state path", toolName)
	case autoSaveSaved:
		logVerbose(cfg, "auto-saved state after %s to %s", toolName, cfg.StatePath)
	case autoSaveFailed:
		logWarn(cfg, "auto-save failed after %s: %v", toolName, autoSaveErr)
	}

	return result, out, nil
}

// autoSave persists state to disk after write operations.
func autoSave(cfg *Config) (autoSaveResult, error) {
	if cfg.ToolCtx.DryRun || cfg.StatePath == "" {
		if cfg.ToolCtx.DryRun {
			return autoSaveSkippedDryRun, nil
		}
		return autoSaveSkippedNoPath, nil
	}
	cfg.ToolCtx.Mu.Lock()
	defer cfg.ToolCtx.Mu.Unlock()
	// Best-effort save; errors are not surfaced to the MCP client.
	if err := stateio.Save(cfg.ToolCtx.State, cfg.StatePath); err != nil {
		return autoSaveFailed, err
	}
	return autoSaveSaved, nil
}

type autoSaveResult string

const (
	autoSaveSaved         autoSaveResult = "saved"
	autoSaveSkippedDryRun autoSaveResult = "skipped_dry_run"
	autoSaveSkippedNoPath autoSaveResult = "skipped_no_path"
	autoSaveFailed        autoSaveResult = "failed"
)

func initConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}
}

func logInfo(cfg *Config, format string, args ...any) {
	logf(cfg, format, args...)
}

func logWarn(cfg *Config, format string, args ...any) {
	logf(cfg, "warning: "+format, args...)
}

func logVerbose(cfg *Config, format string, args ...any) {
	if cfg == nil || !cfg.Verbose {
		return
	}
	logf(cfg, format, args...)
}

func logf(cfg *Config, format string, args ...any) {
	if cfg == nil {
		return
	}
	initConfig(cfg)

	cfg.logMu.Lock()
	defer cfg.logMu.Unlock()
	fmt.Fprintf(cfg.Output, "[mcp] "+format+"\n", args...)
}

func truncateOneLine(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.TrimSpace(s)
	if maxLen <= 3 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
