package main

import (
	"flag"
	"fmt"
	"os"

	"opcodeoracle/internal/asm/x86"
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

	dec := x86.NewDecoder()
	for i, ep := range bin.EntryPoints {
		fmt.Printf("\nEntryPoint[%d]: %s\n", i, ep.String())
		view, err := bin.DataAt(ep)
		if err != nil {
			fmt.Fprintf(os.Stderr, "entrypoint[%d] translation failed: %v\n", i, err)
			continue
		}
		inst, err := dec.Decode(view, ep)
		if err != nil {
			fmt.Fprintf(os.Stderr, "entrypoint[%d] decode failed: %v\n", i, err)
			continue
		}
		fmt.Print(inst.DetailsString())
	}
}
