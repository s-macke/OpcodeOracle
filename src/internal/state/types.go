package state

import (
	"time"

	"opcodeoracle/internal/annotations"
	"opcodeoracle/internal/binary"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/symbols"
)

type State struct {
	Version     string
	Metadata    Metadata
	Binary      binary.Binary
	Symbols     map[uint16][]symbols.Symbol
	Annotations map[uint16][]annotations.Annotation
	Regions     []regions.Region
}

type Metadata struct {
	Created     time.Time
	Modified    time.Time
	SourceFile  string
	Description string
}
