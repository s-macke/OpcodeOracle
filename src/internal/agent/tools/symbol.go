package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"opcodeoracle/internal/symbols"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// AddSymbolTool allows adding or renaming symbols at addresses.
type AddSymbolTool struct {
	ctx *Context
}

// NewAddSymbolTool creates a new add_symbol tool.
func NewAddSymbolTool(ctx *Context) *AddSymbolTool {
	return &AddSymbolTool{ctx: ctx}
}

type addSymbolParams struct {
	Address    string `json:"address"`
	Name       string `json:"name"`
	SymbolType string `json:"symbol_type"`
}

// Info returns the tool's metadata.
func (t *AddSymbolTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "add_symbol",
		Desc: "Add or rename a symbol at an address. Symbols provide meaningful names for subroutines, labels, and data locations. Prefer descriptive names over generic ones.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"address": {
				Type:     schema.String,
				Desc:     "Address in hex (e.g., '$C000' or '0xC000')",
				Required: true,
			},
			"name": {
				Type:     schema.String,
				Desc:     "Symbol name (e.g., 'init_screen', 'sprite_x', 'VIC_CTRL1')",
				Required: true,
			},
			"symbol_type": {
				Type:     schema.String,
				Desc:     "Type of symbol",
				Enum:     []string{"subroutine", "label", "byte", "word"},
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *AddSymbolTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params addSymbolParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	addr, err := parseAddress(params.Address)
	if err != nil {
		return fmt.Sprintf("Error: invalid address: %v", err), nil
	}

	if params.Name == "" {
		return "Error: name cannot be empty", nil
	}

	// Determine symbol type
	symType := symbols.SymbolLabel
	switch strings.ToLower(params.SymbolType) {
	case "subroutine", "sub":
		symType = symbols.SymbolSubroutine
	case "label", "":
		symType = symbols.SymbolLabel
	case "byte":
		symType = symbols.SymbolByte
	case "word":
		symType = symbols.SymbolWord
	default:
		return fmt.Sprintf("Error: invalid symbol_type: %s (must be subroutine, label, byte, or word)", params.SymbolType), nil
	}

	if t.ctx.DryRun {
		t.ctx.Changes = append(t.ctx.Changes, Change{
			Type:    "symbol",
			Address: addr,
			Value:   fmt.Sprintf("%s (%s)", params.Name, symType),
		})
		return fmt.Sprintf("Would add symbol at $%04X: %s (%s)", addr, params.Name, symType), nil
	}

	if err := t.ctx.State.Symbols.Add(addr, symbols.Symbol{
		Name:   params.Name,
		Type:   symType,
		Source: symbols.SourceAssistant,
	}); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	t.ctx.MutationCount++

	return fmt.Sprintf("Added symbol at $%04X: %s (%s)", addr, params.Name, symType), nil
}

var _ tool.InvokableTool = (*AddSymbolTool)(nil)

// QuerySymbolsTool allows querying the symbol table.
type QuerySymbolsTool struct {
	ctx *Context
}

// NewQuerySymbolsTool creates a new query_symbols tool.
func NewQuerySymbolsTool(ctx *Context) *QuerySymbolsTool {
	return &QuerySymbolsTool{ctx: ctx}
}

type querySymbolsParams struct {
	StartAddr  string `json:"start_addr"`
	EndAddr    string `json:"end_addr"`
	SymbolType string `json:"symbol_type"`
	NameFilter string `json:"name_filter"`
}

// Info returns the tool's metadata.
func (t *QuerySymbolsTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "query_symbols",
		Desc: "Query the symbol table to find existing symbols. Can filter by address range, type, or name pattern.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"start_addr": {
				Type:     schema.String,
				Desc:     "Start address for range filter (optional)",
				Required: false,
			},
			"end_addr": {
				Type:     schema.String,
				Desc:     "End address for range filter (optional)",
				Required: false,
			},
			"symbol_type": {
				Type:     schema.String,
				Desc:     "Filter by symbol type (optional)",
				Enum:     []string{"subroutine", "label", "byte", "word", "entry"},
				Required: false,
			},
			"name_filter": {
				Type:     schema.String,
				Desc:     "Filter symbols containing this substring (case-insensitive, optional)",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *QuerySymbolsTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params querySymbolsParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	// Parse address filters
	var startAddr, endAddr uint16
	var hasRange bool
	if params.StartAddr != "" {
		var err error
		startAddr, err = parseAddress(params.StartAddr)
		if err != nil {
			return fmt.Sprintf("Error: invalid start_addr: %v", err), nil
		}
		hasRange = true
	}
	if params.EndAddr != "" {
		var err error
		endAddr, err = parseAddress(params.EndAddr)
		if err != nil {
			return fmt.Sprintf("Error: invalid end_addr: %v", err), nil
		}
		hasRange = true
	} else if hasRange {
		endAddr = 0xFFFF
	}

	// Parse type filter
	var typeFilter symbols.SymbolType
	if params.SymbolType != "" {
		switch strings.ToLower(params.SymbolType) {
		case "subroutine":
			typeFilter = symbols.SymbolSubroutine
		case "label":
			typeFilter = symbols.SymbolLabel
		case "byte":
			typeFilter = symbols.SymbolByte
		case "word":
			typeFilter = symbols.SymbolWord
		case "entry":
			typeFilter = symbols.SymbolEntry
		}
	}

	nameFilter := strings.ToLower(params.NameFilter)

	// Collect matching symbols
	var results []string
	allSymbols := t.ctx.State.Symbols.All()

	// Sort by address
	type addrSym struct {
		addr uint16
		sym  symbols.Symbol
	}
	var sorted []addrSym
	for addr, sym := range allSymbols {
		sorted = append(sorted, addrSym{addr, sym})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].addr < sorted[j].addr
	})

	for _, as := range sorted {
		addr, sym := as.addr, as.sym

		// Apply filters
		if hasRange && (addr < startAddr || addr > endAddr) {
			continue
		}
		if typeFilter != "" && sym.Type != typeFilter {
			continue
		}
		if nameFilter != "" && !strings.Contains(strings.ToLower(sym.Name), nameFilter) {
			continue
		}

		results = append(results, fmt.Sprintf("$%04X: %s (%s, %s)", addr, sym.Name, sym.Type, sym.Source))
	}

	if len(results) == 0 {
		return "No symbols found matching the criteria.", nil
	}

	return fmt.Sprintf("Found %d symbols:\n%s", len(results), strings.Join(results, "\n")), nil
}

var _ tool.InvokableTool = (*QuerySymbolsTool)(nil)
