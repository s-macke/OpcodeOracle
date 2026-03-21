package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"opcodeoracle/internal/regions"
	"opcodeoracle/internal/symbols"

	"github.com/urfave/cli/v2"
)

func infoCommand() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Usage:     "Display state file information",
		ArgsUsage: "<state-file>",
		Action:    cmdInfo,
	}
}

func cmdInfo(c *cli.Context) error {
	if c.NArg() < 1 {
		_ = cli.ShowCommandHelp(c, "info")
		return cli.Exit("error: missing state file argument", ExitInvalidArgs)
	}

	stateFile := c.Args().Get(0)
	s, err := loadState(stateFile)
	if err != nil {
		return err
	}

	// Metadata
	fmt.Printf("Project:       %s\n", filepath.Base(stateFile))
	fmt.Printf("Source:        %s\n", s.Metadata.SourceFile)
	fmt.Printf("Created:       %s\n", s.Metadata.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("Modified:      %s\n", s.Metadata.Modified.Format("2006-01-02 15:04:05"))
	if s.Metadata.Description != "" {
		fmt.Printf("Description:   %s\n", s.Metadata.Description)
	}

	// Binary info
	origin := s.Binary.Origin
	end := s.Binary.End()
	size := len(s.Binary.Data)
	fmt.Printf("Origin:        $%04X\n", origin)
	fmt.Printf("Binary:        $%04X - $%04X (%d bytes)\n", origin, end, size)

	// Entry points
	if len(s.EntryPoints) > 0 {
		entries := make([]string, len(s.EntryPoints))
		for i, ep := range s.EntryPoints {
			entries[i] = fmt.Sprintf("$%04X", ep)
		}
		fmt.Printf("Entry points:  %s\n", strings.Join(entries, ", "))
	}
	if len(s.ExtraCodeAddresses) > 0 {
		addrs := make([]string, len(s.ExtraCodeAddresses))
		for i, addr := range s.ExtraCodeAddresses {
			addrs[i] = fmt.Sprintf("$%04X", addr)
		}
		fmt.Printf("Extra code:    %s\n", strings.Join(addrs, ", "))
	}

	// Symbol stats
	allSymbols := s.Symbols.All()
	symTotal := len(allSymbols)
	symCounts := map[symbols.SymbolType]int{}
	for _, sym := range allSymbols {
		symCounts[sym.Type]++
	}
	fmt.Printf("Symbols:       %d\n", symTotal)
	if symTotal > 0 {
		for _, st := range []symbols.SymbolType{
			symbols.SymbolSubroutine,
			symbols.SymbolLabel,
			symbols.SymbolByte,
			symbols.SymbolWord,
			symbols.SymbolEntry,
		} {
			if n := symCounts[st]; n > 0 {
				label := capitalize(string(st)) + ":"
				fmt.Printf("  %-13s%d\n", label, n)
			}
		}
	}

	// Annotation stats
	allAnnotations := s.Annotations.All()
	annTotal := len(allAnnotations)
	annUser, annAssistant := 0, 0
	for _, aa := range allAnnotations {
		if aa.User != nil {
			annUser++
		}
		if aa.Assistant != nil {
			annAssistant++
		}
	}
	fmt.Printf("Annotations:   %d\n", annTotal)
	if annTotal > 0 {
		if annUser > 0 {
			fmt.Printf("  User:        %d\n", annUser)
		}
		if annAssistant > 0 {
			fmt.Printf("  Assistant:   %d\n", annAssistant)
		}
	}

	// Headline stats
	allHeadlines := s.Headlines.All()
	hdlTotal := len(allHeadlines)
	hdlUser, hdlAssistant := 0, 0
	for _, ah := range allHeadlines {
		if ah.User != nil {
			hdlUser++
		}
		if ah.Assistant != nil {
			hdlAssistant++
		}
	}
	fmt.Printf("Headlines:     %d\n", hdlTotal)
	if hdlTotal > 0 {
		if hdlUser > 0 {
			fmt.Printf("  User:        %d\n", hdlUser)
		}
		if hdlAssistant > 0 {
			fmt.Printf("  Assistant:   %d\n", hdlAssistant)
		}
	}

	// Region stats — only count regions within the binary range
	allRegions := s.Regions.Regions()
	var codeBytes, dataBytes int
	var codeRegions, dataRegions int
	for _, r := range allRegions {
		if r.End < origin || r.Start > end {
			continue
		}
		rStart := r.Start
		rEnd := r.End
		if rStart < origin {
			rStart = origin
		}
		if rEnd > end {
			rEnd = end
		}
		sz := int(rEnd) - int(rStart) + 1
		switch r.Type {
		case regions.RegionCode:
			codeBytes += sz
			codeRegions++
		case regions.RegionData:
			dataBytes += sz
			dataRegions++
		}
	}
	totalRegions := codeRegions + dataRegions
	fmt.Printf("Regions:       %d\n", totalRegions)
	if codeRegions > 0 {
		fmt.Printf("  Code:        %d bytes (%d regions)\n", codeBytes, codeRegions)
	}
	if dataRegions > 0 {
		fmt.Printf("  Data:        %d bytes (%d regions)\n", dataBytes, dataRegions)
	}

	return nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
