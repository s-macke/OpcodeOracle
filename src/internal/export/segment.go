// Package export provides assembly file generation from disassembled state.
package export

import (
	"sort"

	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/symbols"
)

// SegmentType identifies the kind of content in a segment.
type SegmentType string

const (
	SegmentCode SegmentType = "code" // Code without subroutine symbol
	SegmentSub  SegmentType = "sub"  // Subroutine or entry point
	SegmentData SegmentType = "dat"  // Data region
)

// Segment represents a contiguous block of code or data for export.
type Segment struct {
	Start uint16
	End   uint16
	Type  SegmentType
	Name  string // Symbol name if present
}

// identifySegments splits the binary into exportable segments based on regions and symbols.
func (e *Exporter) identifySegments() []Segment {
	var segments []Segment

	// Calculate binary bounds
	binStart := e.state.Binary.Origin
	binEnd := binStart + uint16(len(e.state.Binary.Data)) - 1

	for _, region := range e.state.Regions.Regions() {
		// Skip regions completely outside binary bounds
		if region.End < binStart || region.Start > binEnd {
			continue
		}

		// Clamp region to binary bounds
		start := region.Start
		end := region.End
		if start < binStart {
			start = binStart
		}
		if end > binEnd {
			end = binEnd
		}

		clampedRegion := regions.Region{Start: start, End: end, Type: region.Type}

		if region.Type == regions.RegionData {
			segments = append(segments, Segment{
				Start: start,
				End:   end,
				Type:  SegmentData,
			})
		} else {
			// Code region - split by subroutine/entry symbols
			subSymbols := e.findSubroutineSymbols(start, end)

			if len(subSymbols) == 0 {
				// No subroutines - entire region is code
				segments = append(segments, Segment{
					Start: start,
					End:   end,
					Type:  SegmentCode,
				})
			} else {
				// Split by subroutine symbols
				segments = append(segments, e.splitBySubroutines(clampedRegion, subSymbols)...)
			}
		}
	}

	return segments
}

// symbolEntry holds a symbol's address and name for sorting.
type symbolEntry struct {
	addr uint16
	name string
}

// findSubroutineSymbols returns all subroutine/entry symbols within the address range.
func (e *Exporter) findSubroutineSymbols(start, end uint16) []symbolEntry {
	var result []symbolEntry

	allSymbols := e.state.Symbols.All()
	for addr, syms := range allSymbols {
		if addr < start || addr > end {
			continue
		}
		for _, sym := range syms {
			if sym.Type == symbols.SymbolSubroutine || sym.Type == symbols.SymbolEntry {
				result = append(result, symbolEntry{addr: addr, name: sym.Name})
				break // Only need one symbol per address
			}
		}
	}

	// Sort by address
	sort.Slice(result, func(i, j int) bool {
		return result[i].addr < result[j].addr
	})

	return result
}

// splitBySubroutines splits a code region into segments at subroutine boundaries.
func (e *Exporter) splitBySubroutines(region regions.Region, subSymbols []symbolEntry) []Segment {
	var segments []Segment
	addr := region.Start

	for i, sym := range subSymbols {
		if addr < sym.addr {
			// Code before this subroutine
			segments = append(segments, Segment{
				Start: addr,
				End:   sym.addr - 1,
				Type:  SegmentCode,
			})
		}

		// Find end address (next symbol or region end)
		var endAddr uint16
		if i+1 < len(subSymbols) {
			endAddr = subSymbols[i+1].addr - 1
		} else {
			endAddr = region.End
		}

		segments = append(segments, Segment{
			Start: sym.addr,
			End:   endAddr,
			Type:  SegmentSub,
			Name:  sym.name,
		})

		addr = endAddr + 1
	}

	return segments
}
