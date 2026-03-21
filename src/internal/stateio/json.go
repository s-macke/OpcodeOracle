package stateio

import (
	"encoding/json"
	"fmt"
	"time"

	"opcodeoracle/internal/annotations"
	"opcodeoracle/internal/author"
	"opcodeoracle/internal/binary"
	"opcodeoracle/internal/headlines"
	"opcodeoracle/internal/numparse"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
	"opcodeoracle/internal/xrefs"
)

type jsonState struct {
	Version            string                             `json:"version"`
	Metadata           jsonMetadata                       `json:"metadata"`
	Binary             jsonBinary                         `json:"binary"`
	EntryPoints        []string                           `json:"entryPoints"`
	ExtraCodeAddresses []string                           `json:"extraCodeAddresses,omitempty"`
	ForcedData         []jsonAddressRange                 `json:"forcedData,omitempty"` // legacy; load-only migration
	Symbols            map[string]jsonSymbolValue         `json:"symbols,omitempty"`
	Annotations        map[string]*jsonAddressAnnotations `json:"annotations,omitempty"`
	Headlines          map[string]*jsonAddressHeadlines   `json:"headlines,omitempty"`
	Regions            []jsonRegion                       `json:"regions,omitempty"`
}

// jsonSymbolValue can unmarshal either a single jsonSymbol or an array of them (for backward compatibility)
type jsonSymbolValue []jsonSymbol

func (v *jsonSymbolValue) UnmarshalJSON(data []byte) error {
	// Try array first (old format)
	var arr []jsonSymbol
	if err := json.Unmarshal(data, &arr); err == nil {
		*v = arr
		return nil
	}
	// Try single object (new format)
	var single jsonSymbol
	if err := json.Unmarshal(data, &single); err == nil {
		*v = []jsonSymbol{single}
		return nil
	}
	return fmt.Errorf("invalid symbol value")
}

func (v jsonSymbolValue) MarshalJSON() ([]byte, error) {
	// Always serialize as single object (new format)
	if len(v) == 1 {
		return json.Marshal(v[0])
	}
	// Fallback to array if somehow we have multiple (shouldn't happen)
	return json.Marshal([]jsonSymbol(v))
}

type jsonMetadata struct {
	Created       string `json:"created"`
	Modified      string `json:"modified"`
	SourceFile    string `json:"sourceFile,omitempty"`
	Description   string `json:"description,omitempty"`
	ArchiveOnSave bool   `json:"archiveOnSave,omitempty"`
}

type jsonBinary struct {
	Data   []byte `json:"data"`
	Origin string `json:"origin"`
}

type jsonSymbol struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Source string `json:"source"`
}

type jsonAnnotation struct {
	Comment string `json:"comment"`
}

type jsonAddressAnnotations struct {
	User      *jsonAnnotation `json:"user,omitempty"`
	Assistant *jsonAnnotation `json:"assistant,omitempty"`
}

type jsonAddressHeadlines struct {
	User      *jsonHeadline `json:"user,omitempty"`
	Assistant *jsonHeadline `json:"assistant,omitempty"`
}

type jsonHeadline struct {
	Comment string `json:"comment"`
}

type jsonRegion struct {
	Start  string `json:"start"`
	End    string `json:"end"`
	Type   string `json:"type"`
	Source string `json:"source,omitempty"`
}

type jsonAddressRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// parseHex parses a hex string like "0x0801" to uint16.
func parseHex(s string) (uint16, error) {
	return numparse.ParseHexUint16(s)
}

// formatHex formats a uint16 as "0xNNNN".
func formatHex(v uint16) string {
	return fmt.Sprintf("0x%04X", v)
}

