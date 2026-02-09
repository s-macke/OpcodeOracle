// Package tools provides agent tools for analyzing MOS6502 disassembly.
package tools

import (
	"sync"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/state"

	"github.com/cloudwego/eino/components/tool"
)

// Context holds shared state for all tools.
type Context struct {
	Mu            sync.Mutex
	State         *state.State
	Analyzer      *analysis.Analyzer
	DryRun        bool
	Verbose       bool
	MutationCount uint64

	// Track changes made during dry run
	Changes []Change
}

// Change represents a modification made by the agent.
type Change struct {
	Type    string // "annotation", "headline", "symbol"
	Address uint16
	Value   string
}

// AllTools returns all agent tools configured with the given context.
func AllTools(ctx *Context) []tool.BaseTool {
	return []tool.BaseTool{
		NewViewDisassemblyTool(ctx),
		NewSearchDisassemblyTool(ctx),
		NewAddAnnotationTool(ctx),
		NewAddHeadlineTool(ctx),
		NewAddSymbolTool(ctx),
		NewQuerySymbolsTool(ctx),
		NewQueryXRefsTool(ctx),
		NewListSubroutinesAndDataSegmentsTool(ctx),
		NewGetSubroutineContextTool(ctx),
	}
}
