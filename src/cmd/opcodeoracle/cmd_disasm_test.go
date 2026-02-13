package main

import "testing"

func TestInclusiveToExclusiveEnd(t *testing.T) {
	tests := []struct {
		name string
		in   uint16
		want uint16
	}{
		{name: "normal", in: 0x1234, want: 0x1235},
		{name: "max_uint16", in: 0xFFFF, want: 0xFFFF},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := inclusiveToExclusiveEnd(tc.in)
			if got != tc.want {
				t.Fatalf("inclusiveToExclusiveEnd(%#04x) = %#04x, want %#04x", tc.in, got, tc.want)
			}
		})
	}
}
