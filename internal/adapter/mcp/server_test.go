package mcpadapter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	mcpadapter "github.com/chiranjeet-baruah/go-mcp-starter/internal/adapter/mcp"
	"github.com/chiranjeet-baruah/go-mcp-starter/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/time/rate"
)

type adderFunc func(context.Context, int64, int64) (int64, error)

func (f adderFunc) Add(ctx context.Context, a, b int64) (int64, error) {
	return f(ctx, a, b)
}

func connect(t *testing.T, adder core.Adder, limiter *rate.Limiter, logs io.Writer) (*mcp.ClientSession, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	server, err := mcpadapter.NewServer(adder, slog.New(slog.NewJSONHandler(logs, nil)), limiter)
	if err != nil {
		t.Fatal(err)
	}
	st, ct := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "adapter-test", Version: "0.1.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs, ctx
}

func unlimited() *rate.Limiter { return rate.NewLimiter(rate.Inf, 0) }

func TestToolContract(t *testing.T) {
	cs, ctx := connect(t, core.Calculator{}, unlimited(), io.Discard)
	info := cs.InitializeResult()
	list, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 1 || list.Tools[0].Name != "add" {
		t.Fatalf("unexpected catalog: %+v", list.Tools)
	}
	tool := list.Tools[0]
	a := tool.Annotations
	for _, tt := range []struct {
		name  string
		valid bool
	}{
		{"server identity", info.ServerInfo.Name == "go-mcp-starter" && info.ServerInfo.Version == "0.1.0"},
		{"static tool capability", info.Capabilities.Tools != nil && !info.Capabilities.Tools.ListChanged},
		{"unused capabilities absent", info.Capabilities.Logging == nil && info.Capabilities.Resources == nil && info.Capabilities.Prompts == nil},
		{"human-readable metadata", tool.Title != "" && tool.Description != ""},
		{"safety annotations", a != nil && a.ReadOnlyHint && a.IdempotentHint && a.DestructiveHint != nil && !*a.DestructiveHint && a.OpenWorldHint != nil && !*a.OpenWorldHint},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				t.Fatal("contract not satisfied")
			}
		})
	}
	for _, tt := range []struct {
		name   string
		schema any
		fields []string
	}{{"input schema", tool.InputSchema, []string{"a", "b"}}, {"output schema", tool.OutputSchema, []string{"sum"}}} {
		t.Run(tt.name, func(t *testing.T) {
			schema := tt.schema.(map[string]any)
			if schema["type"] != "object" || schema["additionalProperties"] != false {
				t.Fatalf("expected a closed object schema: %+v", schema)
			}
			required := schema["required"].([]any)
			properties := schema["properties"].(map[string]any)
			if len(required) != len(tt.fields) || len(properties) != len(tt.fields) {
				t.Fatalf("unexpected schema fields: %+v", schema)
			}
			for _, name := range tt.fields {
				p := properties[name].(map[string]any)
				if p["type"] != "integer" || p["minimum"] != -float64(mcpadapter.MaxSafeInteger) || p["maximum"] != float64(mcpadapter.MaxSafeInteger) {
					t.Fatalf("unexpected bounds for %s: %+v", name, p)
				}
			}
		})
	}
}

func TestAddResults(t *testing.T) {
	cs, ctx := connect(t, core.Calculator{}, unlimited(), io.Discard)
	for _, tt := range []struct {
		name       string
		a, b, want int64
	}{
		{"positive", 12, 30, 42}, {"negative", -12, -30, -42}, {"zero", 0, 0, 0}, {"opposite signs", -12, 30, 18},
		{"maximum safe integer", mcpadapter.MaxSafeInteger, 0, mcpadapter.MaxSafeInteger},
		{"minimum safe integer", -mcpadapter.MaxSafeInteger, 0, -mcpadapter.MaxSafeInteger},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "add", Arguments: map[string]int64{"a": tt.a, "b": tt.b}})
			if err != nil {
				t.Fatal(err)
			}
			checkSum(t, result, tt.want)
		})
	}
}

func checkSum(t *testing.T, result *mcp.CallToolResult, want int64) {
	t.Helper()
	if result.IsError || len(result.Content) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, data := range [][]byte{structured, []byte(text)} {
		var out struct {
			Sum int64 `json:"sum"`
		}
		if err := json.Unmarshal(data, &out); err != nil || out.Sum != want {
			t.Fatalf("got %s (%v), want sum %d", data, err, want)
		}
	}
}

