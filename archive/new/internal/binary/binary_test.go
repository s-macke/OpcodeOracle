package binary

import (
	"os"
	"path/filepath"
	"testing"

	"opcodeoracle/internal/asm/x86"
)

func TestNewCopiesInput(t *testing.T) {
	data := []byte{0x01, 0x02}
	entryPoints := []x86.FarAddress{x86.NewFarAddress(0x1000, 0x0020)}

	b := New(data, x86.NewFarAddress(0x1000, 0x0000), x86.NewFarAddress(0x1000, 0xfffe), entryPoints)

	data[0] = 0xff
	entryPoints[0] = x86.NewFarAddress(0x2000, 0x0040)

	if b.Data[0] != 0x01 {
		t.Fatalf("data was not copied: %#v", b.Data)
	}
	if b.EntryPoints[0] != x86.NewFarAddress(0x1000, 0x0020) {
		t.Fatalf("entry point address = %+v", b.EntryPoints[0])
	}
	if b.Origin != x86.NewFarAddress(0x1000, 0x0000) {
		t.Fatalf("origin = %+v", b.Origin)
	}
	if b.DefaultSP != x86.NewFarAddress(0x1000, 0xfffe) {
		t.Fatalf("default sp = %+v", b.DefaultSP)
	}
}

func TestNewFromCOMFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.com")
	if err := os.WriteFile(path, []byte{0x90, 0xc3}, 0o644); err != nil {
		t.Fatal(err)
	}

	b, err := NewFromCOMFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(b.Data) != 2 || b.Data[0] != 0x90 || b.Data[1] != 0xc3 {
		t.Fatalf("data = %#v", b.Data)
	}
	expected := x86.NewFarAddress(DefaultCOMSegment, DefaultCOMOffset)
	if b.Origin != expected {
		t.Fatalf("origin = %+v", b.Origin)
	}
	if len(b.EntryPoints) != 1 {
		t.Fatalf("entry points = %#v", b.EntryPoints)
	}
	if b.EntryPoints[0] != expected {
		t.Fatalf("entry point = %#v", b.EntryPoints[0])
	}
	if b.DefaultSP != x86.NewFarAddress(DefaultCOMSegment, DefaultCOMSP) {
		t.Fatalf("default sp = %+v", b.DefaultSP)
	}
}

func TestNewFromMZFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.exe")

	header := make([]byte, 32)
	header[0] = 'M'
	header[1] = 'Z'
	header[2] = 0x24
	header[3] = 0x00
	header[4] = 0x01
	header[5] = 0x00
	header[6] = 0x01
	header[7] = 0x00
	header[8] = 0x02
	header[9] = 0x00
	header[14] = 0xbc
	header[15] = 0x9a
	header[16] = 0xf0
	header[17] = 0xde
	header[20] = 0x34
	header[21] = 0x12
	header[22] = 0x78
	header[23] = 0x56
	header[24] = 0x1c
	header[25] = 0x00
	header[28] = 0x00
	header[29] = 0x00
	header[30] = 0x00
	header[31] = 0x00
	fileData := append(header, []byte{0x34, 0x12, 0x90, 0xc3}...)
	if err := os.WriteFile(path, fileData, 0o644); err != nil {
		t.Fatal(err)
	}

	b, err := NewFromMZFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(b.Data) != 4 {
		t.Fatalf("data = %#v", b.Data)
	}
	if b.Data[0] != 0x34 || b.Data[1] != 0x22 || b.Data[2] != 0x90 || b.Data[3] != 0xc3 {
		t.Fatalf("relocated data = %#v", b.Data)
	}
	if b.Origin != x86.NewFarAddress(DefaultMZSegment, 0x0000) {
		t.Fatalf("origin = %+v", b.Origin)
	}
	expectedSP := x86.NewFarAddress(DefaultMZSegment+0x9abc, 0xdef0)
	if b.DefaultSP != expectedSP {
		t.Fatalf("default sp = %+v", b.DefaultSP)
	}
	expectedEntry := x86.NewFarAddress(DefaultMZSegment+0x5678, 0x1234)
	if len(b.EntryPoints) != 1 {
		t.Fatalf("entry points = %#v", b.EntryPoints)
	}
	if b.EntryPoints[0] != expectedEntry {
		t.Fatalf("entry point = %#v", b.EntryPoints[0])
	}
}

