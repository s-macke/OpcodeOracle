package annotations

import "errors"

var (
	ErrInvalidAuthor = errors.New("invalid author: must be 'user' or 'assistant'")
)

type AnnotationType int

const (
	AnnotationInline AnnotationType = iota
	AnnotationHeadline
)

type Author int

const (
	AuthorUser Author = iota
	AuthorAssistant
)

func (a Author) String() string {
	switch a {
	case AuthorUser:
		return "user"
	case AuthorAssistant:
		return "assistant"
	default:
		return "unknown"
	}
}

func ParseAuthor(s string) (Author, error) {
	switch s {
	case "user":
		return AuthorUser, nil
	case "assistant":
		return AuthorAssistant, nil
	default:
		return 0, ErrInvalidAuthor
	}
}

type Annotation struct {
	Type    AnnotationType
	Comment string
	Author  Author
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
func (t *Table) Set(addr uint16, typ AnnotationType, comment string, author Author) {
	ann := &Annotation{
		Type:    typ,
		Comment: comment,
		Author:  author,
	}

	addrAnns := t.annotations[addr]
	if addrAnns == nil {
		addrAnns = &AddressAnnotations{}
		t.annotations[addr] = addrAnns
	}

	switch author {
	case AuthorUser:
		addrAnns.User = ann
	case AuthorAssistant:
		addrAnns.Assistant = ann
	}
}

// Get returns the annotation for the given author at the address (nil if none).
func (t *Table) Get(addr uint16, author Author) *Annotation {
	addrAnns := t.annotations[addr]
	if addrAnns == nil {
		return nil
	}

	switch author {
	case AuthorUser:
		return addrAnns.User
	case AuthorAssistant:
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
func (t *Table) Remove(addr uint16, author Author) {
	addrAnns := t.annotations[addr]
	if addrAnns == nil {
		return
	}

	switch author {
	case AuthorUser:
		addrAnns.User = nil
	case AuthorAssistant:
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
// The new type takes precedence over the existing type.
func (t *Table) Extend(addr uint16, typ AnnotationType, comment string, author Author) {
	existing := t.Get(addr, author)
	if existing != nil {
		// Append with newline separator
		comment = existing.Comment + "\n" + comment
	}
	t.Set(addr, typ, comment, author)
}
