package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"opcodeoracle/internal/disasm"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ListSubroutinesTool lists all subroutines in the binary or a range.
type ListSubroutinesTool struct {
	ctx *Context
}

// NewListSubroutinesTool creates a new list_subroutines tool.
func NewListSubroutinesTool(ctx *Context) *ListSubroutinesTool {
	return &ListSubroutinesTool{ctx: ctx}
}

type listSubroutinesParams struct {
	StartAddr string `json:"start_addr"`
	EndAddr   string `json:"end_addr"`
}

// Info returns the tool's metadata.
func (t *ListSubroutinesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_subroutines",
		Desc: "List all subroutines (and entry points) in the binary or within an address range. Shows subroutine addresses and names. Use to get an overview of the code structure.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"start_addr": {
				Type:     schema.String,
				Desc:     "Start address for range filter (optional, defaults to binary start)",
				Required: false,
			},
			"end_addr": {
				Type:     schema.String,
				Desc:     "End address for range filter (optional, defaults to binary end)",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *ListSubroutinesTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params listSubroutinesParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	// Default to binary bounds
	startAddr := t.ctx.State.Binary.Start()
	endAddr := t.ctx.State.Binary.End()

	if params.StartAddr != "" {
		var err error
		startAddr, err = parseAddress(params.StartAddr)
		if err != nil {
			return fmt.Sprintf("Error: invalid start_addr: %v", err), nil
		}
	}
	if params.EndAddr != "" {
		var err error
		endAddr, err = parseAddress(params.EndAddr)
		if err != nil {
			return fmt.Sprintf("Error: invalid end_addr: %v", err), nil
		}
	}

	subs := t.ctx.State.Symbols.SubroutinesInRange(startAddr, endAddr)

	if len(subs) == 0 {
		return fmt.Sprintf("No subroutines found in range $%04X-$%04X", startAddr, endAddr), nil
	}

	var results []string
	results = append(results, fmt.Sprintf("Found %d subroutines in range $%04X-$%04X:", len(subs), startAddr, endAddr))

	for _, sub := range subs {
		// Count callers
		callers := t.ctx.State.XRefs.To(sub.Addr)
		callerCount := 0
		for _, ref := range callers {
			if ref.Type == "call" {
				callerCount++
			}
		}

		// Check if has headline
		hasHeadline := ""
		headlines := t.ctx.State.Headlines.At(sub.Addr)
		if len(headlines) > 0 {
			hasHeadline = " [documented]"
		}

		results = append(results, fmt.Sprintf("  $%04X: %s (%s, %d callers)%s",
			sub.Addr, sub.Symbol.Name, sub.Symbol.Type, callerCount, hasHeadline))
	}

	return strings.Join(results, "\n"), nil
}

var _ tool.InvokableTool = (*ListSubroutinesTool)(nil)

// GetSubroutineContextTool gets detailed context for a subroutine.
type GetSubroutineContextTool struct {
	ctx *Context
}

// NewGetSubroutineContextTool creates a new get_subroutine_context tool.
func NewGetSubroutineContextTool(ctx *Context) *GetSubroutineContextTool {
	return &GetSubroutineContextTool{ctx: ctx}
}

type getSubroutineContextParams struct {
	Address string `json:"address"`
}

