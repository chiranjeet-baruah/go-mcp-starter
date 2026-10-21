# go-mcp-starter

Build and call a local MCP tool with Go.
The server exposes `add(a, b)` over stdio and returns the sum as structured JSON and text.

## Run your first call

Allow about five minutes for the first run, depending on dependency downloads.
You need GVM with Go 1.27.3 installed, plus Node.js 22.19.0 or newer for MCP Inspector.

1. Select Go 1.27.3.

   ```sh
   source "$HOME/.gvm/scripts/gvm"
   gvm use go1.27.3
   go version
   ```

2. Build the server from the repository root.

   ```sh
   go build -o bin/go-mcp-starter ./cmd/go-mcp-starter
   ```

3. Discover the tool with the pinned Inspector CLI.

   ```sh
   npx --yes @modelcontextprotocol/inspector@2.11.0 --cli \
     ./bin/go-mcp-starter --method tools/list --format json
   ```

   The response lists exactly one tool: `add`.

4. Call it.

   ```sh
   npx --yes @modelcontextprotocol/inspector@2.11.0 --cli \
     ./bin/go-mcp-starter --method tools/call --tool-name add \
     --tool-args-json '{"a":12,"b":30}' --format json
   ```

   Look for `"structuredContent":{"sum":42}` in the response.

Running the binary directly waits for MCP messages on stdin.
Diagnostics go to stderr; stdout carries only protocol messages.

For the browser interface, run:

```sh
npx --yes @modelcontextprotocol/inspector@2.11.0 ./bin/go-mcp-starter
```

## Test in Claude Desktop (macOS)

Manually tested with Claude Desktop on macOS on 2026-10-21.
Build the binary using steps 1 and 2 above, then configure Claude Desktop:

1. Open Claude Desktop's app settings, choose **Developer**, and click **Edit Config**.

   The configuration file is `~/Library/Application Support/Claude/claude_desktop_config.json`.

2. Add this server entry, keeping any existing entries in `mcpServers`.

   ```json
   {
     "mcpServers": {
       "go-mcp-starter": {
         "command": "/absolute/path/to/go-mcp-starter/bin/go-mcp-starter",
         "args": []
       }
     }
   }
   ```

   Replace `/absolute/path/to/go-mcp-starter` with the output of `pwd` from the repository root.

3. Save the file, fully quit Claude Desktop with `Cmd+Q`, and reopen it.

   Claude launches the compiled binary automatically and communicates with it over stdio.

4. Click **+ → Connectors** in the chat box and confirm that `go-mcp-starter` exposes the `add` tool.

5. Send this prompt:

   > Use the add tool from go-mcp-starter to add 12 and 30.

   Confirm an actual `add` tool call with `a: 12` and `b: 30`, returning `{"sum":42}`.
   An answer of 42 alone does not verify that Claude used the server.

For connection errors, check the server status and logs in Developer settings.
See the [official local-server setup guide](https://modelcontextprotocol.io/docs/2026-07-28/develop/connect-local-servers) for configuration and restart details.

## Tool contract

`add` performs read-only arithmetic with no filesystem or network access.
It accepts two required integer arguments and rejects extra properties.

| Item | Contract |
| --- | --- |
| Inputs | `a` and `b`, each between -9,007,199,254,740,991 and 9,007,199,254,740,991 |
| Output | `{"sum":42}`, with the same integer bounds as the inputs |
| Call rate | 10 validated calls per second, with a burst of 20 per process |
| Incoming message size | 64 KiB, including protocol metadata |
| Failures | Invalid arguments, out-of-range sums, and rate limits return tool errors with `isError: true` |

The JSON bounds preserve integer precision in clients that use JavaScript numbers.
Successful results include both `structuredContent` and a text block containing the same serialized JSON.
Unknown tools and malformed requests use the SDK's protocol error handling.

## Code boundaries

The project uses Hexagonal Architecture, also called Ports and Adapters.
The MCP adapter calls the `core.Adder` input port, which `core.Calculator` implements.

| Package | Owns |
| --- | --- |
| `cmd/go-mcp-starter` | Dependency wiring, process signals, and stdio startup |
| `internal/core` | The `Adder` port, signed 64-bit addition, cancellation, and overflow errors |
| `internal/adapter/mcp` | MCP schemas and metadata, JSON bounds, rate limiting, result translation, and tool logging |

The core imports only the Go standard library, and a test checks that boundary.
It supports the full signed 64-bit range; the adapter applies the narrower JSON contract.
The application has no external service dependencies requiring output ports.

## Run the checks

Select Go with GVM in each new shell, then run:

```sh
gofmt -w cmd internal
go test -race ./...
go vet ./...
go build -o bin/go-mcp-starter ./cmd/go-mcp-starter
```

Tests use named table-driven cases for arithmetic, adapter behavior, and subprocess stdio.
The suite covers schemas, invalid arguments, concurrency, rate limits, frame limits, clean shutdown, and both modern discovery and legacy initialization.
The [CI workflow](.github/workflows/ci.yml) checks formatting, verifies dependencies, runs vet and race tests, and builds the server on pushes and pull requests.
CI selects the Go version from `go.mod`.

## Compatibility and project decisions

The server identifies itself as `go-mcp-starter` version `0.1.0` and uses the official Go SDK v1.8.0.
The SDK supports MCP 2026-07-28 and earlier revisions, including 2025-11-25.
Application handlers honor cancellation, but this SDK release can still send a response after a stdio request is cancelled, contrary to the July 2026 rule.
The [upstream fix](https://github.com/modelcontextprotocol/go-sdk/commit/3ebc6d1eb7206d50590341f409e1499676db7345) is outside the pinned release, so this project does not claim complete conformance with that revision.

Read [DECISIONS.md](DECISIONS.md) before changing the architecture.
The [initial design](docs/adr/0001-initial-server-design.md) and [architecture decision](docs/adr/0002-hexagonal-architecture.md) explain the choices.
Shared coding-agent instructions live in [AGENTS.md](AGENTS.md); [CLAUDE.md](CLAUDE.md) imports them.
