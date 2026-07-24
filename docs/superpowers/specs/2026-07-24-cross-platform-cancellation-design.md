# Cross-Platform Cancellation and Process Cleanup Design

**Date:** 2026-07-24

## Summary

The SDK currently implements `Transport.Interrupt` by sending `os.Interrupt`
directly to the Claude CLI process. That behavior is unsupported on Windows and
does not match the Claude Agent SDK control protocol, which already exposes an
`interrupt` control request.

Process cleanup also waits for stdout and stderr goroutines before terminating
the process. A tool subprocess that inherits those pipes can therefore consume
one timeout while the readers remain blocked and a second timeout during
process termination. Cancellation through the context passed to `Connect`
additionally relies on `exec.CommandContext`, which only kills the direct
process and can leave descendants alive.

This change gives interrupt and shutdown distinct meanings:

- `Interrupt` stops the current Claude turn through the control protocol while
  keeping the streaming connection usable.
- Cancellation of the `Connect` context requests immediate termination of the
  entire owned process tree.
- `Close` performs bounded cleanup: normal closure gets one graceful period,
  followed by process-tree force termination when necessary.

The public API remains unchanged.

## Goals

- Make `Interrupt` work consistently on Windows, macOS, and Linux.
- Match the Claude Agent SDK control-protocol semantics for turn interruption.
- Ensure context cancellation promptly stops the Claude CLI and its ordinary
  child and grandchild processes.
- Prevent the current sequential reader-wait and process-wait timeouts.
- Make `Close` safe for duplicate and concurrent calls.
- Ensure every SDK-created Windows process is started without a visible console
  window.
- Retain Go 1.18 compatibility.

## Non-Goals

- Killing intentionally detached Unix daemons that create a new session or
  otherwise escape the SDK-owned process group.
- Adding new public cancellation or timeout options in this change.
- Changing application-level event ordering in WhaleTerm.
- Treating `Interrupt` as a connection teardown operation.

## Current Problems

### Interrupt uses an OS signal

`internal/subprocess.Transport.Interrupt` signals the root process with
`os.Interrupt`. Windows rejects this operation, while the control package
already implements the cross-platform `interrupt` request.

### Context cancellation only owns the root process

The Claude CLI is launched with `exec.CommandContext` using the caller's
`Connect` context. When that context is cancelled, the Go runtime kills the
root process. Descendants can remain alive and continue holding inherited
stdout or stderr handles.

### Close waits in the wrong order

`Close` cancels I/O and then waits up to five seconds for reader goroutines
before asking the process to terminate. If descendants still hold the pipes,
the readers cannot finish. Process termination can then consume another
timeout.

The termination function also selects on the transport context after `Close`
has already cancelled it, so the documented graceful SIGTERM period is skipped
unintentionally.

### Windows command creation is not fully centralized

The main Claude CLI and version checks use the Windows-aware command factory,
but session worktree discovery and at least one test invoke `exec.Command*`
directly. Those paths do not inherit `HideWindow` and `CREATE_NO_WINDOW`.

## Considered Approaches

### 1. Control-protocol interrupt only

Replace the OS signal with the existing protocol request and leave cleanup
unchanged.

This is the smallest change, but it does not fix descendant leaks, duplicate
timeouts, or context-cancellation behavior.

### 2. Root-process termination with reordered Close

Use the protocol for interrupt and terminate the root process before waiting
for I/O.

This removes the duplicate timeout for simple commands but still leaves tool
subprocesses alive. It does not meet the cross-platform process-tree
requirement.

### 3. Control protocol plus platform process-tree ownership

Use the control protocol for logical interruption and introduce platform
process-tree controllers for lifecycle cleanup.

- Unix commands run in a dedicated process group.
- Windows commands are assigned to a Job Object configured with
  `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`.
- Windows falls back to a hidden `taskkill.exe /T /F` invocation when Job
  Object assignment is unavailable.

This is the selected design. It cleanly separates turn interruption from
process ownership and addresses the observed shutdown delay.

## Detailed Design

### Interrupt semantics

`Transport.Interrupt(ctx)` will:

1. Validate that the transport is connected and is in streaming mode.
2. Snapshot the control protocol pointer while holding the transport read lock.
3. Release the transport lock before performing the control request.
4. Call `Protocol.Interrupt(ctx)`.

It will not send an OS signal and will not disconnect the transport. One-shot
transports do not have a control protocol and will return a clear unsupported
operation error.

If the transport is already stopping because its owning context was cancelled,
interrupt is idempotent: the requested stopped state has already been reached,
so cancellation-related protocol closure is not surfaced as a second failure.

### Owned process lifetime

The process command will use an SDK-owned context rather than the caller's
`Connect` context. This prevents the default `exec.CommandContext` cancellation
path from killing only the root process.

The transport will keep separate signals for:

- caller context cancellation;
- internal I/O and control-protocol lifetime;
- command fallback cancellation;
- explicit close completion.

After a successful process start, a watcher waits for either caller-context
cancellation or normal transport closure. Caller cancellation immediately:

1. marks cancellation as requested;
2. force-terminates the owned process tree;
3. cancels the command fallback context;
4. cancels control-protocol and I/O work.

Explicit normal `Close` does not trigger that watcher and therefore retains its
graceful period.

If any setup step fails after the process has started, connection rollback uses
the same process-tree force-termination path before releasing pipes and
temporary files. A partially initialized transport must not leak a process or
Job Object.

### Unix process tree

Before start, the Claude command receives `SysProcAttr.Setpgid = true`, placing
the root process and its normal descendants in a dedicated process group.

Lifecycle operations target the negative process-group ID:

