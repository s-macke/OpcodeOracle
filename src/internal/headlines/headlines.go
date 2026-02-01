package headlines

import "opcodeoracle/internal/author"

type Headline struct {
	Comment string
	Author  author.Author
}

// AddressHeadlines holds up to two headlines per address (one per author).
type AddressHeadlines struct {
	User      *Headline
	Assistant *Headline
}

type Table struct {
	headlines map[uint16]*AddressHeadlines
}

func NewTable() *Table {
	return &Table{
		headlines: make(map[uint16]*AddressHeadlines),
	}
}

// Set sets the headline for the given author at the address (replaces any existing).
func (t *Table) Set(addr uint16, comment string, a author.Author) {
	h := &Headline{
		Comment: comment,
		Author:  a,
	}

	addrHdls := t.headlines[addr]
	if addrHdls == nil {
		addrHdls = &AddressHeadlines{}
		t.headlines[addr] = addrHdls
	}

	switch a {
	case author.User:
		addrHdls.User = h
	case author.Assistant:
		addrHdls.Assistant = h
	}
}

// Get returns the headline for the given author at the address (nil if none).
func (t *Table) Get(addr uint16, a author.Author) *Headline {
	addrHdls := t.headlines[addr]
	if addrHdls == nil {
		return nil
	}

	switch a {
	case author.User:
		return addrHdls.User
	case author.Assistant:
		return addrHdls.Assistant
	}
	return nil
}

// At returns all headlines at the given address.
func (t *Table) At(addr uint16) []Headline {
	addrHdls := t.headlines[addr]
	if addrHdls == nil {
		return []Headline{}
	}

	var result []Headline
	if addrHdls.User != nil {
		result = append(result, *addrHdls.User)
	}
	if addrHdls.Assistant != nil {
		result = append(result, *addrHdls.Assistant)
	}
	return result
}

// Remove removes the headline for the given author at the address.
func (t *Table) Remove(addr uint16, a author.Author) {
	addrHdls := t.headlines[addr]
	if addrHdls == nil {
		return
	}

	switch a {
	case author.User:
		addrHdls.User = nil
	case author.Assistant:
		addrHdls.Assistant = nil
	}

	// Clean up if no headlines remain
	if addrHdls.User == nil && addrHdls.Assistant == nil {
		delete(t.headlines, addr)
	}
}

// Clear removes all headlines at the given address.
func (t *Table) Clear(addr uint16) {
	delete(t.headlines, addr)
}

// All returns all headlines as a map from address to AddressHeadlines.
func (t *Table) All() map[uint16]*AddressHeadlines {
	return t.headlines
}

// Extend appends to the headline for the given author at the address.
// If no headline exists, it creates one. Uses newline as separator.
func (t *Table) Extend(addr uint16, comment string, a author.Author) {
	existing := t.Get(addr, a)
	if existing != nil {
		// Append with newline separator
		comment = existing.Comment + "\n" + comment
	}
	t.Set(addr, comment, a)
}
