package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"opcodeoracle/internal/analysis"
	"opcodeoracle/internal/disasm"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	defaultSearchContextLines = 1
	defaultSearchMaxResults   = 20
	maxSearchContextLines     = 3
	maxSearchResults          = 200
)

// SearchDisassemblyTool searches rendered disassembly text.
type SearchDisassemblyTool struct {
	ctx *Context
}

// NewSearchDisassemblyTool creates a new search_disassembly tool.
func NewSearchDisassemblyTool(ctx *Context) *SearchDisassemblyTool {
	return &SearchDisassemblyTool{ctx: ctx}
}

type searchDisassemblyParams struct {
	Query         string `json:"query"`
	CaseSensitive *bool  `json:"case_sensitive"`
	ContextLines  *int   `json:"context_lines"`
	MaxResults    *int   `json:"max_results"`
}

type disasmLine struct {
	Text string
	Addr *uint16
}

// Info returns the tool's metadata.
func (t *SearchDisassemblyTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "search_disassembly",
		Desc: "Search disassembly text for a query string. Returns match addresses, lines, and nearby context. Useful for quickly finding opcodes, symbols, comments, and patterns.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "Text to search for in rendered disassembly output",
				Required: true,
			},
			"case_sensitive": {
				Type:     schema.Boolean,
				Desc:     "Case-sensitive matching (optional, default false)",
				Required: false,
			},
			"context_lines": {
				Type:     schema.Integer,
				Desc:     "Number of context lines before/after each match (optional, default 1, max 3)",
				Required: false,
			},
			"max_results": {
				Type:     schema.Integer,
				Desc:     "Maximum number of matches to return (optional, default 20, max 200)",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the tool.
func (t *SearchDisassemblyTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	t.ctx.Mu.Lock()
	defer t.ctx.Mu.Unlock()

	var params searchDisassemblyParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
	}

	query := strings.TrimSpace(params.Query)
	if query == "" {
		return "Error: query cannot be empty", nil
	}

	startAddr := t.ctx.State.Binary.Start()
	endAddr := t.ctx.State.Binary.End()

	caseSensitive := false
	if params.CaseSensitive != nil {
		caseSensitive = *params.CaseSensitive
	}
	contextLines := clampInt(valueOrDefault(params.ContextLines, defaultSearchContextLines), 0, maxSearchContextLines)
	maxResults := clampInt(valueOrDefault(params.MaxResults, defaultSearchMaxResults), 1, maxSearchResults)

	var boundaries analysis.InstructionBoundaries
	if t.ctx.Analyzer != nil {
		boundaries = t.ctx.Analyzer
	}
	d := disasm.NewDisassembler(t.ctx.State, boundaries)
	exclusiveEnd := endAddr + 1
	if endAddr == 0xFFFF {
		exclusiveEnd = 0xFFFF
	}
	out, err := d.Disassemble(startAddr, exclusiveEnd)
	if err != nil {
		return fmt.Sprintf("Error: disassembly failed: %v", err), nil
	}

	lines := parseDisassemblyLines(out)
	matches := findMatchingLineIndexes(lines, query, caseSensitive)
	if len(matches) == 0 {
		return "No disassembly matches found.", nil
	}

	shown := len(matches)
	if shown > maxResults {
		shown = maxResults
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d matches (showing %d):\n", len(matches), shown))
	for i := 0; i < shown; i++ {
		idx := matches[i]
		sb.WriteString(formatSearchMatchBlock(lines, idx, i+1, contextLines))
		if i < shown-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

func parseDisassemblyLines(out string) []disasmLine {
	raw := strings.Split(out, "\n")
	lines := make([]disasmLine, 0, len(raw))
	var currentAddr *uint16
	for _, line := range raw {
		if addr, ok := parseLeadingAddress(line); ok {
			currentAddr = &addr
		}
		var addrCopy *uint16
		if currentAddr != nil {
			v := *currentAddr
			addrCopy = &v
		}
		lines = append(lines, disasmLine{
			Text: line,
			Addr: addrCopy,
		})
	}
	return lines
}

func findMatchingLineIndexes(lines []disasmLine, query string, caseSensitive bool) []int {
	var indexes []int
	needle := query
	if !caseSensitive {
		needle = strings.ToLower(query)
	}

	for i, line := range lines {
		hay := line.Text
		if !caseSensitive {
			hay = strings.ToLower(hay)
		}
		if strings.Contains(hay, needle) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func formatSearchMatchBlock(lines []disasmLine, idx, matchNo, contextLines int) string {
	addr := "unknown"
	if lines[idx].Addr != nil {
		addr = fmt.Sprintf("$%04X", *lines[idx].Addr)
	}

	start := idx - contextLines
	if start < 0 {
		start = 0
	}
	end := idx + contextLines
	if end >= len(lines) {
		end = len(lines) - 1
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%d] %s\n", matchNo, addr))
	for i := start; i <= end; i++ {
		prefix := "  "
		if i == idx {
			prefix = "> "
		}
		sb.WriteString(prefix + lines[i].Text + "\n")
	}
	return sb.String()
}

func parseLeadingAddress(line string) (uint16, bool) {
	s := strings.TrimLeft(line, " \t")
	if len(s) < 5 || s[0] != '$' {
		return 0, false
	}
	hex := s[1:5]
	for _, c := range hex {
		if !isHexDigit(c) {
			return 0, false
		}
	}
	v, err := strconv.ParseUint(hex, 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(v), true
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func valueOrDefault(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

var _ tool.InvokableTool = (*SearchDisassemblyTool)(nil)
