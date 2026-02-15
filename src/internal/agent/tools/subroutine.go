package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"opcodeoracle/internal/disasm"
	"opcodeoracle/internal/headlines"
	"opcodeoracle/internal/numparse"
	"opcodeoracle/internal/segments"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ListSubroutinesAndDataSegmentsTool lists subroutines and non-subroutine segments.
type ListSubroutinesAndDataSegmentsTool struct {
	ctx *Context
}

// NewListSubroutinesAndDataSegmentsTool creates a new list_subroutines_and_data_segments tool.
func NewListSubroutinesAndDataSegmentsTool(ctx *Context) *ListSubroutinesAndDataSegmentsTool {
	return &ListSubroutinesAndDataSegmentsTool{ctx: ctx}
}

type listSubroutinesAndDataSegmentsParams struct {
	StartAddr *string `json:"start_addr,omitempty"`
	EndAddr   *string `json:"end_addr,omitempty"`
}

// Info returns the tool's metadata.
func (t *ListSubroutinesAndDataSegmentsTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_subroutines_and_data_segments",
		Desc: "List subroutine/entry, code, and data segments in the binary or within an address range. Subroutine rows include names and caller counts for quick code-structure overview.",
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
func (t *ListSubroutinesAndDataSegmentsTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params listSubroutinesAndDataSegmentsParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	// Default to binary bounds
	startAddr := t.ctx.State.Binary.Start()
	endAddr := t.ctx.State.Binary.End()

	if params.StartAddr != nil && *params.StartAddr != "" {
		var err error
		startAddr, err = numparse.ParseUint16(*params.StartAddr)
		if err != nil {
			return fmt.Sprintf("Error: invalid start_addr: %v", err), nil
		}
	}
	if params.EndAddr != nil && *params.EndAddr != "" {
		var err error
		endAddr, err = numparse.ParseUint16(*params.EndAddr)
		if err != nil {
			return fmt.Sprintf("Error: invalid end_addr: %v", err), nil
		}
	}

	allSegs := segments.Plan(t.ctx.State)
	segs := segments.FilterIntersecting(allSegs, startAddr, endAddr)

	var results []string
	if desc := t.ctx.State.Metadata.Description; desc != "" {
		results = append(results, fmt.Sprintf("Description: %s", desc))
		results = append(results, "")
	}
	results = append(results, fmt.Sprintf("Found %d segments in range $%04X-$%04X:", len(segs), startAddr, endAddr))

	var subLines []string
	var codeLines []string
	var dataLines []string
	allHeadlines := t.ctx.State.Headlines.All()

	for _, seg := range segs {
		switch seg.Type {
		case segments.Sub:
			callers := t.ctx.State.XRefs.To(seg.Start)
			callerCount := 0
			for _, ref := range callers {
				if ref.Type == "call" {
					callerCount++
				}
			}

			symType := "subroutine"
			if sym, ok := t.ctx.State.Symbols.At(seg.Start); ok {
				symType = string(sym.Type)
			}

			hasHeadline := ""
			if len(t.ctx.State.Headlines.At(seg.Start)) > 0 {
				hasHeadline = " [documented]"
			}

			name := seg.Name
			if name == "" {
				name = fmt.Sprintf("SUB_%04X", seg.Start)
			}

			subLines = append(subLines, fmt.Sprintf("  $%04X: %s (%s, %d callers)%s [%04X-%04X]",
				seg.Start, name, symType, callerCount, hasHeadline, seg.Start, seg.End))
		case segments.Code:
			hasHeadline := ""
			if segmentHasHeadlineInRange(seg.Start, seg.End, allHeadlines) {
				hasHeadline = " [documented]"
			}
			codeSymbol := ""
			if sym, ok := t.ctx.State.Symbols.At(seg.Start); ok {
				codeSymbol = fmt.Sprintf(" (%s, %s)", sym.Name, sym.Type)
			}
			codeLines = append(codeLines, fmt.Sprintf("  $%04X-$%04X: code%s%s", seg.Start, seg.End, codeSymbol, hasHeadline))
		case segments.Data:
			hasHeadline := ""
			if segmentHasHeadlineInRange(seg.Start, seg.End, allHeadlines) {
				hasHeadline = " [documented]"
			}
			dataSymbol := ""
			if sym, ok := t.ctx.State.Symbols.At(seg.Start); ok {
				dataSymbol = fmt.Sprintf(" (%s, %s)", sym.Name, sym.Type)
			}
			dataLines = append(dataLines, fmt.Sprintf("  $%04X-$%04X: data%s%s", seg.Start, seg.End, dataSymbol, hasHeadline))
		}
	}

	results = append(results, "")
	results = append(results, fmt.Sprintf("Subroutine/Entry segments (%d):", len(subLines)))
	results = append(results, subLines...)

	results = append(results, "")
	results = append(results, fmt.Sprintf("Code segments (%d):", len(codeLines)))
	results = append(results, codeLines...)

	results = append(results, "")
	results = append(results, fmt.Sprintf("Data segments (%d):", len(dataLines)))
	results = append(results, dataLines...)

	if len(segs) == 0 {
		results = append(results, "")
		results = append(results, "No segments intersect the requested range.")
	}

	return strings.Join(results, "\n"), nil
}

var _ tool.InvokableTool = (*ListSubroutinesAndDataSegmentsTool)(nil)

func segmentHasHeadlineInRange(start, end uint16, all map[uint16]*headlines.AddressHeadlines) bool {
	for addr := range all {
		if addr >= start && addr <= end {
			return true
		}
	}
	return false
}

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

	addr, err := numparse.ParseUint16(params.Address)
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
