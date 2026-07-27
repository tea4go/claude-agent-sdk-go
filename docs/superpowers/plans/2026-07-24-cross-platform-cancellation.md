# Cross-Platform Cancellation Implementation Plan

> Execute this plan directly in the local SDK checkout. Do not perform Git
> operations. Use strict red-green-refactor cycles and keep Go 1.18
> compatibility.

**Goal:** Make logical interruption use the Claude control protocol, make
context cancellation stop the full owned process tree promptly on Unix and
Windows, reduce normal shutdown to one bounded grace period, and guarantee that
SDK-created Windows commands never display a console window.

**Architecture:** Separate logical turn interruption from process teardown.
`Interrupt` delegates to `control.Protocol`. A transport-owned process context
prevents Go's root-only `CommandContext` cancellation. A platform controller
owns a Unix process group or Windows Job Object, while a caller-context watcher
initiates immediate tree force termination. `Close` is a concurrent,
idempotent state transition that terminates the tree before waiting for pipe
readers.

**Tech stack:** Go 1.18, `os/exec`, Unix process groups,
`golang.org/x/sys/windows` Job Objects, existing SDK control protocol and test
helpers.

---

## Task 1: Route Interrupt Through the Control Protocol

**Files:**

- Modify: `internal/subprocess/transport.go`
- Modify: `internal/subprocess/transport_test.go`
- Modify: `internal/subprocess/config_test.go`
- Modify: `internal/subprocess/process_test.go`

### Step 1: Write failing streaming interrupt tests

Add tests using `newTransportMockCLIWithControlProtocol` that:

- connect a streaming transport;
- call `Interrupt` on every platform;
- assert no error and that `IsConnected` remains true;
- prove the mock receives and acknowledges a control request.

Add a one-shot test expecting an explicit “not available in one-shot mode”
error. Remove Unix-signal assumptions and Windows skips from interrupt tests.

### Step 2: Run focused tests and verify RED

Run:

```bash
go test ./internal/subprocess -run 'Interrupt|EnvironmentSetup' -count=1
```

Expected failure: Windows remains unsupported conceptually and the existing
mock without control responses causes interrupt behavior to differ from the new
expectations.

### Step 3: Implement protocol interrupt

Change `Transport.Interrupt` to:

- validate connected and streaming state under `RLock`;
- snapshot `protocol` and lifecycle cancellation state;
- release the lock;
- call `protocol.Interrupt(ctx)`;
- normalize the already-cancelled lifecycle case as idempotent success.

Remove `runtime.GOOS`, `os.Interrupt`, and the Windows-only error branch from
this method.

### Step 4: Run focused tests and verify GREEN

Run the focused command again and confirm all interrupt tests pass.

## Task 2: Add Transport-Owned Process Lifetime

**Files:**

- Modify: `internal/subprocess/transport.go`
- Modify: `internal/subprocess/process.go`
- Modify: `internal/subprocess/process_test.go`

### Step 1: Write failing context-cancellation lifecycle tests

Add a test helper script that remains alive and, on Unix, starts a child and
grandchild that inherit stdout. Record all PIDs in a temporary file.

Add tests that:

- connect using a cancellable context;
- wait for the helper readiness marker;
- cancel the context without calling `Close`;
- assert the root and ordinary descendants stop promptly;
- finally call `Close` to verify cleanup remains safe.

Add a test proving a setup failure after process start rolls the process back.

### Step 2: Run focused tests and verify RED

Run:

```bash
go test ./internal/subprocess -run 'ContextCancellation|ConnectRollback' -count=1
```

Expected failure: current `exec.CommandContext` kills only the root and no
process-tree watcher exists.

### Step 3: Introduce independent lifecycle contexts

In `Transport`, add:

- caller context reference or done channel;
- SDK-owned command context and cancel;
- SDK-owned I/O context and cancel;
- close/watcher completion signals;
- cancellation-requested state protected by the transport mutex.

Create the Claude command with the SDK-owned command context. Use the caller
context for connect-time validation and protocol initialization, then start a
watcher after successful process start.

The watcher must select between caller cancellation and normal close:

- caller cancellation marks the transport as cancelled;
- immediately invokes tree force termination;
- activates command fallback cancellation;
- stops protocol and I/O work;
- normal close completion exits the watcher without force-killing.

Connection rollback after `Start` must use the same force-cleanup path.

### Step 4: Run focused tests and verify GREEN

Run the focused command and verify prompt root cleanup. On Unix, verify recorded
descendant PIDs no longer exist.

## Task 3: Implement Unix Process-Group Ownership

**Files:**

- Add: `internal/subprocess/process_unix.go`
- Add: `internal/subprocess/process_unix_test.go`
- Refactor: `internal/subprocess/process.go`
- Modify: `internal/subprocess/transport.go`

### Step 1: Write failing process-group tests

Test that:

- command setup enables `SysProcAttr.Setpgid`;
- graceful termination sends to the process group;
- a child ignoring SIGTERM is force-killed after the grace timeout;
- a descendant holding stdout cannot create a second wait interval;
- already-finished groups are treated as success.

Use injectable internal grace duration for tests, leaving the production
default at five seconds.

### Step 2: Run Unix focused tests and verify RED

Run:

```bash
go test ./internal/subprocess -run 'ProcessGroup|SingleGracePeriod' -count=1
```

### Step 3: Implement the Unix controller

Create build-tagged Unix helpers that:

- configure a dedicated process group before `Start`;
- record the positive root PID as the process-group ID;
- send `SIGTERM` or `SIGKILL` to `-pgid`;
- poll group existence with signal 0 until exit or timeout;
- normalize `ESRCH` as already stopped.

