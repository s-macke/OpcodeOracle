package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"opcodeoracle/internal/numparse"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/reinterpret"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type reinterpretAsCodeParams struct {
	CodeAddress string `json:"code_address"`
}

type reinterpretAsDataParams struct {
	StartAddr string `json:"start_addr"`
	EndAddr   string `json:"end_addr"`
}

// ReinterpretAsCodeTool forces a single address as code seed and reruns analysis.
type ReinterpretAsCodeTool struct {
	ctx *Context
}

// NewReinterpretAsCodeTool creates a new reinterpret_as_code tool.
func NewReinterpretAsCodeTool(ctx *Context) *ReinterpretAsCodeTool {
	return &ReinterpretAsCodeTool{ctx: ctx}
}

// Info returns the tool's metadata.
func (t *ReinterpretAsCodeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "reinterpret_as_code",
		Desc: "Force reinterpretation of one address as code seed, then rerun analysis from scratch to rebuild regions, symbols, xrefs, and CFG.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"code_address": {
				Type:     schema.String,
				Desc:     "Single address to force as code seed",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *ReinterpretAsCodeTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params reinterpretAsCodeParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	addr, err := numparse.ParseUint16(params.CodeAddress)
	if err != nil {
		return fmt.Sprintf("Error: invalid code_address: %v", err), nil
	}
	if t.ctx.DryRun {
		return fmt.Sprintf("Would reinterpret $%04X as code and rerun full analysis", addr), nil
	}

	analyzer, err := reinterpret.AsCode(t.ctx.State, addr, regions.RegionSourceAssistant)
	if err != nil {
		return fmt.Sprintf("Error: reinterpretation failed: %v", err), nil
	}
	t.ctx.Analyzer = analyzer
	t.ctx.MutationCount++
	return fmt.Sprintf("Reinterpreted $%04X as code; rebuilt CFG with %d instructions",
		addr, len(analyzer.InstructionAddresses())), nil
}

var _ tool.InvokableTool = (*ReinterpretAsCodeTool)(nil)

// ReinterpretAsDataTool hard-locks a range as data and reruns analysis.
type ReinterpretAsDataTool struct {
	ctx *Context
}

// NewReinterpretAsDataTool creates a new reinterpret_as_data tool.
func NewReinterpretAsDataTool(ctx *Context) *ReinterpretAsDataTool {
	return &ReinterpretAsDataTool{ctx: ctx}
}

// Info returns the tool's metadata.
func (t *ReinterpretAsDataTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "reinterpret_as_data",
		Desc: "Force reinterpretation of a range as hard-locked data, then rerun analysis from scratch to rebuild regions, symbols, xrefs, and CFG.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"start_addr": {
				Type:     schema.String,
				Desc:     "Range start to force as hard-locked data",
				Required: true,
			},
			"end_addr": {
				Type:     schema.String,
				Desc:     "Range end to force as hard-locked data",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *ReinterpretAsDataTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params reinterpretAsDataParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	start, err := numparse.ParseUint16(params.StartAddr)
	if err != nil {
		return fmt.Sprintf("Error: invalid start_addr: %v", err), nil
	}
	end, err := numparse.ParseUint16(params.EndAddr)
	if err != nil {
		return fmt.Sprintf("Error: invalid end_addr: %v", err), nil
	}
	if t.ctx.DryRun {
		return fmt.Sprintf("Would reinterpret $%04X-$%04X as hard-locked data and rerun full analysis", start, end), nil
	}

	analyzer, err := reinterpret.AsData(t.ctx.State, start, end, regions.RegionSourceAssistant)
	if err != nil {
		return fmt.Sprintf("Error: reinterpretation failed: %v", err), nil
	}
	t.ctx.Analyzer = analyzer
	t.ctx.MutationCount++
	return fmt.Sprintf("Reinterpreted $%04X-$%04X as hard-locked data; rebuilt CFG with %d instructions",
		start, end, len(analyzer.InstructionAddresses())), nil
}

var _ tool.InvokableTool = (*ReinterpretAsDataTool)(nil)