- graceful close sends `SIGTERM`;
- forced close sends `SIGKILL`;
- already-exited and not-found conditions are treated as successful cleanup.

The SDK never targets its own process group. The process-group ID is recorded
only after a successful start.

### Windows process tree

All Windows commands retain:

```go
HideWindow:    true
CreationFlags: CREATE_NO_WINDOW
```

After the Claude process starts, the SDK creates and assigns a Job Object. The
job uses `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, so closing or explicitly
terminating the job cleans up all ordinary descendants.

The implementation uses `golang.org/x/sys/windows` rather than maintaining raw
Win32 syscall declarations.

If Job Object creation or assignment is not available, the SDK records a
fallback controller. Forced cleanup invokes:

```text
taskkill.exe /PID <pid> /T /F
```

through the same Windows-aware command factory, so the fallback cannot create
a console window. Because the root command uses an SDK-owned context, it
remains available for tree discovery until the fallback has run.

Windows has no general SIGTERM equivalent for a hidden non-console process.
Normal close first closes stdin and allows the process tree to exit naturally
during the graceful period; force cleanup then terminates the Job Object.

### Close ordering and bound

`Close` will use a dedicated close state and completion channel so duplicate or
concurrent callers share one cleanup operation and one result.

The cleanup owner will:

1. mark the transport as closing and reject new operations;
2. close stdin;
3. for normal closure, request platform-appropriate graceful tree termination
   and wait for at most five seconds total;
4. force-terminate the process tree if anything remains;
5. activate command-context fallback cancellation;
6. cancel control-protocol and I/O work;
7. close pipes as necessary, wait for reader goroutines, and reap the root
   process exactly once;
8. release the Job Object and temporary resources;
9. publish the close result to all waiting callers.

If caller-context cancellation was already requested, steps 2 and 3 do not
introduce another grace period: force termination begins immediately.

Process-tree cleanup happens before waiting for pipe readers. This ensures
descendants cannot keep inherited handles open throughout a separate timeout.
Normal `Close` therefore has one approximately five-second graceful bound, not
two sequential five-second waits. Context cancellation normally begins hard
termination immediately.

No blocking control request or process wait occurs while holding the main
transport mutex.

### Command creation and hidden-window invariant

Every repository use of `exec.Command` or `exec.CommandContext`, including test
helpers, will be routed through `internal/cli.NewExecCommandContext`, except
for the platform-specific implementation of that factory itself.

The following known paths are included:

- Claude CLI process creation;
- Claude CLI version discovery;
- session `git worktree list`;
- Windows `taskkill.exe` fallback;
- tests that run `git init` or helper processes.

A source audit test or equivalent verification will ensure no new direct
command construction bypasses the factory. Windows-specific tests will assert
both hidden-window flags.

## Concurrency and Error Handling

- `Interrupt` snapshots state under a read lock and performs protocol I/O after
  releasing it.
- `Close` is idempotent and concurrent callers wait on the same completion
  channel.
- Context cancellation and `Close` may race; platform controllers use
  synchronization so tree termination and handle release occur at most once.
- Expected already-exited, process-not-found, and closed-handle results are
  normalized as successful cleanup.
- A failed graceful signal does not prevent the force-kill fallback.
- Cleanup continues after individual close errors and returns a prioritized
  wrapped error that preserves the actionable cause without relying on
  post-Go-1.18 APIs.

## Testing Strategy

### Protocol behavior

- Verify transport interrupt emits a control request with subtype `interrupt`.
- Verify the transport remains connected after a successful interrupt.
- Verify Windows no longer has a special unsupported-interrupt path.
- Verify one-shot mode returns the documented error.

### Lifecycle behavior

- Spawn a helper that creates a child and grandchild and records their PIDs.
- Verify caller-context cancellation removes the complete owned tree promptly.
- Verify normal close handles a cooperative process without force termination.
- Verify an uncooperative process reaches force termination after one graceful
  timeout.
- Verify inherited stdout and stderr handles cannot cause a second timeout.
- Verify duplicate and concurrent `Close` calls return consistently.
- Exercise `Interrupt` racing with `Close` under the race detector.
- Verify setup failure after process start rolls back the complete process tree.

Unix CI runs real process-group tests. Windows-specific controller behavior is
covered by unit seams and native Windows CI when available.

### Windows build and visibility

- Assert `HideWindow` and `CREATE_NO_WINDOW` on commands produced by the Windows
  factory.
- Ensure `taskkill.exe` uses that factory.
- Cross-compile all packages and tests for Windows.
- Audit the source tree for direct `exec.Command*` uses outside the factory.

### Verification commands

```bash
go test ./...
go test -race ./...
GOOS=windows go test -exec=true ./...
gofmt -s -w <changed-go-files>
go vet ./...
```

## Compatibility

No exported method signatures change.

Behavior changes are intentional:

- Windows streaming interrupt becomes supported.
- Streaming interrupt becomes a protocol operation on every platform.
- One-shot interrupt reports that the operation is unavailable instead of
  sending a Unix-only signal.
- Cancelling the `Connect` context force-cleans the owned process tree instead
  of relying on root-only `exec.CommandContext` cancellation.
- Normal close is bounded by one graceful interval before force cleanup.

The added `golang.org/x/sys/windows` dependency is used only by Windows-tagged
files.

## Operational Expectations

For an ordinary complex tool script:

- the control-protocol interrupt acknowledgement should return quickly while
  the connection stays alive;
- cancelling the owning context initiates tree force termination immediately;
- normal disconnect may wait up to the single five-second graceful period;
- force cleanup cannot terminate deliberately detached Unix daemons that have
  escaped the SDK-owned process group.
