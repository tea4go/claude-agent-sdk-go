# Immediate Process Abort Implementation Plan

> **For agentic workers:** Execute inline in this session. Do not perform Git operations. Follow the red-green-refactor cycle for every behavior change.

**Goal:** Add an explicit immediate abort API that force-stops the full Claude CLI process tree without changing graceful close semantics.

**Architecture:** Reuse the existing platform `processTree.forceStop()` implementations. Add an optional public transport capability so existing custom transports remain compatible, then route `Client.Abort()` and built-in subprocess `Abort()` through a shared cleanup path. A concurrent abort must escalate an already-running graceful close.

**Tech Stack:** Go 1.18+, `os/exec`, Unix process groups, Windows Job Objects, Go testing.

---

### Task 1: Public client abort contract

**Files:**
- Modify: `types.go`
- Modify: `client.go`
- Test: `client_test.go`

- [ ] Add failing tests proving `Client.Abort()` uses an abort-capable transport, clears client state, remains idempotent, falls back to `Close()` for a legacy transport, and returns a wrapped abort error.
- [ ] Run `go test . -run 'TestClientAbort' -count=1` and confirm failure because the API does not exist.
- [ ] Add `Abort()` to `Client`, add optional `AbortableTransport`, and implement the minimal client delegation and state cleanup.
- [ ] Re-run the focused tests and confirm they pass.

### Task 2: Built-in subprocess immediate abort

**Files:**
- Modify: `internal/subprocess/transport.go`
- Modify: `internal/subprocess/process.go`
- Test: `internal/subprocess/process_unix_test.go`
- Test: `internal/subprocess/transport_test.go`

- [ ] Add a failing Unix integration test using a parent and child that ignore `SIGTERM`; assert `Abort()` returns well below the graceful timeout and both processes disappear.
- [ ] Add a failing concurrency test proving `Abort()` escalates an in-progress `Close()` instead of waiting for the complete grace period.
- [ ] Run the focused subprocess tests and confirm failure because `Abort()` does not exist.
- [ ] Refactor transport shutdown into graceful and forced entry points while retaining one idempotent cleanup owner.
- [ ] Re-run focused tests and existing lifecycle/context-cancellation tests.

### Task 3: Documentation and full verification

**Files:**
- Modify: `docs/reference.md`
- Modify: `docs/architecture/interfaces.md`

- [ ] Document the distinction among `Interrupt`, `Disconnect`, and `Abort`, including legacy custom-transport fallback.
- [ ] Run `gofmt -s` on changed Go files.
- [ ] Run `go test ./... -count=1`.
- [ ] Run `go test -race ./... -count=1`.
- [ ] Run `go vet ./...`.
- [ ] Cross-compile tests for Windows amd64 and arm64 to verify platform-specific code and hidden-window helpers compile.

