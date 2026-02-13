package regions

import (
	"fmt"
	"sort"
)

type RegionType string

const (
	RegionCode RegionType = "code"
	RegionData RegionType = "data"
)

type RegionSource string

const (
	RegionSourceAuto      RegionSource = "auto"
	RegionSourceAssistant RegionSource = "assistant"
	RegionSourceUser      RegionSource = "user"
)

type Region struct {
	Start  uint16
	End    uint16
	Type   RegionType
	Source RegionSource
}

type Table struct {
	regions []Region
}

func NewTable() *Table {
	return &Table{
		regions: make([]Region, 0),
	}
}

// RegionAt returns the region containing the given address.
func (t *Table) RegionAt(addr uint16) *Region {
	for i := range t.regions {
		if addr >= t.regions[i].Start && addr <= t.regions[i].End {
			return &t.regions[i]
		}
	}
	return nil
}

// At returns the region type at the given address.
// Returns empty string if no region contains the address.
func (t *Table) At(addr uint16) RegionType {
	r := t.RegionAt(addr)
	if r == nil { // If everything works, that should never happen
		return RegionData
	}
	return r.Type
}

// SourceAt returns the region source at the given address.
func (t *Table) SourceAt(addr uint16) RegionSource {
	r := t.RegionAt(addr)
	if r == nil {
		return RegionSourceAuto
	}
	return normalizeSource(r.Source)
}

// Set sets the region type for the given address range.
// Handles splitting and merging of existing regions.
func (t *Table) Set(start, end uint16, regionType RegionType) {
	t.SetWithSource(start, end, regionType, RegionSourceAuto)
}

// SetWithSource sets region type + source for the given range, respecting source priority.
func (t *Table) SetWithSource(start, end uint16, regionType RegionType, source RegionSource) {
	if start > end {
		return
	}
	source = normalizeSource(source)

	if len(t.regions) == 0 {
		t.regions = []Region{{
			Start:  start,
			End:    end,
			Type:   regionType,
			Source: source,
		}}
		if err := t.Validate(); err != nil {
			panic(err)
		}
		return
	}

	var result []Region

	for _, r := range t.regions {
		r.Source = normalizeSource(r.Source)
		// Region is completely before new range
		if r.End < start {
			result = append(result, r)
			continue
		}
		// Region is completely after new range
		if r.Start > end {
			result = append(result, r)
			continue
		}

		// Region overlaps - split into left/overlap/right and apply source priority to overlap.
		if r.Start < start {
			result = append(result, Region{
				Start:  r.Start,
				End:    start - 1,
				Type:   r.Type,
				Source: r.Source,
			})
		}

		overlapStart := max16(r.Start, start)
		overlapEnd := min16(r.End, end)
		if sourcePriority(source) >= sourcePriority(r.Source) {
			result = append(result, Region{
				Start:  overlapStart,
				End:    overlapEnd,
				Type:   regionType,
				Source: source,
			})
		} else {
			result = append(result, Region{
				Start:  overlapStart,
				End:    overlapEnd,
				Type:   r.Type,
				Source: r.Source,
			})
		}

		if r.End > end {
			result = append(result, Region{
				Start:  end + 1,
				End:    r.End,
				Type:   r.Type,
				Source: r.Source,
			})
		}
	}

	// Sort by start address
	sort.Slice(result, func(i, j int) bool {
		return result[i].Start < result[j].Start
	})

	// Merge adjacent regions of the same type
	t.regions = merge(result)

	if err := t.Validate(); err != nil {
		panic(err)
	}
}

// Validate checks that regions are non-overlapping, merged, and fully cover the table's address range.
func (t *Table) Validate() error {
	if len(t.regions) == 0 {
		return fmt.Errorf("no regions defined")
	}

	first := t.regions[0]
	if first.Start != 0x0000 {
		return fmt.Errorf("region coverage starts at %04X, want 0000", first.Start)
	}
	if first.Start > first.End {
		return fmt.Errorf("invalid region %04X-%04X", first.Start, first.End)
	}

	prev := first
	for i := 1; i < len(t.regions); i++ {
		curr := t.regions[i]
		if curr.Start > curr.End {
			return fmt.Errorf("invalid region %04X-%04X", curr.Start, curr.End)
		}
		if curr.Start <= prev.End {
			return fmt.Errorf("regions overlap or are out of order at %04X-%04X and %04X-%04X",
				prev.Start, prev.End, curr.Start, curr.End)
		}
		if uint32(curr.Start) != uint32(prev.End)+1 {
			return fmt.Errorf("gap between regions at %04X-%04X and %04X-%04X",
				prev.Start, prev.End, curr.Start, curr.End)
		}
		if curr.Type == prev.Type && normalizeSource(curr.Source) == normalizeSource(prev.Source) {
			return fmt.Errorf("adjacent regions not merged at %04X-%04X and %04X-%04X",
				prev.Start, prev.End, curr.Start, curr.End)
		}
		prev = curr
	}

	if prev.End != 0xFFFF {
		return fmt.Errorf("region coverage ends at %04X, want FFFF", prev.End)
	}

	return nil
}

func merge(regions []Region) []Region {
	if len(regions) == 0 {
		return regions
	}

	var merged []Region
	current := regions[0]
	current.Source = normalizeSource(current.Source)

	for i := 1; i < len(regions); i++ {
		regions[i].Source = normalizeSource(regions[i].Source)
		// Check if adjacent and same type
		if regions[i].Start == current.End+1 &&
			regions[i].Type == current.Type &&
			normalizeSource(regions[i].Source) == normalizeSource(current.Source) {
			current.End = regions[i].End
		} else {
			merged = append(merged, current)
			current = regions[i]
		}
	}
	merged = append(merged, current)

	return merged
}

// Regions returns all regions.
func (t *Table) Regions() []Region {
	return t.regions
}

// SetRegions replaces all regions.
func (t *Table) SetRegions(regions []Region) {
	normalized := make([]Region, len(regions))
	for i, r := range regions {
		normalized[i] = Region{
			Start:  r.Start,
			End:    r.End,
			Type:   r.Type,
			Source: normalizeSource(r.Source),
		}
	}
	t.regions = normalized
}

func sourcePriority(source RegionSource) int {
	switch normalizeSource(source) {
	case RegionSourceUser:
		return 3
	case RegionSourceAssistant:
		return 2
	default:
		return 1
	}
}

func normalizeSource(source RegionSource) RegionSource {
	if source == "" {
		return RegionSourceAuto
	}
	return source
}

func min16(a, b uint16) uint16 {
	if a < b {
		return a
	}
	return b
}

func max16(a, b uint16) uint16 {
	if a > b {
		return a
	}
	return b
}