Refactor common process code to call platform methods without referencing Unix
signals in Windows builds.

### Step 4: Run focused tests and verify GREEN

Confirm process group and timing tests pass without leaked helpers.

## Task 4: Implement Windows Job Object Ownership

**Files:**

- Add: `internal/subprocess/process_windows.go`
- Add: `internal/subprocess/process_windows_test.go`
- Modify: `go.mod`
- Generate: `go.sum`

### Step 1: Add Windows controller contract tests

Use a small internal operations interface so unit tests can simulate:

- Job Object creation and assignment;
- `KILL_ON_JOB_CLOSE` configuration;
- force termination and handle closure;
- assignment failure selecting the taskkill fallback;
- duplicate force/close calls.

Tests must not require a real GUI or child process.

### Step 2: Add the Windows dependency

Add a Go-1.18-compatible pinned version of:

```text
golang.org/x/sys/windows
```

### Step 3: Implement Job Object and fallback

The Windows controller will:

- create a Job Object;
- set `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`;
- open the root process with the rights needed for job assignment;
- assign the process immediately after `Start`;
- terminate the job for hard cancellation;
- close every opened handle exactly once.

If job setup or assignment fails, retain the root PID and force cleanup using:

```text
taskkill.exe /PID <pid> /T /F
```

Construct `taskkill.exe` only through
`internal/cli.NewExecCommandContext`.

### Step 4: Cross-compile focused packages

Run:

```bash
GOOS=windows go test -exec=true ./internal/subprocess ./internal/cli
```

Expected: all Windows files and tests compile successfully.

## Task 5: Refactor Close Into One Concurrent, Bounded Cleanup

**Files:**

- Modify: `internal/subprocess/transport.go`
- Modify: `internal/subprocess/process.go`
- Modify: `internal/subprocess/process_test.go`
- Modify: `client_test.go`

### Step 1: Write failing Close tests

Add tests for:

- two concurrent `Close` calls;
- repeated `Close`;
- `Interrupt` racing with `Close`;
- cooperative close;
- uncooperative close reaching force kill after one test grace interval;
- caller-cancelled close skipping the grace interval;
- pipe readers finishing only after descendant termination.

Assert all concurrent close callers see the same result and the transport ends
disconnected.

### Step 2: Run focused tests and verify RED

Run:

```bash
go test ./internal/subprocess -run 'Close|SingleGracePeriod' -count=1
```

Expected failure: current close holds the main lock, waits readers before
termination, and has no shared close result.

### Step 3: Implement close state

Add closing state, a completion channel, a cleanup mutex or once mechanism, and
a stored close error.

The cleanup owner:

1. snapshots resources and marks disconnected;
2. closes stdin;
3. if caller cancellation was not requested, gives the platform tree one
   graceful period;
4. force-kills any remaining tree;
5. cancels the command fallback and I/O contexts;
6. closes protocol and adapters;
7. closes remaining pipes, waits readers, and calls `cmd.Wait` exactly once;
8. releases platform handles and temporary files;
9. publishes the result.

Do not hold the main mutex during protocol calls, waits, or OS operations.

### Step 4: Run focused tests and race tests

Run:

```bash
go test ./internal/subprocess -run 'Close|Interrupt|Process' -count=1
go test -race ./internal/subprocess -run 'Close|Interrupt' -count=1
```

## Task 6: Enforce the Windows Hidden-Window Invariant

**Files:**

- Modify: `internal/cli/exec_command_windows.go`
- Add: `internal/cli/exec_command_windows_test.go`
- Modify: `internal/session/session.go`
- Modify: `internal/session/session_test.go`
- Add or modify: repository source-audit test

### Step 1: Write failing command-factory and audit tests

Test Windows command construction for:

- `HideWindow == true`;
- `CreationFlags` containing `CREATE_NO_WINDOW`.

Add a source audit that rejects direct `exec.Command` and
`exec.CommandContext` outside:

- `internal/cli/exec_command_windows.go`;
- `internal/cli/exec_command_unix.go`.

### Step 2: Run tests and verify RED

The audit should find direct calls in session worktree discovery and its test.

### Step 3: Centralize every command

Route session `git worktree list`, test `git init`, and Windows taskkill through
`cli.NewExecCommandContext`.

Keep both Windows hiding flags unconditional in the factory.

### Step 4: Re-run audit and platform tests

Run:

```bash
go test ./internal/cli ./internal/session ./internal/subprocess
GOOS=windows go test -exec=true ./internal/cli ./internal/session ./internal/subprocess
```

## Task 7: Documentation and Full Verification

**Files:**

- Modify: `docs/reference.md`
- Modify: `docs/parity.md`
- Modify: `internal/subprocess/CLAUDE.md` if lifecycle documentation is stale

### Step 1: Update behavior documentation

Document:

- protocol-based streaming interrupt;
- one-shot interrupt limitation;
- context cancellation as forceful process-tree cleanup;
- normal close as one five-second grace interval;
- Windows hidden-window and Job Object behavior.

### Step 2: Format changed Go files

Run `gofmt -s -w` on the explicit changed Go files.

### Step 3: Run complete verification

Run:

```bash
go test ./...
go test -race ./...
go vet ./...
GOOS=windows go test -exec=true ./...
```

Also audit:

```bash
rg -n '\bexec\.(Command|CommandContext)\b' .
```

Only the two platform command factory implementations may construct commands
directly.

### Step 4: Inspect local files without Git

Because the user explicitly prohibited Git operations, inspect changed files
using direct file reads, `rg`, test output, and formatting checks only. Report
the exact files changed and verification results for the user's own check.
