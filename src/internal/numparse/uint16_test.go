package numparse

import "testing"

func TestParseUint16(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint16
		wantErr bool
	}{
		{name: "decimal", input: "1234", want: 1234},
		{name: "decimal_max", input: "65535", want: 0xFFFF},
		{name: "hex_dollar", input: "$C000", want: 0xC000},
		{name: "hex_0x", input: "0xC000", want: 0xC000},
		{name: "hex_0X", input: "0XC000", want: 0xC000},
		{name: "reject_bare_hex", input: "C000", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "invalid_hex", input: "0xGGGG", wantErr: true},
		{name: "overflow", input: "70000", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseUint16(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseUint16(%q) error = %v, wantErr = %v", tc.input, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Fatalf("ParseUint16(%q) = %04X, want %04X", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseHexUint16(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint16
		wantErr bool
	}{
		{name: "0x_prefix", input: "0x0801", want: 0x0801},
		{name: "0X_prefix", input: "0XABCD", want: 0xABCD},
		{name: "bare_hex_upper", input: "ABCD", want: 0xABCD},
		{name: "bare_hex_lower", input: "abcd", want: 0xABCD},
		{name: "empty", input: "", wantErr: true},
		{name: "invalid", input: "0xGGGG", wantErr: true},
		{name: "overflow", input: "0x10000", wantErr: true},
		{name: "reject_dollar_prefix", input: "$C000", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseHexUint16(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseHexUint16(%q) error = %v, wantErr = %v", tc.input, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Fatalf("ParseHexUint16(%q) = %04X, want %04X", tc.input, got, tc.want)
			}
		})
	}
}
