package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"opcodeoracle/internal/analysis"
	binfile "opcodeoracle/internal/binary"
)

func main() {
	filePath := flag.String("file", "", "path to a .com or DOS MZ .exe file")
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "missing input: provide -file")
		os.Exit(1)
	}

	bin, err := binfile.LoadFile(*filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Origin:      %s\n", bin.Origin.String())
	fmt.Printf("DefaultSP:   %s\n", bin.DefaultSP.String())
	fmt.Printf("EntryPoints:\n")
	for i, ep := range bin.EntryPoints {
		fmt.Printf("  [%d] %s\n", i, ep.String())
	}

	result, err := analysis.NewAnalyzer().Analyze(bin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, stop := range result.DecodeStops {
		fmt.Fprintf(os.Stderr, "warning: decode stopped at %s: %v\n", stop.Address.String(), stop.Err)
	}

	addresses := sortedInstructionAddresses(result)
	fmt.Printf("Instructions: %d\n", len(addresses))
	for _, linear := range addresses {
		inst := result.Instructions[linear]
		fmt.Printf("%s  %s\n", inst.Address.String(), inst.Text)
	}
}

func sortedInstructionAddresses(result analysis.Result) []uint32 {
	addresses := make([]uint32, 0, len(result.Instructions))
	for linear := range result.Instructions {
		addresses = append(addresses, linear)
	}
	sort.Slice(addresses, func(i, j int) bool {
		return addresses[i] < addresses[j]
	})
	return addresses
}
