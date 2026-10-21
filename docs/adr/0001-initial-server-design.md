# Initial server design

## Local scope

Build `go-mcp-starter` as a local stdio server with one `add` tool to learn MCP through a complete tool call.
The application performs arithmetic without private data, external services, or persistent state, so HTTP hosting, OAuth, UI resources, and skills import are outside this version's scope.

The adapter restricts operands and results to the integers represented exactly by common JSON clients, while the core handles the full signed 64-bit range and detects overflow.
It limits incoming messages to 64 KiB and validated calls to 10 per second with a burst of 20, rejecting excess calls without queueing them.
These limits are project defaults, not values prescribed by MCP.

## Toolchain

Use `gvm use go1.27.3` in each shell and declare `go 1.27.3` in the module.
Use the [official Go SDK](https://github.com/modelcontextprotocol/go-sdk/tree/v1.8.0) for transport, discovery, legacy initialization, typed tool validation, and structured results.
Use the SDK's existing `jsonschema-go` dependency to infer and bound schemas and `golang.org/x/time/rate` for a concurrency-safe token bucket.
Pin direct dependencies in `go.mod` and retain checksums in `go.sum`.

## SDK release policy

Keep published SDK v1.8.0 rather than pinning an unreleased upstream commit, as explicitly selected during planning.
This release supports the modern and legacy lifecycles, but its stdio transport can reply to a cancelled request under MCP 2026-07-28.
The [upstream cancellation fix](https://github.com/modelcontextprotocol/go-sdk/commit/3ebc6d1eb7206d50590341f409e1499676db7345) is not in v1.8.0.
Test application cancellation and normal lifecycles without claiming that the dependency fully conforms to the July revision.
Revisit the pin when a published release contains the fix.

## MCP references

- [Tool schemas, annotations, results, and errors](https://modelcontextprotocol.io/specification/2026-07-28/server/tools)
- [Stdio framing, logging, cancellation, and shutdown](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio)
- [Go SDK tool registration and validation](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/docs/server.md)
