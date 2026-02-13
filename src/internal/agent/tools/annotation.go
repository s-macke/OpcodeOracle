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
	Extend  bool   `json:"extend"`
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
			"extend": {
				Type:     schema.Boolean,
				Desc:     "Append to existing assistant annotation instead of replacing (optional)",
				Required: false,
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
			Type:    "annotation",
			Address: addr,
			Value:   params.Comment,
		})
		if params.Extend {
			return fmt.Sprintf("Would extend annotation at $%04X: %s", addr, params.Comment), nil
		}
		return fmt.Sprintf("Would add annotation at $%04X: %s", addr, params.Comment), nil
	}

	if params.Extend {
		t.ctx.State.Annotations.Extend(addr, params.Comment, author.Assistant)
		t.ctx.MutationCount++
		return fmt.Sprintf("Extended annotation at $%04X: %s", addr, params.Comment), nil
	}

	t.ctx.State.Annotations.Set(addr, params.Comment, author.Assistant)
	t.ctx.MutationCount++
	return fmt.Sprintf("Added annotation at $%04X: %s", addr, params.Comment), nil
}

var _ tool.InvokableTool = (*AddAnnotationTool)(nil)

// RemoveAnnotationTool allows removing inline annotations.
type RemoveAnnotationTool struct {
	ctx *Context
}

// NewRemoveAnnotationTool creates a new remove_annotation tool.
func NewRemoveAnnotationTool(ctx *Context) *RemoveAnnotationTool {
	return &RemoveAnnotationTool{ctx: ctx}
}

type removeAnnotationParams struct {
	Address string `json:"address"`
	Author  string `json:"author"`
}

// Info returns the tool's metadata.
func (t *RemoveAnnotationTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "remove_annotation",
		Desc: "Remove an inline annotation at a specific address. By default removes assistant-authored annotation; set author to remove user-authored annotation.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"address": {
				Type:     schema.String,
				Desc:     "Address in hex (e.g., '$C000' or '0xC000')",
				Required: true,
			},
			"author": {
				Type:     schema.String,
				Desc:     "Annotation author to remove (optional)",
				Enum:     []string{"assistant", "user"},
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *RemoveAnnotationTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params removeAnnotationParams
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

	existing := t.ctx.State.Annotations.Get(addr, a)
	if existing == nil {
		return fmt.Sprintf("No annotation found at $%04X for author %s", addr, a), nil
	}

	if t.ctx.DryRun {
		t.ctx.Changes = append(t.ctx.Changes, Change{
			Type:    "annotation",
			Address: addr,
			Value:   fmt.Sprintf("remove (%s)", a),
		})
		return fmt.Sprintf("Would remove annotation at $%04X (author: %s)", addr, a), nil
	}

	t.ctx.State.Annotations.Remove(addr, a)
	t.ctx.MutationCount++
	return fmt.Sprintf("Removed annotation at $%04X (author: %s)", addr, a), nil
}

var _ tool.InvokableTool = (*RemoveAnnotationTool)(nil)