// Info returns the tool's metadata.
func (t *GetSubroutineContextTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_subroutine_context",
		Desc: "Get detailed context for a subroutine including its code, callers, and callees. Useful for understanding a subroutine's purpose before adding documentation.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"address": {
				Type:     schema.String,
				Desc:     "Subroutine address in hex (e.g., '$C000' or '0xC000')",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *GetSubroutineContextTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params getSubroutineContextParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	addr, err := parseAddress(params.Address)
	if err != nil {
		return fmt.Sprintf("Error: invalid address: %v", err), nil
	}

	var results []string

	// Get symbol info
	sym, hasSym := t.ctx.State.Symbols.At(addr)
	if hasSym {
		results = append(results, fmt.Sprintf("Subroutine: %s at $%04X (%s)", sym.Name, addr, sym.Type))
	} else {
		results = append(results, fmt.Sprintf("Subroutine at $%04X (no symbol)", addr))
	}

	// Get existing headline
	headlines := t.ctx.State.Headlines.At(addr)
	if len(headlines) > 0 {
		results = append(results, "\nExisting headline:")
		for _, h := range headlines {
			results = append(results, fmt.Sprintf("  [%s] %s", h.Author, h.Comment))
		}
	}

	// Get callers
	callers := t.ctx.State.XRefs.To(addr)
	if len(callers) > 0 {
		results = append(results, "\nCallers:")
		for _, ref := range callers {
			callerSym := ""
			if s, ok := t.ctx.State.Symbols.At(ref.From); ok {
				callerSym = " (" + s.Name + ")"
			}
			results = append(results, fmt.Sprintf("  $%04X%s [%s]", ref.From, callerSym, ref.Type))
		}
	} else {
		results = append(results, "\nNo callers found (may be entry point or called indirectly)")
	}

	// Get callees (what this subroutine calls)
	// We need to scan from the subroutine start to find all outgoing refs
	// Find the end of this subroutine (next subroutine or reasonable limit)
	subEnd := findSubroutineEnd(t.ctx, addr)

	var callees []string
	for scanAddr := addr; ; scanAddr++ {
		refs := t.ctx.State.XRefs.From(scanAddr)
		for _, ref := range refs {
			if ref.Type == "call" {
				calleeSym := ""
				if s, ok := t.ctx.State.Symbols.At(ref.To); ok {
					calleeSym = s.Name
				} else {
					calleeSym = fmt.Sprintf("$%04X", ref.To)
				}
				callees = append(callees, fmt.Sprintf("  $%04X -> %s", ref.From, calleeSym))
			}
		}
		if scanAddr == subEnd {
			break
		}
	}

	if len(callees) > 0 {
		results = append(results, "\nCalls:")
		results = append(results, callees...)
	}

	// Show the disassembly
	results = append(results, "\nDisassembly:")
	d := disasm.NewDisassembler(t.ctx.State, t.ctx.Analyzer)
	exclusiveEnd := subEnd + 1
	if subEnd == 0xFFFF {
		exclusiveEnd = 0xFFFF
	}
	code, err := d.Disassemble(addr, exclusiveEnd)
	if err != nil {
		results = append(results, fmt.Sprintf("  (error: %v)", err))
	} else {
		results = append(results, code)
	}

	return strings.Join(results, "\n"), nil
}

// findSubroutineEnd finds a reasonable end address for a subroutine.
func findSubroutineEnd(ctx *Context, start uint16) uint16 {
	// Look for the next subroutine symbol or limit to 256 bytes
	maxEnd32 := uint32(start) + 256
	if maxEnd32 > uint32(ctx.State.Binary.End()) {
		maxEnd32 = uint32(ctx.State.Binary.End())
	}
	maxEnd := uint16(maxEnd32)

	subs := ctx.State.Symbols.SubroutinesInRange(start+1, maxEnd)
	if len(subs) > 0 {
		// End just before the next subroutine
		return subs[0].Addr - 1
	}

	// No next subroutine found, use a reasonable limit
	// Try to find an RTS/RTI by scanning instruction boundaries
	if ctx.Analyzer != nil {
		addrs := ctx.Analyzer.InstructionAddresses()
		for _, instrAddr := range addrs {
			if instrAddr <= start {
				continue
			}
			if instrAddr > maxEnd {
				break
			}
			// Read opcode
			opcode, err := ctx.State.Binary.ReadByte(instrAddr)
			if err != nil {
				continue
			}
			// RTS = $60, RTI = $40
			if opcode == 0x60 || opcode == 0x40 {
				return instrAddr
			}
		}
	}

	return maxEnd
}

var _ tool.InvokableTool = (*GetSubroutineContextTool)(nil)
