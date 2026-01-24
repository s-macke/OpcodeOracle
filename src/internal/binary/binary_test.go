package binary

import (
	"testing"
)

func TestReadByte(t *testing.T) {
	b := Binary{
		Data:   []byte{0x10, 0x20, 0x30, 0x40},
		Origin: 0x0800,
	}

	tests := []struct {
		addr    uint16
		want    byte
		wantErr bool
	}{
		{0x0800, 0x10, false},
		{0x0801, 0x20, false},
		{0x0803, 0x40, false},
		{0x07FF, 0, true}, // below origin
		{0x0804, 0, true}, // beyond end
	}

	for _, tt := range tests {
		got, err := b.ReadByte(tt.addr)
		if (err != nil) != tt.wantErr {
			t.Errorf("ReadByte(%04X) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ReadByte(%04X) = %02X, want %02X", tt.addr, got, tt.want)
		}
	}
}

func TestReadWord(t *testing.T) {
	b := Binary{
		Data:   []byte{0x10, 0x20, 0x30, 0x40},
		Origin: 0x0800,
	}

	tests := []struct {
		addr    uint16
		want    uint16
		wantErr bool
	}{
		{0x0800, 0x2010, false}, // little-endian: 0x10, 0x20
		{0x0801, 0x3020, false},
		{0x0802, 0x4030, false},
		{0x0803, 0, true}, // not enough bytes
		{0x07FF, 0, true}, // below origin
	}

	for _, tt := range tests {
		got, err := b.ReadWord(tt.addr)
		if (err != nil) != tt.wantErr {
			t.Errorf("ReadWord(%04X) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ReadWord(%04X) = %04X, want %04X", tt.addr, got, tt.want)
		}
	}
}

func TestIsEntryPoint(t *testing.T) {
	b := Binary{
		Data:        []byte{0x10, 0x20, 0x30, 0x40},
		Origin:      0x0800,
		EntryPoints: []EntryPoint{0x0800, 0x0810},
	}

	tests := []struct {
		addr uint16
		want bool
	}{
		{0x0800, true},
		{0x0810, true},
		{0x0801, false},
		{0x0000, false},
	}

	for _, tt := range tests {
		got := b.IsEntryPoint(tt.addr)
		if got != tt.want {
			t.Errorf("IsEntryPoint(%04X) = %v, want %v", tt.addr, got, tt.want)
		}
	}
}
