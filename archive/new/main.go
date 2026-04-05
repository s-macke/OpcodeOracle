package main

import (
	"flag"
	"fmt"
	"os"

	"opcodeoracle/internal/analysis"
	binfile "opcodeoracle/internal/binary"
	"opcodeoracle/internal/disasm"
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

	dis := disasm.NewDisassembler()
	lines, err := dis.Disassemble(bin, result)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(dis.String(lines))
}
