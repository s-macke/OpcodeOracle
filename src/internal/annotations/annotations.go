package annotations

import "opcodeoracle/internal/author"

type Annotation struct {
	Comment string
	Author  author.Author
}

// AddressAnnotations holds up to two annotations per address (one per author).
type AddressAnnotations struct {
	User      *Annotation
	Assistant *Annotation
}

type Table struct {
	annotations map[uint16]*AddressAnnotations
}

func NewTable() *Table {
	return &Table{
		annotations: make(map[uint16]*AddressAnnotations),
	}
}

// Set sets the annotation for the given author at the address (replaces any existing).
func (t *Table) Set(addr uint16, comment string, a author.Author) {
	ann := &Annotation{
		Comment: comment,
		Author:  a,
	}

	addrAnns := t.annotations[addr]
	if addrAnns == nil {
		addrAnns = &AddressAnnotations{}
		t.annotations[addr] = addrAnns
	}

	switch a {
	case author.User:
		addrAnns.User = ann
	case author.Assistant:
		addrAnns.Assistant = ann
	}
}

// Get returns the annotation for the given author at the address (nil if none).
func (t *Table) Get(addr uint16, a author.Author) *Annotation {
	addrAnns := t.annotations[addr]
	if addrAnns == nil {
		return nil
	}

	switch a {
	case author.User:
		return addrAnns.User
	case author.Assistant:
		return addrAnns.Assistant
	}
	return nil
}

// At returns all annotations at the given address (for backwards compatibility).
func (t *Table) At(addr uint16) []Annotation {
	addrAnns := t.annotations[addr]
	if addrAnns == nil {
		return []Annotation{}
	}

	var result []Annotation
	if addrAnns.User != nil {
		result = append(result, *addrAnns.User)
	}
	if addrAnns.Assistant != nil {
		result = append(result, *addrAnns.Assistant)
	}
	return result
}

// Remove removes the annotation for the given author at the address.
func (t *Table) Remove(addr uint16, a author.Author) {
	addrAnns := t.annotations[addr]
	if addrAnns == nil {
		return
	}

	switch a {
	case author.User:
		addrAnns.User = nil
	case author.Assistant:
		addrAnns.Assistant = nil
	}

	// Clean up if no annotations remain
	if addrAnns.User == nil && addrAnns.Assistant == nil {
		delete(t.annotations, addr)
	}
}

// Clear removes all annotations at the given address.
func (t *Table) Clear(addr uint16) {
	delete(t.annotations, addr)
}

// All returns all annotations as a map from address to AddressAnnotations.
func (t *Table) All() map[uint16]*AddressAnnotations {
	return t.annotations
}

// Extend appends to the annotation for the given author at the address.
// If no annotation exists, it creates one. Uses newline as separator.
func (t *Table) Extend(addr uint16, comment string, a author.Author) {
	existing := t.Get(addr, a)
	if existing != nil {
		// Append with newline separator
		comment = existing.Comment + "\n" + comment
	}
	t.Set(addr, comment, a)
}
