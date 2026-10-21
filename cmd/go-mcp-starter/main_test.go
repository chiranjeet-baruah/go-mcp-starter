package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	mcpadapter "github.com/chiranjeet-baruah/go-mcp-starter/internal/adapter/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var serverBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "go-mcp-starter-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	serverBinary = filepath.Join(dir, "go-mcp-starter")
	if runtime.GOOS == "windows" {
		serverBinary += ".exe"
	}
	if out, err := exec.Command("go", "build", "-race", "-o", serverBinary, ".").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build integration binary: %v\n%s", err, out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func TestStdioServer(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25"} {
		t.Run(version, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, serverBinary)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			client := mcp.NewClient(&mcp.Implementation{Name: "stdio-test", Version: "0.1.0"}, nil)
			session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd, TerminateDuration: 3 * time.Second}, &mcp.ClientSessionOptions{ProtocolVersion: version})
			if err != nil {
				t.Fatalf("connect: %v", err)
			}
			t.Cleanup(func() { session.Close() })
			if got := session.InitializeResult().ProtocolVersion; got != version {
				t.Fatalf("negotiated %s, want %s", got, version)
			}
			tools, err := session.ListTools(ctx, nil)
			if err != nil || len(tools.Tools) != 1 || tools.Tools[0].Name != "add" {
				t.Fatalf("unexpected tools: %+v, %v", tools, err)
			}
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "add", Arguments: map[string]int{"a": 12, "b": 30}})
			if err != nil || result.IsError {
				t.Fatalf("call failed: %+v, %v", result, err)
			}
			if got := result.Content[0].(*mcp.TextContent).Text; got != `{"sum":42}` {
				t.Fatalf("got %s, want sum 42", got)
			}
			if err := session.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}
			if !cmd.ProcessState.Success() {
				t.Fatalf("server did not exit cleanly: %s", stderr.String())
			}
			if !strings.Contains(stderr.String(), "server session ended") || strings.Contains(stderr.String(), "server run cancelled") {
				t.Fatalf("server did not stop on EOF: %s", stderr.String())
			}
		})
	}
}

func TestInputFrames(t *testing.T) {
	for _, tt := range []struct {
		name, input, wantLog string
		wantError            bool
	}{
		{"empty input closes cleanly", "", "server session ended", false},
		{"malformed JSON is rejected", "{broken\n", "server session ended with error", true},
		{"oversized frame is rejected", `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"padding":"` + strings.Repeat("x", mcpadapter.MaxFrameBytes) + "\"}}\n", "maximum line length", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, serverBinary)
			cmd.Stdin = strings.NewReader(tt.input)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			if (err != nil) != tt.wantError || ctx.Err() != nil {
				t.Fatalf("unexpected process result: %v; stderr: %s", err, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("unexpected stdout: %s", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.wantLog) {
				t.Fatalf("missing diagnostic %q: %s", tt.wantLog, stderr.String())
			}
		})
	}
}

func TestSignalShutdown(t *testing.T) {
	for _, signal := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(signal.String(), func(t *testing.T) {
			if runtime.GOOS == "windows" {
				t.Skip("POSIX process signals")
			}
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, serverBinary)
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			defer stdin.Close()
			stderr, err := cmd.StderrPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			var event map[string]any
			if err := json.NewDecoder(stderr).Decode(&event); err != nil {
				t.Fatal(err)
			}
			if event["msg"] != "server run start" {
				t.Fatalf("unexpected startup event: %+v", event)
			}
			if err := cmd.Process.Signal(signal); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Wait(); err != nil {
				t.Fatalf("shutdown failed: %v", err)
			}
		})
	}
}