// stateToJSON converts a State to its JSON representation.
func stateToJSON(s *state.State) *jsonState {
	js := &jsonState{
		Version: "1.1",
		Metadata: jsonMetadata{
			Created:       s.Metadata.Created.UTC().Format(time.RFC3339),
			Modified:      s.Metadata.Modified.UTC().Format(time.RFC3339),
			SourceFile:    s.Metadata.SourceFile,
			Description:   s.Metadata.Description,
			ArchiveOnSave: s.Metadata.ArchiveOnSave,
		},
		Binary: jsonBinary{
			Data:   s.Binary.Data,
			Origin: formatHex(s.Binary.Origin),
		},
		EntryPoints:        make([]string, len(s.EntryPoints)),
		ExtraCodeAddresses: make([]string, len(s.ExtraCodeAddresses)),
	}

	for i, ep := range s.EntryPoints {
		js.EntryPoints[i] = formatHex(ep)
	}
	for i, addr := range s.ExtraCodeAddresses {
		js.ExtraCodeAddresses[i] = formatHex(addr)
	}
	// Convert symbols
	if allSyms := s.Symbols.All(); len(allSyms) > 0 {
		js.Symbols = make(map[string]jsonSymbolValue)
		for addr, sym := range allSyms {
			addrStr := formatHex(addr)
			js.Symbols[addrStr] = jsonSymbolValue{{
				Name:   sym.Name,
				Type:   string(sym.Type),
				Source: string(sym.Source),
			}}
		}
	}

	// Convert annotations
	if allAnns := s.Annotations.All(); len(allAnns) > 0 {
		js.Annotations = make(map[string]*jsonAddressAnnotations)
		for addr, addrAnns := range allAnns {
			addrStr := formatHex(addr)
			jsonAddrAnns := &jsonAddressAnnotations{}

			if addrAnns.User != nil {
				jsonAddrAnns.User = &jsonAnnotation{Comment: addrAnns.User.Comment}
			}
			if addrAnns.Assistant != nil {
				jsonAddrAnns.Assistant = &jsonAnnotation{Comment: addrAnns.Assistant.Comment}
			}

			js.Annotations[addrStr] = jsonAddrAnns
		}
	}

	// Convert headlines
	if allHdls := s.Headlines.All(); len(allHdls) > 0 {
		js.Headlines = make(map[string]*jsonAddressHeadlines)
		for addr, addrHdls := range allHdls {
			addrStr := formatHex(addr)
			jsonAddrHdls := &jsonAddressHeadlines{}

			if addrHdls.User != nil {
				jsonAddrHdls.User = &jsonHeadline{Comment: addrHdls.User.Comment}
			}
			if addrHdls.Assistant != nil {
				jsonAddrHdls.Assistant = &jsonHeadline{Comment: addrHdls.Assistant.Comment}
			}

			js.Headlines[addrStr] = jsonAddrHdls
		}
	}

	// Convert regions
	if regs := s.Regions.Regions(); len(regs) > 0 {
		js.Regions = make([]jsonRegion, len(regs))
		for i, r := range regs {
			js.Regions[i] = jsonRegion{
				Start:  formatHex(r.Start),
				End:    formatHex(r.End),
				Type:   string(r.Type),
				Source: string(r.Source),
			}
		}
	}

	return js
}

