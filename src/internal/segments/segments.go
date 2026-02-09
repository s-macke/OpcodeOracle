package segments

import (
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
)

// Type identifies the kind of content in a segment.
type Type string

const (
	Code Type = "code" // Code without subroutine symbol
	Sub  Type = "sub"  // Subroutine or entry point
	Data Type = "dat"  // Data region
)

// Segment represents a contiguous block of code or data.
type Segment struct {
	Start uint16
	End   uint16
	Type  Type
	Name  string // Symbol name if present
}

// Plan splits the binary into segments based on regions and symbols.
func Plan(st *state.State) []Segment {
	var out []Segment

	binStart := st.Binary.Start()
	binEnd := st.Binary.End()

	for _, region := range st.Regions.Regions() {
		if region.End < binStart || region.Start > binEnd {
			continue
		}

		start := region.Start
		end := region.End
		if start < binStart {
			start = binStart
		}
		if end > binEnd {
			end = binEnd
		}

		clamped := regions.Region{Start: start, End: end, Type: region.Type}
		if region.Type == regions.RegionData {
			out = append(out, Segment{Start: start, End: end, Type: Data})
			continue
		}

		subSymbols := st.Symbols.SubroutinesInRange(start, end)
		if len(subSymbols) == 0 {
			out = append(out, Segment{Start: start, End: end, Type: Code})
			continue
		}

		out = append(out, splitBySubroutines(clamped, subSymbols)...)
	}

	return out
}

// FilterIntersecting returns segments that intersect [start, end].
func FilterIntersecting(segs []Segment, start, end uint16) []Segment {
	var out []Segment
	for _, seg := range segs {
		if seg.End < start || seg.Start > end {
			continue
		}
		out = append(out, seg)
	}
	return out
}

func splitBySubroutines(region regions.Region, subSymbols []symbols.AddressedSymbol) []Segment {
	var out []Segment
	addr := region.Start

	for i, sym := range subSymbols {
		if addr < sym.Addr {
			out = append(out, Segment{
				Start: addr,
				End:   sym.Addr - 1,
				Type:  Code,
			})
		}

		var endAddr uint16
		if i+1 < len(subSymbols) {
			endAddr = subSymbols[i+1].Addr - 1
		} else {
			endAddr = region.End
		}

		out = append(out, Segment{
			Start: sym.Addr,
			End:   endAddr,
			Type:  Sub,
			Name:  sym.Symbol.Name,
		})

		addr = endAddr + 1
	}

	return out
}
