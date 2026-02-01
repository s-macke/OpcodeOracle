package binary

import "errors"

var (
	ErrAddressOutOfRange = errors.New("address out of range")
)

type EntryPoint uint16

type Binary struct {
	Data        []byte
	Origin      uint16
	EntryPoints []EntryPoint
}

// ReadByte reads a byte at the given virtual address.
func (b *Binary) ReadByte(addr uint16) (byte, error) {
	offset := int(addr) - int(b.Origin)
	if offset < 0 || offset >= len(b.Data) {
		return 0, ErrAddressOutOfRange
	}
	return b.Data[offset], nil
}

// ReadWord reads a little-endian word at the given virtual address.
func (b *Binary) ReadWord(addr uint16) (uint16, error) {
	offset := int(addr) - int(b.Origin)
	if offset < 0 || offset+1 >= len(b.Data) {
		return 0, ErrAddressOutOfRange
	}
	lo := uint16(b.Data[offset])
	hi := uint16(b.Data[offset+1])
	return lo | (hi << 8), nil
}

// IsEntryPoint checks if the given address is an entry point.
func (b *Binary) IsEntryPoint(addr uint16) bool {
	for _, ep := range b.EntryPoints {
		if uint16(ep) == addr {
			return true
		}
	}
	return false
}

// Start returns the start address (origin) of the binary.
func (b *Binary) Start() uint16 {
	return b.Origin
}

// End returns the last valid address in the binary (inclusive).
func (b *Binary) End() uint16 {
	return b.Origin + uint16(len(b.Data)) - 1
}