// jsonToState converts a JSON representation to a State.
func jsonToState(js *jsonState) (*state.State, error) {
	created, err := time.Parse(time.RFC3339, js.Metadata.Created)
	if err != nil {
		return nil, fmt.Errorf("invalid created timestamp: %w", err)
	}
	modified, err := time.Parse(time.RFC3339, js.Metadata.Modified)
	if err != nil {
		return nil, fmt.Errorf("invalid modified timestamp: %w", err)
	}

	origin, err := parseHex(js.Binary.Origin)
	if err != nil {
		return nil, fmt.Errorf("invalid binary origin: %w", err)
	}

	s := &state.State{
		Version: js.Version,
		Metadata: state.Metadata{
			Created:       created,
			Modified:      modified,
			SourceFile:    js.Metadata.SourceFile,
			Description:   js.Metadata.Description,
			ArchiveOnSave: js.Metadata.ArchiveOnSave,
		},
		Binary: binary.Binary{
			Data:   js.Binary.Data,
			Origin: origin,
		},
		EntryPoints:        make([]uint16, len(js.EntryPoints)),
		ExtraCodeAddresses: make([]uint16, len(js.ExtraCodeAddresses)),
		Symbols:            symbols.NewTable(),
		Annotations:        annotations.NewTable(),
		Headlines:          headlines.NewTable(),
		Regions:            regions.NewTable(),
		XRefs:              xrefs.NewTable(),
	}

	for i, epStr := range js.EntryPoints {
		ep, err := parseHex(epStr)
		if err != nil {
			return nil, fmt.Errorf("invalid entry point: %w", err)
		}
		s.EntryPoints[i] = ep
	}
	for i, addrStr := range js.ExtraCodeAddresses {
		addr, err := parseHex(addrStr)
		if err != nil {
			return nil, fmt.Errorf("invalid extra code address: %w", err)
		}
		s.ExtraCodeAddresses[i] = addr
	}
	// Convert symbols (handles both old array format and new single-symbol format)
	for addrStr, symVal := range js.Symbols {
		addr, err := parseHex(addrStr)
		if err != nil {
			return nil, fmt.Errorf("invalid symbol address: %w", err)
		}
		// symVal is always a slice (either from array or single object)
		for _, sym := range symVal {
			if err := s.Symbols.Add(addr, symbols.Symbol{
				Name:   sym.Name,
				Type:   symbols.SymbolType(sym.Type),
				Source: symbols.SymbolSource(sym.Source),
			}); err != nil {
				return nil, fmt.Errorf("invalid symbol at %s: %w", addrStr, err)
			}
		}
	}

	// Convert annotations
	for addrStr, addrAnns := range js.Annotations {
		addr, err := parseHex(addrStr)
		if err != nil {
			return nil, fmt.Errorf("invalid annotation address: %w", err)
		}

		if addrAnns.User != nil {
			s.Annotations.Set(addr, addrAnns.User.Comment, author.User)
		}
		if addrAnns.Assistant != nil {
			s.Annotations.Set(addr, addrAnns.Assistant.Comment, author.Assistant)
		}
	}

	// Convert headlines
	for addrStr, addrHdls := range js.Headlines {
		addr, err := parseHex(addrStr)
		if err != nil {
			return nil, fmt.Errorf("invalid headline address: %w", err)
		}

		if addrHdls.User != nil {
			s.Headlines.Set(addr, addrHdls.User.Comment, author.User)
		}
		if addrHdls.Assistant != nil {
			s.Headlines.Set(addr, addrHdls.Assistant.Comment, author.Assistant)
		}
	}

	// Convert regions - default to full data region if empty
	if len(js.Regions) == 0 {
		s.Regions.SetRegions([]regions.Region{
			{Start: 0x0000, End: 0xFFFF, Type: regions.RegionData, Source: regions.RegionSourceAuto},
		})
	} else {
		regs := make([]regions.Region, len(js.Regions))
		for i, jr := range js.Regions {
			start, err := parseHex(jr.Start)
			if err != nil {
				return nil, fmt.Errorf("invalid region start: %w", err)
			}
			end, err := parseHex(jr.End)
			if err != nil {
				return nil, fmt.Errorf("invalid region end: %w", err)
			}
			regType := regions.RegionType(jr.Type)
			if regType != regions.RegionCode && regType != regions.RegionData {
				return nil, fmt.Errorf("invalid region type %q", jr.Type)
			}
			source := regions.RegionSource(jr.Source)
			if source == "" {
				source = regions.RegionSourceAuto
			}
			if source != regions.RegionSourceAuto &&
				source != regions.RegionSourceAssistant &&
				source != regions.RegionSourceUser {
				return nil, fmt.Errorf("invalid region source %q", jr.Source)
			}
			regs[i] = regions.Region{
				Start:  start,
				End:    end,
				Type:   regType,
				Source: source,
			}
		}
		s.Regions.SetRegions(regs)
		if err := s.Regions.Validate(); err != nil {
			return nil, fmt.Errorf("invalid regions: %w", err)
		}
	}

	// Legacy migration: forcedData -> high-priority data regions.
	for _, fr := range js.ForcedData {
		start, err := parseHex(fr.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid forcedData start: %w", err)
		}
		end, err := parseHex(fr.End)
		if err != nil {
			return nil, fmt.Errorf("invalid forcedData end: %w", err)
		}
		s.Regions.SetWithSource(start, end, regions.RegionData, regions.RegionSourceUser)
	}

	return s, nil
}
