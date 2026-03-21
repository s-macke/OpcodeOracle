package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"opcodeoracle/internal/state"
	"opcodeoracle/internal/stateio"

	"github.com/urfave/cli/v2"
)

func TestCmdInfoShowsExtraCodeAddresses(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "project.opcodeoracle.json")

	s := state.NewState([]byte{0xEA}, 0x0800, []uint16{0x0800}, "project.prg")
	s.ExtraCodeAddresses = []uint16{0x0810, 0x0900}
	if err := stateio.Save(s, statePath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	ctx := cli.NewContext(cli.NewApp(), flagSet, nil)
	if err := flagSet.Parse([]string{statePath}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	out := captureStdout(t, func() {
		if err := cmdInfo(ctx); err != nil {
			t.Fatalf("cmdInfo() error = %v", err)
		}
	})

	if !strings.Contains(out, "Entry points:  $0800") {
		t.Fatalf("expected info output to include entry points, got:\n%s", out)
	}
	if !strings.Contains(out, "Extra code:    $0810, $0900") {
		t.Fatalf("expected info output to include extra code addresses, got:\n%s", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy() error = %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	return buf.String()
}
