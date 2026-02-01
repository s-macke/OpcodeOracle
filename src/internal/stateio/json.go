package stateio

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"opcodeoracle/internal/annotations"
	"opcodeoracle/internal/binary"
	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/state"
	"opcodeoracle/internal/symbols"
	"opcodeoracle/internal/xrefs"
)

// jsonState is the JSON representation of State.
type jsonState struct {
	Version     string                             `json:"version"`
	Metadata    jsonMetadata                       `json:"metadata"`
	Binary      jsonBinary                         `json:"binary"`
	EntryPoints []string                           `json:"entryPoints"`
	Symbols     map[string][]jsonSymbol            `json:"symbols,omitempty"`
	Annotations map[string]*jsonAddressAnnotations `json:"annotations,omitempty"`
	Regions     []jsonRegion                       `json:"regions,omitempty"`
}

type jsonMetadata struct {
	Created     string `json:"created"`
	Modified    string `json:"modified"`
	SourceFile  string `json:"sourceFile,omitempty"`
	Description string `json:"description,omitempty"`
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
	Type    string `json:"type"`
	Comment string `json:"comment"`
}

type jsonAddressAnnotations struct {
	User      *jsonAnnotation `json:"user,omitempty"`
	Assistant *jsonAnnotation `json:"assistant,omitempty"`
}

type jsonRegion struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Type  string `json:"type"`
}

// parseHex parses a hex string like "0x0801" to uint16.
func parseHex(s string) (uint16, error) {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	val, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid hex address %q: %w", s, err)
	}
	return uint16(val), nil
}

// formatHex formats a uint16 as "0xNNNN".
func formatHex(v uint16) string {
	return fmt.Sprintf("0x%04X", v)
}

// stateToJSON converts a State to its JSON representation.
func stateToJSON(s *state.State) *jsonState {
	js := &jsonState{
		Version: s.Version,
		Metadata: jsonMetadata{
			Created:     s.Metadata.Created.UTC().Format(time.RFC3339),
			Modified:    s.Metadata.Modified.UTC().Format(time.RFC3339),
			SourceFile:  s.Metadata.SourceFile,
			Description: s.Metadata.Description,
		},
		Binary: jsonBinary{
			Data:   s.Binary.Data,
			Origin: formatHex(s.Binary.Origin),
		},
		EntryPoints: make([]string, len(s.EntryPoints)),
	}

	for i, ep := range s.EntryPoints {
		js.EntryPoints[i] = formatHex(ep)
	}

	// Convert symbols
	if s.Symbols != nil {
		allSyms := s.Symbols.All()
		if len(allSyms) > 0 {
			js.Symbols = make(map[string][]jsonSymbol)
			for addr, syms := range allSyms {
				addrStr := formatHex(addr)
				for _, sym := range syms {
					js.Symbols[addrStr] = append(js.Symbols[addrStr], jsonSymbol{
						Name:   sym.Name,
						Type:   string(sym.Type),
						Source: string(sym.Source),
					})
				}
			}
		}
	}

	// Convert annotations
	if s.Annotations != nil {
		allAnns := s.Annotations.All()
		if len(allAnns) > 0 {
			js.Annotations = make(map[string]*jsonAddressAnnotations)
			for addr, addrAnns := range allAnns {
				addrStr := formatHex(addr)
				jsonAddrAnns := &jsonAddressAnnotations{}

				if addrAnns.User != nil {
					jsonAddrAnns.User = &jsonAnnotation{
						Type:    annotationTypeToString(addrAnns.User.Type),
						Comment: addrAnns.User.Comment,
					}
				}
				if addrAnns.Assistant != nil {
					jsonAddrAnns.Assistant = &jsonAnnotation{
						Type:    annotationTypeToString(addrAnns.Assistant.Type),
						Comment: addrAnns.Assistant.Comment,
					}
				}

				js.Annotations[addrStr] = jsonAddrAnns
			}
		}
	}

	// Convert regions
	if s.Regions != nil {
		regs := s.Regions.Regions()
		if len(regs) > 0 {
			js.Regions = make([]jsonRegion, len(regs))
			for i, r := range regs {
				js.Regions[i] = jsonRegion{
					Start: formatHex(r.Start),
					End:   formatHex(r.End),
					Type:  string(r.Type),
				}
			}
		}
	}

	return js
}

func annotationTypeToString(t annotations.AnnotationType) string {
	switch t {
	case annotations.AnnotationInline:
		return "inline"
	case annotations.AnnotationHeadline:
		return "headline"
	default:
		return "inline"
	}
}

func stringToAnnotationType(s string) (annotations.AnnotationType, error) {
	switch s {
	case "inline":
		return annotations.AnnotationInline, nil
	case "headline":
		return annotations.AnnotationHeadline, nil
	default:
		return 0, fmt.Errorf("invalid annotation type %q", s)
	}
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
			Created:     created,
			Modified:    modified,
			SourceFile:  js.Metadata.SourceFile,
			Description: js.Metadata.Description,
		},
		Binary: binary.Binary{
			Data:   js.Binary.Data,
			Origin: origin,
		},
		EntryPoints: make([]uint16, len(js.EntryPoints)),
		Symbols:     symbols.NewTable(),
		Annotations: annotations.NewTable(),
		Regions:     regions.NewTable(),
		XRefs:       xrefs.NewTable(),
	}

	for i, epStr := range js.EntryPoints {
		ep, err := parseHex(epStr)
		if err != nil {
			return nil, fmt.Errorf("invalid entry point: %w", err)
		}
		s.EntryPoints[i] = ep
	}

	// Convert symbols
	for addrStr, syms := range js.Symbols {
		addr, err := parseHex(addrStr)
		if err != nil {
			return nil, fmt.Errorf("invalid symbol address: %w", err)
		}
		for _, sym := range syms {
			s.Symbols.Add(addr, symbols.Symbol{
				Name:   sym.Name,
				Type:   symbols.SymbolType(sym.Type),
				Source: symbols.SymbolSource(sym.Source),
			})
		}
	}

	// Convert annotations
	for addrStr, addrAnns := range js.Annotations {
		addr, err := parseHex(addrStr)
		if err != nil {
			return nil, fmt.Errorf("invalid annotation address: %w", err)
		}

		if addrAnns.User != nil {
			annType, err := stringToAnnotationType(addrAnns.User.Type)
			if err != nil {
				return nil, fmt.Errorf("at address %s user: %w", addrStr, err)
			}
			s.Annotations.Set(addr, annType, addrAnns.User.Comment, annotations.AuthorUser)
		}

		if addrAnns.Assistant != nil {
			annType, err := stringToAnnotationType(addrAnns.Assistant.Type)
			if err != nil {
				return nil, fmt.Errorf("at address %s assistant: %w", addrStr, err)
			}
			s.Annotations.Set(addr, annType, addrAnns.Assistant.Comment, annotations.AuthorAssistant)
		}
	}

	// Convert regions - default to full data region if empty
	if len(js.Regions) == 0 {
		s.Regions.SetRegions([]regions.Region{
			{Start: 0x0000, End: 0xFFFF, Type: regions.RegionData},
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
			var regType regions.RegionType
			switch jr.Type {
			case "code":
				regType = regions.RegionCode
			case "data":
				regType = regions.RegionData
			default:
				return nil, fmt.Errorf("invalid region type %q", jr.Type)
			}
			regs[i] = regions.Region{
				Start: start,
				End:   end,
				Type:  regType,
			}
		}
		s.Regions.SetRegions(regs)
		if err := s.Regions.Validate(); err != nil {
			return nil, fmt.Errorf("invalid regions: %w", err)
		}
	}

	return s, nil
}
