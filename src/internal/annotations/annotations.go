package annotations

import "errors"

var (
	ErrIndexOutOfRange = errors.New("index out of range")
)

type AnnotationType int

const (
	AnnotationInline AnnotationType = iota
	AnnotationHeadline
)

type Annotation struct {
	Type    AnnotationType
	Comment string
	Author  string
}

type Table struct {
	annotations map[uint16][]Annotation
}

func NewTable() *Table {
	return &Table{
		annotations: make(map[uint16][]Annotation),
	}
}

// At returns all annotations at the given address.
func (t *Table) At(addr uint16) []Annotation {
	if anns, ok := t.annotations[addr]; ok {
		return anns
	}
	return []Annotation{}
}

// Add adds an annotation at the given address.
func (t *Table) Add(addr uint16, typ AnnotationType, comment, author string) {
	ann := Annotation{
		Type:    typ,
		Comment: comment,
		Author:  author,
	}
	t.annotations[addr] = append(t.annotations[addr], ann)
}

// Remove removes an annotation by index at the given address.
func (t *Table) Remove(addr uint16, index int) error {
	anns := t.annotations[addr]
	if index < 0 || index >= len(anns) {
		return ErrIndexOutOfRange
	}
	t.annotations[addr] = append(anns[:index], anns[index+1:]...)
	if len(t.annotations[addr]) == 0 {
		delete(t.annotations, addr)
	}
	return nil
}

// Clear removes all annotations at the given address.
func (t *Table) Clear(addr uint16) {
	delete(t.annotations, addr)
}

// All returns all annotations as a map from address to annotation slice.
func (t *Table) All() map[uint16][]Annotation {
	return t.annotations
}
