package core_test

import (
	"context"
	"errors"
	"math"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chiranjeet-baruah/go-mcp-starter/internal/core"
)

func TestCalculatorAdd(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer stop()
	for _, tt := range []struct {
		name string
		ctx  context.Context
		a, b int64
		want int64
		err  error
	}{
		{name: "positive", a: 12, b: 30, want: 42},
		{name: "negative", a: -12, b: -30, want: -42},
		{name: "zero"},
		{name: "opposite signs", a: -12, b: 30, want: 18},
		{name: "maximum", a: math.MaxInt64 - 1, b: 1, want: math.MaxInt64},
		{name: "minimum", a: math.MinInt64 + 1, b: -1, want: math.MinInt64},
		{name: "opposite limits", a: math.MinInt64, b: math.MaxInt64, want: -1},
		{name: "positive overflow", a: math.MaxInt64, b: 1, err: core.ErrOverflow},
		{name: "negative overflow", a: math.MinInt64, b: -1, err: core.ErrOverflow},
		{name: "cancelled", ctx: cancelled, a: 1, b: 2, err: context.Canceled},
		{name: "expired deadline", ctx: expired, a: 1, b: 2, err: context.DeadlineExceeded},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			got, err := (core.Calculator{}).Add(ctx, tt.a, tt.b)
			if got != tt.want || !errors.Is(err, tt.err) {
				t.Fatalf("Add(%d, %d) = (%d, %v), want (%d, %v)", tt.a, tt.b, got, err, tt.want, tt.err)
			}
		})
	}
}

func TestCoreDependencyBoundary(t *testing.T) {
	for _, tt := range []struct{ name, target, want string }{
		{"core uses only the standard library", ".", "github.com/chiranjeet-baruah/go-mcp-starter/internal/core"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", tt.target)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("inspect dependencies: %v\n%s", err, out)
			}
			if got := strings.TrimSpace(string(out)); got != tt.want {
				t.Fatalf("core must depend only on the standard library; found:\n%s", got)
			}
		})
	}
}
