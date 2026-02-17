package xrefs

import "sort"

type XRefType string

const (
	XRefCall   XRefType = "call"
	XRefJump   XRefType = "jump"
	XRefBranch XRefType = "branch"
	XRefRead   XRefType = "read"
	XRefWrite  XRefType = "write"
)

type XRef struct {
	From uint16
	To   uint16
	Type XRefType
}

type Table struct {
	xrefs []XRef
}

func NewTable() *Table {
	return &Table{
		xrefs: make([]XRef, 0),
	}
}

// To returns all cross-references pointing to the given address.
// Results are returned in deterministic order for stable output rendering.
func (t *Table) To(addr uint16) []XRef {
	var result []XRef
	for _, x := range t.xrefs {
		if x.To == addr {
			result = append(result, x)
		}
	}
	if result == nil {
		return []XRef{}
	}
	sortXRefs(result)
	return result
}

// From returns all cross-references originating from the given address.
// Results are returned in deterministic order for stable output rendering.
func (t *Table) From(addr uint16) []XRef {
	var result []XRef
	for _, x := range t.xrefs {
		if x.From == addr {
			result = append(result, x)
		}
	}
	if result == nil {
		return []XRef{}
	}
	sortXRefs(result)
	return result
}

// Add adds a cross-reference if not already present.
func (t *Table) Add(from, to uint16, refType XRefType) {
	for _, x := range t.xrefs {
		if x.From == from && x.To == to && x.Type == refType {
			return
		}
	}
	t.xrefs = append(t.xrefs, XRef{
		From: from,
		To:   to,
		Type: refType,
	})
}

// Remove removes all cross-references from the given source to the given target.
func (t *Table) Remove(from, to uint16) {
	var remaining []XRef
	for _, x := range t.xrefs {
		if x.From != from || x.To != to {
			remaining = append(remaining, x)
		}
	}
	t.xrefs = remaining
}

func sortXRefs(refs []XRef) {
	if len(refs) < 2 {
		return
	}

	sort.Slice(refs, func(i, j int) bool {
		if refs[i].From != refs[j].From {
			return refs[i].From < refs[j].From
		}
		if refs[i].Type != refs[j].Type {
			return refs[i].Type < refs[j].Type
		}
		return refs[i].To < refs[j].To
	})
}
