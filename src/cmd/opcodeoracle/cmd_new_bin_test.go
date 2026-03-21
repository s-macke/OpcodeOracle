package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"opcodeoracle/internal/stateio"

	"github.com/urfave/cli/v2"
)

func TestCmdNewBinTruncatesDataToAddressSpace(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("restore Chdir() error = %v", err)
		}
	})

	binaryPath := filepath.Join(tmpDir, "oversize.bin")

	const (
		skip   = 3
		origin = 0x1000
		extra  = 0x20
	)

	maxDataLen := maxAddressSpaceSize - origin
	fileData := make([]byte, skip+maxDataLen+extra)
	for i := range fileData {
		fileData[i] = byte(i)
	}
	if err := os.WriteFile(binaryPath, fileData, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.String("skip", "0", "")
	flagSet.String("entry", "", "")
	flagSet.String("origin", "0", "")
	flagSet.String("description", "", "")
	if err := flagSet.Set("skip", "3"); err != nil {
		t.Fatalf("flagSet.Set(skip) error = %v", err)
	}
	if err := flagSet.Set("entry", "$1000"); err != nil {
		t.Fatalf("flagSet.Set(entry) error = %v", err)
	}
	if err := flagSet.Set("origin", "$1000"); err != nil {
		t.Fatalf("flagSet.Set(origin) error = %v", err)
	}

	ctx := cli.NewContext(cli.NewApp(), flagSet, nil)
	if err := cmdNewBin(ctx, binaryPath); err != nil {
		t.Fatalf("cmdNewBin() error = %v", err)
	}

	s, err := stateio.Load(filepath.Join(tmpDir, outputFilename(binaryPath)))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := len(s.Binary.Data); got != maxDataLen {
		t.Fatalf("len(Binary.Data) = %d, want %d", got, maxDataLen)
	}

	if got, want := s.Binary.Origin, uint16(origin); got != want {
		t.Fatalf("Binary.Origin = %04X, want %04X", got, want)
	}

	if got, want := s.Binary.Data[0], fileData[skip]; got != want {
		t.Fatalf("Binary.Data[0] = %02X, want %02X", got, want)
	}

	last := len(s.Binary.Data) - 1
	if got, want := s.Binary.Data[last], fileData[skip+last]; got != want {
		t.Fatalf("Binary.Data[last] = %02X, want %02X", got, want)
	}
}

func TestOutputFilenameUsesCurrentDirectoryBasename(t *testing.T) {
	got := outputFilename("/tmp/input/demo.prg")
	want := "demo.opcodeoracle.json"
	if got != want {
		t.Fatalf("outputFilename() = %q, want %q", got, want)
	}
}
