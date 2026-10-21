// Package mcpadapter translates MCP requests into calls to the arithmetic port.
package mcpadapter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/chiranjeet-baruah/go-mcp-starter/internal/core"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/time/rate"
)

const (
	// MaxSafeInteger is the largest integer represented exactly by common JSON clients.
	MaxSafeInteger int64 = 1<<53 - 1
	// MaxFrameBytes bounds each incoming stdio message, including its metadata.
	MaxFrameBytes = 64 * 1024
)

type addInput struct {
	A int64 `json:"a" jsonschema:"First integer to add."`
	B int64 `json:"b" jsonschema:"Second integer to add."`
}

type addOutput struct {
	Sum int64 `json:"sum" jsonschema:"The exact sum of a and b."`
}

// NewServer constructs the driving adapter with its dependencies supplied by main.
func NewServer(adder core.Adder, logger *slog.Logger, limiter *rate.Limiter) (*mcp.Server, error) {
	input, err := jsonschema.For[addInput](nil)
	if err != nil {
		return nil, fmt.Errorf("infer add input schema: %w", err)
	}
	output, err := jsonschema.For[addOutput](nil)
	if err != nil {
		return nil, fmt.Errorf("infer add output schema: %w", err)
	}
	for _, schema := range []*jsonschema.Schema{input, output} {
		for _, property := range schema.Properties {
			property.Minimum = jsonschema.Ptr(-float64(MaxSafeInteger))
			property.Maximum = jsonschema.Ptr(float64(MaxSafeInteger))
		}
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "go-mcp-starter", Version: "0.1.0"}, &mcp.ServerOptions{
		Logger: logger,
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{ListChanged: false},
		},
	})
	closedWorld, nonDestructive := false, false
	mcp.AddTool(server, &mcp.Tool{
		Name:         "add",
		Title:        "Add two integers",
		Description:  "Add two integers exactly; each operand and the sum must be between -9007199254740991 and 9007199254740991, inclusive.",
		InputSchema:  input,
		OutputSchema: output,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: &nonDestructive,
			IdempotentHint:  true,
			OpenWorldHint:   &closedWorld,
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in addInput) (*mcp.CallToolResult, addOutput, error) {
		if err := ctx.Err(); err != nil {
			return nil, addOutput{}, err
		}
		if !limiter.Allow() {
			return nil, addOutput{}, errors.New("add rate limit exceeded; retry later (10 calls/second, burst 20)")
		}
		sum, err := adder.Add(ctx, in.A, in.B)
		if err != nil {
			return nil, addOutput{}, fmt.Errorf("cannot add these integers: %w", err)
		}
		if sum < -MaxSafeInteger || sum > MaxSafeInteger {
			return nil, addOutput{}, errors.New("sum must be between -9007199254740991 and 9007199254740991; use smaller operands")
		}
		return nil, addOutput{Sum: sum}, nil
	})
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}
			started := time.Now()
			result, err := next(ctx, method, req)
			outcome := "ok"
			if err != nil {
				outcome = "protocol_error"
			} else if call, ok := result.(*mcp.CallToolResult); ok && call.IsError {
				outcome = "tool_error"
			}
			tool := "unknown"
			if params, ok := req.GetParams().(*mcp.CallToolParamsRaw); ok && params != nil {
				tool = params.Name
			}
			logger.Info("tool call", "tool", tool, "duration", time.Since(started), "outcome", outcome)
			return result, err
		}
	})
	return server, nil
}
