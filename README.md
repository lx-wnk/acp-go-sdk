<a href="https://agentclientprotocol.com/" >
  <img alt="Agent Client Protocol" src="https://zed.dev/img/acp/banner-dark.webp">
</a>

# ACP Go SDK

[![CI](https://github.com/lx-wnk/acp-go-sdk/actions/workflows/ci.yaml/badge.svg)](https://github.com/lx-wnk/acp-go-sdk/actions/workflows/ci.yaml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](./LICENSE)

> **A maintained fork of [coder/acp-go-sdk](https://github.com/coder/acp-go-sdk).**
> Upstream has had no release or maintainer response since June 2026, while the ACP
> schema kept moving; this fork tracks it. The Go module path stays
> `github.com/coder/acp-go-sdk` on purpose, so every fix here remains a patch that can
> be offered back upstream — which also means you install it with a `replace`
> directive rather than a plain `go get`. See [Installation](#installation).

Go library for the Agent Client Protocol (ACP) - a standardized communication protocol
between code editors and AI‑powered coding agents.

Learn more about the protocol itself at <https://agentclientprotocol.com>.

## Installation

Because the module keeps upstream's path, `go get github.com/coder/acp-go-sdk@v1.21.0`
resolves to Coder's repository, which has no such version — the module proxy answers
404\. Point the path at this fork instead:

<!-- `$ printf 'go mod edit -replace github.com/coder/acp-go-sdk=github.com/lx-wnk/acp-go-sdk@v%s\ngo get github.com/coder/acp-go-sdk\ngo mod tidy\n' "$(cat schema/version)"` as bash -->

```bash
go mod edit -replace github.com/coder/acp-go-sdk=github.com/lx-wnk/acp-go-sdk@v1.21.0
go get github.com/coder/acp-go-sdk
go mod tidy
```

Your `require` line will name upstream's version while `replace` supplies this fork's
code; that is expected. Imports stay `github.com/coder/acp-go-sdk`, so switching back
to upstream later is a one-line deletion.

### Versioning

This module's version tracks the ACP schema release it is generated from, not Go API
compatibility: `vX.Y.Z` is generated from schema `X.Y.Z`.

A minor bump can therefore remove exported types or add methods to the `Client` and
`Agent` interfaces, which breaks compilation for code that implements them. Schema
1.21.0, for example, graduates elicitation out of the experimental surface: 18
`Unstable*` types are renamed and `Client` gains two required methods.

Pin the exact version shown above rather than `@latest` or `@v1`, and upgrade
deliberately, reading the release notes for the schema version you are moving to.

The wire protocol is more stable than the Go API. A schema bump that breaks
compilation usually leaves JSON-RPC interoperability with older peers intact.

## Get Started

### Understand the Protocol

Start by reading the [official ACP documentation](https://agentclientprotocol.com)
to understand the core concepts and protocol specification.

### Try the Examples

The [examples directory](./example)
contains simple implementations of both Agents and Clients in Go.
You can run them from your terminal or connect to external ACP agents.

- `go run ./example/agent` starts a minimal ACP agent over stdio.
- `go run ./example/claude-code` demonstrates bridging to Claude Code.
- `go run ./example/client` connects to a running agent and streams a sample turn.
- `go run ./example/gemini` bridges to the Gemini CLI in ACP mode (flags: -model, -sandbox, -debug, -gemini /path/to/gemini).

You can watch the interaction by running `go run ./example/client` locally.

### Explore the API

pkg.go.dev indexes by module path, and this fork shares upstream's — so the page below
shows upstream's v0.13.5 and none of the API this README documents. Until upstream
catches up, use `go doc` against a local checkout, or read `CHANGELOG.md`.

- <https://pkg.go.dev/github.com/coder/acp-go-sdk> (upstream v0.13.5, not this fork)

If you're building an [Agent](https://agentclientprotocol.com/protocol/overview#agent):

- Implement the `acp.Agent` interface (and optionally `acp.AgentLoader` for `session/load`).
- Create a connection with `acp.NewAgentSideConnection(agent, os.Stdout, os.Stdin)`.
- Send updates and make client requests using the returned connection.

If you're building a [Client](https://agentclientprotocol.com/protocol/overview#client):

- Implement the `acp.Client` interface (and optionally `acp.ClientTerminal` for
  terminal features).
- Launch or connect to your Agent process (stdio), then create a connection with
  `acp.NewClientSideConnection(client, stdin, stdout)`.
- Call `Initialize`, `NewSession`, and `Prompt` to run a turn and stream updates.

Helper constructors are provided to reduce boilerplate when working with union types:

- Content blocks: `acp.TextBlock`, `acp.ImageBlock`, `acp.AudioBlock`,
  `acp.ResourceLinkBlock`, `acp.ResourceBlock`.
- Tool content: `acp.ToolContent`, `acp.ToolDiffContent`, `acp.ToolTerminalRef`.
- Utility: `acp.Ptr[T]` for pointer fields in request/update structs.

### Elicitation

Since schema 1.21.0, `Client` requires `CreateElicitation` and `CompleteElicitation`.
An agent uses them to ask the user for structured input, either by having the client
render a form or by directing the user to a URL.

A client that cannot present either should report the method as unavailable rather
than answering on the user's behalf:

```go
func (c *myClient) CreateElicitation(ctx context.Context, params acp.CreateElicitationRequest) (acp.CreateElicitationResponse, error) {
	return acp.CreateElicitationResponse{}, acp.NewMethodNotFound(acp.ClientMethodElicitationCreate)
}
```

`Decline` means the user declined. Returning it when nobody was asked tells the agent
a decision was made that never was.

The URL arm is chosen by the agent, so treat it as untrusted input. `SafeURL` parses
it and rejects every scheme except `https` and `http`:

```go
if params.Url == nil {
	return acp.CreateElicitationResponse{}, acp.NewMethodNotFound(acp.ClientMethodElicitationCreate)
}
target, err := params.Url.SafeURL()
if err != nil {
	return acp.CreateElicitationResponse{}, err
}
// Show target.Host to the user before navigating.
```

`SafeURL` also returns a host with non-ASCII labels in its ASCII (Punycode) form, so
what you display cannot be confused with a different name: a URL whose host reads as
`apple.com` but begins with a Cyrillic `а` comes back as `xn--pple-43d.com`.

A successful parse is not permission to open the URL. Show the origin to the user
first, and do not prefetch the URL to build a preview.

### Extension methods

ACP supports **extension methods** for custom JSON-RPC methods whose names start with `_`.
Use them to add functionality without conflicting with future ACP versions.

#### Handling inbound extension methods

Implement `acp.ExtensionMethodHandler` on your Agent or Client. Your handler will be
invoked for any incoming method starting with `_`.

```go
// HandleExtensionMethod handles ACP extension methods (names starting with "_").
func (a MyAgent) HandleExtensionMethod(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "_example.com/hello":
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return map[string]any{"greeting": "hello " + p.Name}, nil
	default:
		return nil, acp.NewMethodNotFound(method)
	}
}
```

> Note: Per the ACP spec, unknown extension notifications should be ignored.
> This SDK suppresses noisy logs for unhandled extension notifications that return
> “Method not found”.

#### Calling extension methods

From either side, use `CallExtension` / `NotifyExtension` on the connection.

```go
raw, err := conn.CallExtension(ctx, "_example.com/hello", map[string]any{"name": "world"})
if err != nil {
	return err
}

var resp struct {
	Greeting string `json:"greeting"`
}
if err := json.Unmarshal(raw, &resp); err != nil {
	return err
}

if err := conn.NotifyExtension(ctx, "_example.com/progress", map[string]any{"pct": 50}); err != nil {
	return err
}
```

#### Advertising extension support via `_meta`

ACP uses the `_meta` field inside capability objects as the negotiation/advertising
surface for extensions.

- Client -> Agent: `InitializeRequest.ClientCapabilities.Meta`
- Agent -> Client: `InitializeResponse.AgentCapabilities.Meta`

Keys `traceparent`, `tracestate`, and `baggage` are reserved in `_meta` for W3C trace
context/OpenTelemetry compatibility.

### Study a Production Implementation

For a complete, production‑ready integration, see the
[Gemini CLI Agent](https://github.com/google-gemini/gemini-cli) which exposes an
ACP interface. The Go example client `example/gemini` demonstrates connecting
to it via stdio.

## Resources

- [Go package docs](https://pkg.go.dev/github.com/coder/acp-go-sdk) (upstream v0.13.5, not this fork)
- [Examples (Go)](./example)
- [Protocol Documentation](https://agentclientprotocol.com)
- [Agent Client Protocol GitHub Repository](https://github.com/agentclientprotocol/agent-client-protocol)

## Contributing

Fixes are offered upstream wherever they apply — see [CONTRIBUTING.md](./CONTRIBUTING.md)
for that policy, the verify commands, and why `*_gen.go` is never hand-edited.
Vulnerabilities go to [SECURITY.md](./SECURITY.md), not to a public issue.

## License

Apache 2.0. See [LICENSE](./LICENSE). This distribution is a modified fork; see
[NOTICE](./NOTICE).
