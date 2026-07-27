# Immediate Process Abort Design

## Goal

Add an explicit, cross-platform abort path that stops the Claude CLI process tree immediately, while preserving the existing graceful `Close`/`Disconnect` behavior needed for session persistence.

## Lifecycle semantics

- `Interrupt(ctx)` stops the current turn through the control protocol and keeps the streaming connection reusable.
- `Close()` and `Client.Disconnect()` close stdin, allow the CLI to persist session state during the existing grace period, and force-stop only after the grace period expires.
- `Abort()` immediately force-stops the owned process tree and then performs the same deterministic resource cleanup as `Close()`.
- Cancelling the context passed to `Connect` remains an immediate abort and shares the same force-stop primitive.

## Public API

`Client` gains `Abort() error`. `Transport` remains source-compatible for custom implementations: the SDK exposes an optional `AbortableTransport` interface containing `Abort() error`. `Client.Abort()` calls that interface when supported and falls back to `Transport.Close()` for legacy custom transports.

The built-in subprocess transport implements `Abort() error`. Calls are idempotent. If `Abort()` races with an in-progress graceful `Close()`, it immediately invokes `forceStop()` and then waits for the shared close operation to finish.

## Platform behavior

- Unix: force-stop the dedicated process group with `SIGKILL`, covering the CLI and ordinary descendants.
- Windows: terminate the Job Object. If Job Object attachment was unavailable, use the existing `taskkill.exe /T /F` fallback created through the SDK command helper, which sets `HideWindow` and `CREATE_NO_WINDOW`.

No new `exec.Command` call sites are introduced.

## Error and cleanup behavior

Abort treats an already-exited process as a successful cleanup outcome. It reports genuine force-stop or wait errors, closes protocols and pipes, waits for readers within the existing bound, reaps the child, releases Job Object/process-group resources, and deletes temporary SDK files.

## Tests

- Public client abort delegates to an abort-capable transport, disconnects client state, is repeatable, falls back to close for legacy transports, and wraps transport errors.
- Subprocess abort is fast for a process tree that ignores `SIGTERM`.
- Subprocess abort kills both the Unix process-group leader and its child.
- Abort racing with graceful close escalates immediately.
- Existing close grace-period, context-cancellation, idempotency, and Windows hidden-window tests continue to pass.

