package main

import (
	"testing"

	"opcodeoracle/internal/mcpserver"
)

func TestMCPCommandIncludesVerboseFlag(t *testing.T) {
	cmd := mcpCommand()
	if cmd == nil {
		t.Fatal("mcpCommand() = nil")
	}

	found := false
	for _, flag := range cmd.Flags {
		names := flag.Names()
		for _, name := range names {
			if name == "verbose" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("expected mcp command to expose --verbose flag")
	}
}

func TestBuildMCPRunOptions(t *testing.T) {
	tests := []struct {
		name      string
		transport string
		listen    string
		path      string
		want      mcpserver.RunOptions
		wantErr   bool
	}{
		{
			name:      "default_stdio",
			transport: "stdio",
			want:      mcpserver.RunOptions{Transport: mcpserver.TransportStdio},
		},
		{
			name:      "empty_transport_defaults_to_stdio",
			transport: "",
			want:      mcpserver.RunOptions{Transport: mcpserver.TransportStdio},
		},
		{
			name:      "http_valid",
			transport: "http",
			listen:    "127.0.0.1:8080",
			path:      "/mcp",
			want: mcpserver.RunOptions{
				Transport:  mcpserver.TransportHTTP,
				ListenAddr: "127.0.0.1:8080",
				Path:       "/mcp",
			},
		},
		{
			name:      "http_missing_listen",
			transport: "http",
			path:      "/mcp",
			wantErr:   true,
		},
		{
			name:      "http_empty_path",
			transport: "http",
			listen:    "127.0.0.1:8080",
			path:      "",
			wantErr:   true,
		},
		{
			name:      "http_invalid_path",
			transport: "http",
			listen:    "127.0.0.1:8080",
			path:      "mcp",
			wantErr:   true,
		},
		{
			name:      "invalid_transport",
			transport: "sse",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildMCPRunOptions(tc.transport, tc.listen, tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("buildMCPRunOptions(%q, %q, %q) error = nil, want error", tc.transport, tc.listen, tc.path)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildMCPRunOptions(%q, %q, %q) error = %v, want nil", tc.transport, tc.listen, tc.path, err)
			}
			if got != tc.want {
				t.Fatalf("buildMCPRunOptions(%q, %q, %q) = %+v, want %+v", tc.transport, tc.listen, tc.path, got, tc.want)
			}
		})
	}
}
