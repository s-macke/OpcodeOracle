package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"opcodeoracle/internal/author"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// AddAnnotationTool allows adding inline annotations to the disassembly.
type AddAnnotationTool struct {
	ctx *Context
}

// NewAddAnnotationTool creates a new add_annotation tool.
func NewAddAnnotationTool(ctx *Context) *AddAnnotationTool {
	return &AddAnnotationTool{ctx: ctx}
}

type addAnnotationParams struct {
	Address string `json:"address"`
	Comment string `json:"comment"`
}

// Info returns the tool's metadata.
func (t *AddAnnotationTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "add_annotation",
		Desc: "Add an inline comment (annotation) at a specific address. Annotations appear on the same line as instructions. Use to explain WHY code does something, not just WHAT it does.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"address": {
				Type:     schema.String,
				Desc:     "Address in hex (e.g., '$C000' or '0xC000')",
				Required: true,
			},
			"comment": {
				Type:     schema.String,
				Desc:     "The annotation text explaining the instruction's purpose",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *AddAnnotationTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params addAnnotationParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	addr, err := parseAddress(params.Address)
	if err != nil {
		return fmt.Sprintf("Error: invalid address: %v", err), nil
	}

	if params.Comment == "" {
		return "Error: comment cannot be empty", nil
	}

	params.Comment = strings.ReplaceAll(params.Comment, `\n`, "\n")

	// Validate address is within binary bounds
	if addr < t.ctx.State.Binary.Start() || addr > t.ctx.State.Binary.End() {
		return fmt.Sprintf("Error: address $%04X is outside binary bounds ($%04X-$%04X)",
			addr, t.ctx.State.Binary.Start(), t.ctx.State.Binary.End()), nil
	}

	if t.ctx.DryRun {
		t.ctx.Changes = append(t.ctx.Changes, Change{
			Type:    "annotation",
			Address: addr,
			Value:   params.Comment,
		})
		return fmt.Sprintf("Would add annotation at $%04X: %s", addr, params.Comment), nil
	}

	t.ctx.State.Annotations.Set(addr, params.Comment, author.Assistant)
	t.ctx.MutationCount++
	return fmt.Sprintf("Added annotation at $%04X: %s", addr, params.Comment), nil
}

var _ tool.InvokableTool = (*AddAnnotationTool)(nil)
