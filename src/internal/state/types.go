package state

import (
	"time"

	"opcodeoracle/internal/annotations"
	"opcodeoracle/internal/binary"
	"opcodeoracle/internal/headlines"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/symbols"
	"opcodeoracle/internal/xrefs"
)

type State struct {
	Version     string
	Metadata    Metadata
	Binary      binary.Binary
	EntryPoints []uint16
	Symbols     *symbols.Table
	Annotations *annotations.Table
	Headlines   *headlines.Table
	Regions     *regions.Table
	XRefs       *xrefs.Table // Not persisted, computed during disassembly
}

type Metadata struct {
	Created       time.Time
	Modified      time.Time
	SourceFile    string
	Description   string
	ArchiveOnSave bool
}
