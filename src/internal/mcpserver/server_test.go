package mcpserver

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"opcodeoracle/internal/agent/tools"
	"opcodeoracle/internal/state"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRunHTTPValidation(t *testing.T) {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "test-server", Version: "0.0.1"},
		nil,
	)

	tests := []struct {
		name    string
		opts    RunOptions
		wantErr bool
	}{
		{
			name: "missing_listen",
			opts: RunOptions{
				Transport: TransportHTTP,
				Path:      "/mcp",
			},
			wantErr: true,
		},
		{
			name: "missing_path",
			opts: RunOptions{
				Transport:  TransportHTTP,
				ListenAddr: "127.0.0.1:8080",
			},
			wantErr: true,
		},
		{
			name: "invalid_path",
			opts: RunOptions{
				Transport:  TransportHTTP,
				ListenAddr: "127.0.0.1:8080",
				Path:       "mcp",
			},
			wantErr: true,
		},
		{
			name: "invalid_transport",
			opts: RunOptions{
				Transport: "bad",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var logOut bytes.Buffer
			cfg := &Config{
				ToolCtx: &tools.Context{
					State: &state.State{},
				},
				Output: &logOut,
			}

			err := Run(context.Background(), server, cfg, tc.opts)
			if tc.wantErr && err == nil {
				t.Fatalf("Run(..., %+v) error = nil, want error", tc.opts)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("Run(..., %+v) error = %v, want nil", tc.opts, err)
			}
			if tc.name == "invalid_transport" && !strings.Contains(logOut.String(), `invalid MCP transport: "bad"`) {
				t.Fatalf("expected invalid transport log, got: %q", logOut.String())
			}
		})
	}
}

func TestDelegateLoggingDefault(t *testing.T) {
	var logOut bytes.Buffer
	cfg := &Config{
		ToolCtx: &tools.Context{State: &state.State{}},
		Output:  &logOut,
	}

	_, _, err := delegate(context.Background(), stubInvokable{result: "ok"}, map[string]string{"address": "$C000"}, "query_xrefs", cfg)
	if err != nil {
		t.Fatalf("delegate error = %v, want nil", err)
	}

	logs := logOut.String()
	if !strings.Contains(logs, "tool query_xrefs ok") {
		t.Fatalf("expected success summary log, got: %q", logs)
	}
	if strings.Contains(logs, "args=") {
		t.Fatalf("expected args to be omitted in non-verbose logs, got: %q", logs)
	}
}

func TestDelegateLoggingVerboseIncludesArgsAndResult(t *testing.T) {
	var logOut bytes.Buffer
	cfg := &Config{
		ToolCtx:   &tools.Context{State: &state.State{}},
		Output:    &logOut,
		Verbose:   true,
		StatePath: "state.orc",
	}

	_, _, err := delegate(context.Background(), stubInvokable{result: "tool-result"}, map[string]string{"address": "$C000"}, "query_xrefs", cfg)
	if err != nil {
		t.Fatalf("delegate error = %v, want nil", err)
	}

	logs := logOut.String()
	if !strings.Contains(logs, "tool query_xrefs args=") {
		t.Fatalf("expected args in verbose logs, got: %q", logs)
	}
	if !strings.Contains(logs, "tool query_xrefs result=") {
		t.Fatalf("expected result preview in verbose logs, got: %q", logs)
	}
}

func TestDelegateLoggingError(t *testing.T) {
	var logOut bytes.Buffer
	cfg := &Config{
		ToolCtx: &tools.Context{State: &state.State{}},
		Output:  &logOut,
	}

	_, _, err := delegate(context.Background(), stubInvokable{err: errors.New("boom")}, map[string]string{"address": "$C000"}, "query_xrefs", cfg)
	if err == nil {
		t.Fatal("delegate error = nil, want error")
	}
	if !strings.Contains(logOut.String(), "tool query_xrefs failed") {
		t.Fatalf("expected failure log, got: %q", logOut.String())
	}
}

func TestDelegateAndSaveVerboseDryRunSkip(t *testing.T) {
	var logOut bytes.Buffer
	cfg := &Config{
		ToolCtx: &tools.Context{
			State:  &state.State{},
			DryRun: true,
		},
		Output:  &logOut,
		Verbose: true,
	}

	_, _, err := delegateAndSave(context.Background(), stubInvokable{result: "ok"}, map[string]string{"address": "$C000"}, "add_symbol", cfg)
	if err != nil {
		t.Fatalf("delegateAndSave error = %v, want nil", err)
	}
	if !strings.Contains(logOut.String(), "auto-save skipped after add_symbol: dry-run enabled") {
		t.Fatalf("expected dry-run skip log, got: %q", logOut.String())
	}
}

type stubInvokable struct {
	result string
	err    error
}

func (s stubInvokable) InvokableRun(context.Context, string, ...einotool.Option) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.result, nil
}
