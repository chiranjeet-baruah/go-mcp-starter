package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	mcpadapter "github.com/chiranjeet-baruah/go-mcp-starter/internal/adapter/mcp"
	"github.com/chiranjeet-baruah/go-mcp-starter/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/time/rate"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := mcpadapter.NewServer(core.Calculator{}, logger, rate.NewLimiter(10, 20))
	if err == nil {
		err = server.Run(ctx, &mcp.StdioTransport{MaxLineLength: mcpadapter.MaxFrameBytes})
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
