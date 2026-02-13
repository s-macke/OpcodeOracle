package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"opcodeoracle/internal/author"
	"opcodeoracle/internal/numparse"

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
	Extend  bool   `json:"extend"`
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
			"extend": {
				Type:     schema.Boolean,
				Desc:     "Append to existing assistant headline instead of replacing (optional)",
				Required: false,
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

	addr, err := numparse.ParseUint16(params.Address)
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
		if params.Extend {
			return fmt.Sprintf("Would extend headline at $%04X: %s", addr, params.Comment), nil
		}
		return fmt.Sprintf("Would add headline at $%04X: %s", addr, params.Comment), nil
	}

	if params.Extend {
		t.ctx.State.Headlines.Extend(addr, params.Comment, author.Assistant)
		t.ctx.MutationCount++
		return fmt.Sprintf("Extended headline at $%04X: %s", addr, params.Comment), nil
	}

	t.ctx.State.Headlines.Set(addr, params.Comment, author.Assistant)
	t.ctx.MutationCount++
	return fmt.Sprintf("Added headline at $%04X: %s", addr, params.Comment), nil
}

var _ tool.InvokableTool = (*AddHeadlineTool)(nil)

// RemoveHeadlineTool allows removing block comments (headlines).
type RemoveHeadlineTool struct {
	ctx *Context
}

// NewRemoveHeadlineTool creates a new remove_headline tool.
func NewRemoveHeadlineTool(ctx *Context) *RemoveHeadlineTool {
	return &RemoveHeadlineTool{ctx: ctx}
}

type removeHeadlineParams struct {
	Address string `json:"address"`
	Author  string `json:"author"`
}

// Info returns the tool's metadata.
func (t *RemoveHeadlineTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "remove_headline",
		Desc: "Remove a block comment (headline) at a specific address. By default removes assistant-authored headline; set author to remove user-authored headline.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"address": {
				Type:     schema.String,
				Desc:     "Address in hex (e.g., '$C000' or '0xC000')",
				Required: true,
			},
			"author": {
				Type:     schema.String,
				Desc:     "Headline author to remove (optional)",
				Enum:     []string{"assistant", "user"},
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *RemoveHeadlineTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params removeHeadlineParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	addr, err := numparse.ParseUint16(params.Address)
	if err != nil {
		return fmt.Sprintf("Error: invalid address: %v", err), nil
	}

	authorStr := params.Author
	if authorStr == "" {
		authorStr = author.Assistant.String()
	}
	a, err := author.Parse(authorStr)
	if err != nil {
		return fmt.Sprintf("Error: invalid author: %v", err), nil
	}

	existing := t.ctx.State.Headlines.Get(addr, a)
	if existing == nil {
		return fmt.Sprintf("No headline found at $%04X for author %s", addr, a), nil
	}

	if t.ctx.DryRun {
		t.ctx.Changes = append(t.ctx.Changes, Change{
			Type:    "headline",
			Address: addr,
			Value:   fmt.Sprintf("remove (%s)", a),
		})
		return fmt.Sprintf("Would remove headline at $%04X (author: %s)", addr, a), nil
	}

	t.ctx.State.Headlines.Remove(addr, a)
	t.ctx.MutationCount++
	return fmt.Sprintf("Removed headline at $%04X (author: %s)", addr, a), nil
}

var _ tool.InvokableTool = (*RemoveHeadlineTool)(nil)