func TestNewFromMZFileIgnoresTrailingBytesBeyondDeclaredSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trailing.exe")

	header := make([]byte, 32)
	header[0] = 'M'
	header[1] = 'Z'
	header[2] = 0x24
	header[3] = 0x00
	header[4] = 0x01
	header[5] = 0x00
	header[8] = 0x02
	header[9] = 0x00
	header[20] = 0x00
	header[21] = 0x00
	header[22] = 0x00
	header[23] = 0x00
	fileData := append(header, []byte{0x90, 0xc3, 0xaa, 0xbb}...)
	if err := os.WriteFile(path, fileData, 0o644); err != nil {
		t.Fatal(err)
	}

	b, err := NewFromMZFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Data) != 4 {
		t.Fatalf("data length = %d", len(b.Data))
	}
	if b.Data[0] != 0x90 || b.Data[1] != 0xc3 || b.Data[2] != 0xaa || b.Data[3] != 0xbb {
		t.Fatalf("data = %#v", b.Data)
	}

	header[2] = 0x22
	header[3] = 0x00
	fileData = append(header, []byte{0x90, 0xc3, 0xaa, 0xbb, 0xcc, 0xdd}...)
	if err := os.WriteFile(path, fileData, 0o644); err != nil {
		t.Fatal(err)
	}

	b, err = NewFromMZFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Data) != 2 || b.Data[0] != 0x90 || b.Data[1] != 0xc3 {
		t.Fatalf("truncated data = %#v", b.Data)
	}
}

func TestNewFromMZFileSupportsFullLastPageWhenRemainderIsZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fullpage.exe")

	header := make([]byte, 32)
	header[0] = 'M'
	header[1] = 'Z'
	header[2] = 0x00
	header[3] = 0x00
	header[4] = 0x01
	header[5] = 0x00
	header[8] = 0x02
	header[9] = 0x00
	image := make([]byte, 512)
	image[0] = 0x90
	image[511] = 0xc3
	fileData := append(header, image...)
	if err := os.WriteFile(path, fileData, 0o644); err != nil {
		t.Fatal(err)
	}

	b, err := NewFromMZFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Data) != 480 {
		t.Fatalf("data length = %d", len(b.Data))
	}
	if b.Data[0] != 0x90 || b.Data[479] != 0x00 {
		t.Fatalf("data = %#v ... %#v", b.Data[:2], b.Data[len(b.Data)-2:])
	}
}

func TestNewFromMZFileRejectsDeclaredSizeSmallerThanHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.exe")

	header := make([]byte, 32)
	header[0] = 'M'
	header[1] = 'Z'
	header[2] = 0x10
	header[3] = 0x00
	header[4] = 0x01
	header[5] = 0x00
	header[8] = 0x02
	header[9] = 0x00
	if err := os.WriteFile(path, header, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := NewFromMZFile(path); err == nil {
		t.Fatal("expected error for declared size smaller than header")
	}
}

func TestNewFromMZFileRejectsDeclaredSizeLargerThanActual(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.exe")

	header := make([]byte, 32)
	header[0] = 'M'
	header[1] = 'Z'
	header[2] = 0x40
	header[3] = 0x00
	header[4] = 0x01
	header[5] = 0x00
	header[8] = 0x02
	header[9] = 0x00
	if err := os.WriteFile(path, append(header, 0x90), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := NewFromMZFile(path); err == nil {
		t.Fatal("expected error for declared size larger than actual")
	}
}

func TestBinaryDataAtForMZEntryPoint(t *testing.T) {
	b := New(
		[]byte{0xaa, 0xbb, 0xcc, 0xdd},
		x86.NewFarAddress(0x1000, 0x0000),
		x86.NewFarAddress(0x1000, 0xfffe),
		nil,
	)

	view, err := b.DataAt(x86.NewFarAddress(0x1000, 0x0002))
	if err != nil {
		t.Fatal(err)
	}
	if len(view) != 2 {
		t.Fatalf("view length = %d", len(view))
	}
	if view[0] != 0xcc || view[1] != 0xdd {
		t.Fatalf("view = %#v", view)
	}
}
