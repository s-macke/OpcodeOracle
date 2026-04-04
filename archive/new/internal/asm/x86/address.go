package x86

import "fmt"

type FarAddress struct {
	Segment uint16
	Offset  uint16
}

func NewFarAddress(segment uint16, offset uint16) FarAddress {
	return FarAddress{
		Segment: segment,
		Offset:  offset,
	}
}

func (a FarAddress) Linear() uint32 {
	return uint32(a.Segment)<<4 + uint32(a.Offset)
}

func (a FarAddress) String() string {
	return fmt.Sprintf("%04x:%04x", a.Segment, a.Offset)
}