func TestInvalidArguments(t *testing.T) {
	cs, ctx := connect(t, core.Calculator{}, unlimited(), io.Discard)
	for _, tt := range []struct {
		name, tool, input string
		protocolError     bool
	}{
		{"missing operands", "add", `{}`, false},
		{"missing b", "add", `{"a":1}`, false},
		{"null operand", "add", `{"a":null,"b":1}`, false},
		{"string operand", "add", `{"a":"1","b":2}`, false},
		{"fractional operand", "add", `{"a":1.5,"b":2}`, false},
		{"extra property", "add", `{"a":1,"b":2,"extra":3}`, false},
		{"array arguments", "add", `[]`, false},
		{"operand above maximum", "add", `{"a":9007199254740992,"b":0}`, false},
		{"operand below minimum", "add", `{"a":-9007199254740992,"b":0}`, false},
		{"sum above maximum", "add", `{"a":9007199254740991,"b":1}`, false},
		{"sum below minimum", "add", `{"a":-9007199254740991,"b":-1}`, false},
		{"unknown tool", "unknown", `{}`, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: tt.tool, Arguments: json.RawMessage(tt.input)})
			if tt.protocolError {
				if err == nil {
					t.Fatal("expected a protocol error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError || len(result.Content) == 0 || result.StructuredContent != nil {
				t.Fatalf("expected a tool error without a successful payload, got %+v", result)
			}
		})
	}
}

func TestPortDelegationAndErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
	}{
		{"delegates computation", nil}, {"translates core errors", core.ErrOverflow},
	} {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			port := adderFunc(func(_ context.Context, a, b int64) (int64, error) {
				called = true
				if a != 12 || b != 30 {
					t.Errorf("port received (%d, %d)", a, b)
				}
				if tt.err != nil {
					return 0, tt.err
				}
				return 7, nil
			})
			cs, ctx := connect(t, port, unlimited(), io.Discard)
			result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "add", Arguments: map[string]int{"a": 12, "b": 30}})
			if err != nil || !called {
				t.Fatalf("port not invoked successfully: %v", err)
			}
			if tt.err != nil {
				if !result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, core.ErrOverflow.Error()) {
					t.Fatalf("core error not translated: %+v", result)
				}
			} else {
				checkSum(t, result, 7)
			}
		})
	}
}

func TestRateLimitAndLogging(t *testing.T) {
	var logs bytes.Buffer
	cs, ctx := connect(t, core.Calculator{}, rate.NewLimiter(0, 1), &logs)
	params := &mcp.CallToolParams{Name: "add", Arguments: map[string]int{"a": 12345, "b": 67890}}
	for _, tt := range []struct {
		name      string
		wantError bool
	}{
		{"first call admitted", false}, {"exhausted bucket rejected", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cs.CallTool(ctx, params)
			if err != nil || result.IsError != tt.wantError {
				t.Fatalf("unexpected admission result: %+v, %v", result, err)
			}
			if !tt.wantError {
				checkSum(t, result, 80235)
			}
		})
	}
	text := logs.String()
	for _, tt := range []struct {
		name  string
		valid bool
	}{
		{"tool identity logged", strings.Contains(text, `"tool":"add"`)},
		{"failure outcome logged", strings.Contains(text, `"outcome":"tool_error"`)},
		{"duration logged", strings.Contains(text, `"duration":`)},
		{"payload omitted", !strings.Contains(text, "12345") && !strings.Contains(text, "67890") && !strings.Contains(text, "80235")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				t.Fatalf("logging contract failed: %s", text)
			}
		})
	}
}

func TestConcurrentCalls(t *testing.T) {
	for _, calls := range []int{1, 16} {
		t.Run(fmt.Sprintf("%d calls", calls), func(t *testing.T) {
			cs, ctx := connect(t, core.Calculator{}, unlimited(), io.Discard)
			var wg sync.WaitGroup
			for i := range calls {
				wg.Go(func() {
					result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "add", Arguments: map[string]int{"a": i, "b": 1}})
					if err != nil || result.IsError {
						t.Errorf("concurrent call failed: %+v, %v", result, err)
					}
				})
			}
			wg.Wait()
		})
	}
}

func TestPortCancellation(t *testing.T) {
	for _, tt := range []struct{ name string }{{"client cancellation reaches the core port"}} {
		t.Run(tt.name, func(t *testing.T) {
			started, stopped := make(chan struct{}), make(chan struct{})
			port := adderFunc(func(ctx context.Context, _, _ int64) (int64, error) {
				close(started)
				<-ctx.Done()
				close(stopped)
				return 0, ctx.Err()
			})
			cs, ctx := connect(t, port, unlimited(), io.Discard)
			callCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := cs.CallTool(callCtx, &mcp.CallToolParams{Name: "add", Arguments: map[string]int{"a": 1, "b": 2}})
				done <- err
			}()
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal("handler did not start")
			}
			cancel()
			if err := <-done; !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want cancellation", err)
			}
			select {
			case <-stopped:
			case <-ctx.Done():
				t.Fatal("port did not observe cancellation")
			}
		})
	}
}
