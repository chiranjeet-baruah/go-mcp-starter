# Repository instructions

Read `README.md` for setup and `DECISIONS.md` before proposing architectural changes or structural refactors.
Keep instructions here as the shared source; `CLAUDE.md` imports this file.

## Purpose and toolchain

- This is a local MCP learning server with one `add` tool.
- Use Go 1.27.3 and keep dependencies pinned in `go.mod` and `go.sum`.
- Activate Go in each fresh shell with `source "$HOME/.gvm/scripts/gvm"` followed by `gvm use go1.27.3`.
- Use the released MCP SDK v1.8.0 and preserve its documented cancellation exception until a deliberate upgrade.

## Architecture

- Keep `cmd/go-mcp-starter` responsible for dependency wiring and process lifecycle.
- Keep `internal/core` responsible for the `Adder` input port, arithmetic, cancellation, and arithmetic errors.
- The production core may import only the Go standard library.
- Keep MCP types, JSON schemas, interoperability bounds, rate limiting, logging, and error translation in `internal/adapter/mcp`.
- Inject the core-owned port into the adapter; add output ports only for actual external dependencies.

## MCP behavior

- Reserve stdout for MCP messages and write diagnostics to stderr without raw arguments or results.
- Validate arguments and successful output against explicit schemas and keep structured results consistent with JSON text content.
- Keep safety annotations accurate and advertise only implemented capabilities.
- Preserve the JSON integer bounds, frame limit, and rate limit documented in the README.
- Honor cancellation and EOF shutdown; do not claim complete July 2026 conformance while the SDK exception remains.

## Testing and verification

- Always use named table-driven test cases with `t.Run` across the repository.
- Test core behavior independently, adapter translation through the input port, and actual subprocess stdio at the command boundary.
- Use small handwritten test doubles instead of a mocking framework.
- Cover invalid inputs, boundaries, concurrency, and shutdown where relevant; do not weaken tests to make a change pass.
- Format with `gofmt -w cmd internal`, then run `go test -race ./...`, `go vet ./...`, and `go build -o bin/go-mcp-starter ./cmd/go-mcp-starter`.
- Keep `.github/workflows/ci.yml` aligned with these checks; CI selects Go from `go.mod` using a commit-pinned `actions/setup-go` action.

## Decision records and communication

- Record accepted architectural choices in `DECISIONS.md` with a link to a concise ADR when context is needed.
- Preserve stable IDs, reverse-chronological order, and exactly one sentence per prose line.
- Use only ACTIVE, REJECTED, SUPERSEDED, or EXPERIMENTAL statuses in decision headers.
- Prefer the smallest correct change; separate verified facts from assumptions and report checks actually run.
- Keep README instructions short, numbered where sequential, and consistent with executable commands.
