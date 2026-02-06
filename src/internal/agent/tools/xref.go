package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// QueryXRefsTool allows querying cross-references.
type QueryXRefsTool struct {
	ctx *Context
}

// NewQueryXRefsTool creates a new query_xrefs tool.
func NewQueryXRefsTool(ctx *Context) *QueryXRefsTool {
	return &QueryXRefsTool{ctx: ctx}
}

type queryXRefsParams struct {
	Address   string `json:"address"`
	Direction string `json:"direction"`
}

// Info returns the tool's metadata.
func (t *QueryXRefsTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "query_xrefs",
		Desc: "Query cross-references to or from an address. Shows calls, jumps, branches, and data accesses. Use to understand how code is connected.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"address": {
				Type:     schema.String,
				Desc:     "Address in hex (e.g., '$C000' or '0xC000')",
				Required: true,
			},
			"direction": {
				Type:     schema.String,
				Desc:     "Direction of references: 'to' (who references this address), 'from' (what this address references), or 'both'",
				Enum:     []string{"to", "from", "both"},
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *QueryXRefsTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params queryXRefsParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	addr, err := parseAddress(params.Address)
	if err != nil {
		return fmt.Sprintf("Error: invalid address: %v", err), nil
	}

	direction := params.Direction
	if direction == "" {
		direction = "both"
	}

	var results []string

	// Get symbol name if available
	symName := ""
	if sym, ok := t.ctx.State.Symbols.At(addr); ok {
		symName = " (" + sym.Name + ")"
	}

	// References TO this address (callers, jumpers, etc.)
	if direction == "to" || direction == "both" {
		refs := t.ctx.State.XRefs.To(addr)
		if len(refs) > 0 {
			results = append(results, fmt.Sprintf("References TO $%04X%s:", addr, symName))
			for _, ref := range refs {
				fromSym := ""
				if sym, ok := t.ctx.State.Symbols.At(ref.From); ok {
					fromSym = " (" + sym.Name + ")"
				}
				results = append(results, fmt.Sprintf("  $%04X%s [%s]", ref.From, fromSym, ref.Type))
			}
		} else if direction == "to" {
			results = append(results, fmt.Sprintf("No references TO $%04X%s", addr, symName))
		}
	}

	// References FROM this address (what it calls, jumps to, etc.)
	if direction == "from" || direction == "both" {
		refs := t.ctx.State.XRefs.From(addr)
		if len(refs) > 0 {
			results = append(results, fmt.Sprintf("References FROM $%04X%s:", addr, symName))
			for _, ref := range refs {
				toSym := ""
				if sym, ok := t.ctx.State.Symbols.At(ref.To); ok {
					toSym = " (" + sym.Name + ")"
				}
				results = append(results, fmt.Sprintf("  $%04X%s [%s]", ref.To, toSym, ref.Type))
			}
		} else if direction == "from" {
			results = append(results, fmt.Sprintf("No references FROM $%04X%s", addr, symName))
		}
	}

	if len(results) == 0 {
		return fmt.Sprintf("No cross-references found for $%04X%s", addr, symName), nil
	}

	return strings.Join(results, "\n"), nil
}

var _ tool.InvokableTool = (*QueryXRefsTool)(nil)
