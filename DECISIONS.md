# Project Decision Log

> **AI Note:** Read this file before proposing architectural changes or structural refactors.
> **Human Note:** Keep entries reverse-chronological and write exactly one sentence per prose line.

### [004] Adopt Hexagonal Architecture | ACTIVE | 2026-10-21
* **Problem:** MCP transport details could become coupled to arithmetic logic.
* **Decision:** Put arithmetic behind a core-owned input port and call it from an MCP adapter wired by the command entry point.
* **Why:** This makes the application independently testable while keeping the first version to three production packages.
* **Context:** See [ADR-0002](docs/adr/0002-hexagonal-architecture.md).

### [003] Keep the released SDK despite its cancellation gap | ACTIVE | 2026-10-21
* **Problem:** The latest released SDK has a known stdio cancellation gap whose upstream fix is unreleased.
* **Decision:** Pin the official SDK to v1.8.0 and document the cancellation exception.
* **Why:** Published release reproducibility takes priority over adopting unreleased code for this learning project.
* **Context:** See [the SDK release policy](docs/adr/0001-initial-server-design.md#sdk-release-policy).

### [002] Use Go 1.27.3 and the official Go SDK | ACTIVE | 2026-10-21
* **Problem:** The server needs a reproducible Go toolchain and an MCP implementation without custom protocol code.
* **Decision:** Select Go 1.27.3 through GVM and use the official MCP Go SDK with pinned dependencies.
* **Why:** This follows the chosen language and delegates protocol compatibility and schema handling to the SDK.
* **Context:** See [the toolchain decision](docs/adr/0001-initial-server-design.md#toolchain).

### [001] Start with one local addition tool | ACTIVE | 2026-10-21
* **Problem:** Learning MCP is harder when the first server also requires external APIs, storage, or authentication.
* **Decision:** Expose only an add tool through local stdio.
* **Why:** One deterministic operation demonstrates discovery, validation, execution, and results with little application logic.
* **Context:** See [the local scope](docs/adr/0001-initial-server-design.md#local-scope).
