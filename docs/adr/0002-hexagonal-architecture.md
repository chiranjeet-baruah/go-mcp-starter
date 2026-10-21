# Keep arithmetic independent of MCP

The repository follows Hexagonal Architecture at the user's request, using an input port and an MCP driving adapter.
This adds structure to a small addition example so the arithmetic can be tested and called independently of MCP.

The `core.Adder` interface belongs to the core and exposes `Add(context.Context, int64, int64) (int64, error)`.
The stateless `core.Calculator` implements this port, detects signed integer overflow, and honors an already-cancelled context.
The core depends only on Go's standard library and owns its arithmetic error.

The MCP adapter calls the input port rather than implementing arithmetic itself.
It owns JSON interoperability limits, typed MCP schemas, safety annotations, rate limiting, result formatting, error translation, and operational logging.
The command entry point constructs these dependencies and manages process signals and the stdio transport.

Keep the port and arithmetic implementation in one cohesive core package rather than adding separate domain and application packages for a single operation.
There are no output ports because the use case does not call an external dependency.
Introduce one when a real use case requires persistence or another external service.

## References

- [Alistair Cockburn's original Ports and Adapters description](https://alistair.cockburn.us/hexagonal-architecture/)
- [Go ports and adapters explanation supplied during planning](https://dev.to/buarki/hexagonal-architectureports-and-adapters-clarifying-key-concepts-using-go-14oo)
- [Go architecture example supplied during planning](https://alamrafiul.com/posts/go-hexagonal-architecture/)
