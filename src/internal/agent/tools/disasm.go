package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"opcodeoracle/internal/disasm"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ViewDisassemblyTool allows viewing disassembled code at an address range.
type ViewDisassemblyTool struct {
	ctx *Context
}

// NewViewDisassemblyTool creates a new view_disassembly tool.
func NewViewDisassemblyTool(ctx *Context) *ViewDisassemblyTool {
	return &ViewDisassemblyTool{ctx: ctx}
}

type viewDisassemblyParams struct {
	StartAddr string `json:"start_addr"`
	EndAddr   string `json:"end_addr"`
}

// Info returns the tool's metadata.
func (t *ViewDisassemblyTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "view_disassembly",
		Desc: "View disassembled code at an address range. Shows instructions, existing annotations, headlines, and symbols. Use hex addresses like '$C000' or '0xC000'.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"start_addr": {
				Type:     schema.String,
				Desc:     "Start address in hex (e.g., '$C000' or '0xC000')",
				Required: true,
			},
			"end_addr": {
				Type:     schema.String,
				Desc:     "End address in hex (inclusive). If not provided, shows ~32 bytes from start.",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *ViewDisassemblyTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params viewDisassemblyParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	startAddr, err := parseAddress(params.StartAddr)
	if err != nil {
		return fmt.Sprintf("Error: invalid start_addr: %v", err), nil
	}

	// Default end address to start + 32 bytes
	endAddr := startAddr + 32
	if startAddr > 0xFFFF-32 {
		endAddr = 0xFFFF
	}
	if params.EndAddr != "" {
		endAddr, err = parseAddress(params.EndAddr)
		if err != nil {
			return fmt.Sprintf("Error: invalid end_addr: %v", err), nil
		}
	}

	if startAddr > endAddr {
		return fmt.Sprintf("Error: start address ($%04X) is greater than end address ($%04X)", startAddr, endAddr), nil
	}

	// Create disassembler and generate output
	d := disasm.NewDisassembler(t.ctx.State, t.ctx.Analyzer)
	// +1 because Disassemble uses exclusive end (guard uint16 overflow)
	exclusiveEnd := endAddr + 1
	if endAddr == 0xFFFF {
		exclusiveEnd = 0xFFFF
	}
	output, err := d.Disassemble(startAddr, exclusiveEnd)
	if err != nil {
		return fmt.Sprintf("Error: disassembly failed: %v", err), nil
	}

	return output, nil
}

var _ tool.InvokableTool = (*ViewDisassemblyTool)(nil)
