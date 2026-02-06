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

// AddHeadlineTool allows adding block comments (headlines) above addresses.
type AddHeadlineTool struct {
	ctx *Context
}

// NewAddHeadlineTool creates a new add_headline tool.
func NewAddHeadlineTool(ctx *Context) *AddHeadlineTool {
	return &AddHeadlineTool{ctx: ctx}
}

type addHeadlineParams struct {
	Address string `json:"address"`
	Comment string `json:"comment"`
}

// Info returns the tool's metadata.
func (t *AddHeadlineTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "add_headline",
		Desc: "Add a block comment (headline) above an address. Headlines appear as separate comment lines before the instruction. Use for subroutine descriptions, section headers, and major code block explanations.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"address": {
				Type:     schema.String,
				Desc:     "Address in hex (e.g., '$C000' or '0xC000') - typically a subroutine entry point",
				Required: true,
			},
			"comment": {
				Type:     schema.String,
				Desc:     "The headline text describing the subroutine or section (can be multi-line)",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *AddHeadlineTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params addHeadlineParams
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
			Type:    "headline",
			Address: addr,
			Value:   params.Comment,
		})
		return fmt.Sprintf("Would add headline at $%04X: %s", addr, params.Comment), nil
	}

	t.ctx.State.Headlines.Set(addr, params.Comment, author.Assistant)
	return fmt.Sprintf("Added headline at $%04X: %s", addr, params.Comment), nil
}

var _ tool.InvokableTool = (*AddHeadlineTool)(nil)
