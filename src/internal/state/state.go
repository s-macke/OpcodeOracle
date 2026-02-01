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

const CurrentVersion = "1.0"

// NewState creates a new state with the given binary data.
func NewState(data []byte, origin uint16, entryPoints []uint16, sourceFile string) *State {
	now := time.Now().UTC()

	regTable := regions.NewTable()
	regTable.SetRegions([]regions.Region{
		{Start: 0x0000, End: 0xFFFF, Type: regions.RegionData},
	})

	return &State{
		Version: CurrentVersion,
		Metadata: Metadata{
			Created:    now,
			Modified:   now,
			SourceFile: sourceFile,
		},
		Binary: binary.Binary{
			Data:   data,
			Origin: origin,
		},
		EntryPoints: entryPoints,
		Symbols:     symbols.NewTable(),
		Annotations: annotations.NewTable(),
		Headlines:   headlines.NewTable(),
		Regions:     regTable,
		XRefs:       xrefs.NewTable(),
	}
}
