package author

import (
	"testing"
)

func TestAuthorString(t *testing.T) {
	tests := []struct {
		author Author
		want   string
	}{
		{User, "user"},
		{Assistant, "assistant"},
		{Author(99), "unknown"},
	}

	for _, tc := range tests {
		got := tc.author.String()
		if got != tc.want {
			t.Errorf("Author(%d).String() = %q, want %q", tc.author, got, tc.want)
		}
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		want    Author
		wantErr bool
	}{
		{"user", User, false},
		{"assistant", Assistant, false},
		{"User", 0, true},
		{"ASSISTANT", 0, true},
		{"auto", 0, true},
		{"", 0, true},
	}

	for _, tc := range tests {
		got, err := Parse(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("Parse(%q) error = %v, wantErr = %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("Parse(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
