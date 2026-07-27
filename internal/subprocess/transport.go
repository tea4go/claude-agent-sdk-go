// Package subprocess provides the subprocess transport implementation for Claude Code CLI.
package subprocess

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tea4go/claude-agent-sdk-go/internal/cli"
	"github.com/tea4go/claude-agent-sdk-go/internal/control"
	"github.com/tea4go/claude-agent-sdk-go/internal/parser"
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

const (
	// channelBufferSize is the buffer size for message and error channels.
	channelBufferSize = 10
	// ioShutdownTimeout bounds readers after the process tree and pipes stop.
	ioShutdownTimeout = time.Second
	// windowsOS is the GOOS value for Windows platform.
	windowsOS = "windows"
)

// Transport implements the Transport interface using subprocess communication.
type Transport struct {
	// Process management
	cmd        *exec.Cmd
	cliPath    string
	options    *shared.Options
	closeStdin bool
	promptArg  *string // For one-shot queries, prompt passed as CLI argument
	entrypoint string  // CLAUDE_CODE_ENTRYPOINT value (sdk-go or sdk-go-client)

	// Connection state
	connected bool
	closing   bool
	closeDone chan struct{}
	closeErr  error
	mu        sync.RWMutex

	// I/O streams
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     *os.File      // Temporary file for stderr isolation
	stderrPipe io.ReadCloser // Pipe for callback-based stderr handling

	// Temporary files (cleaned up on Close)
	mcpConfigFile     *os.File // Temporary MCP config file
	skillRegistryDirs []string // Temporary Skill registry plugin wrapper dirs

	// Message parsing
	parser *parser.Parser

	// Stream validation
	validator *shared.StreamValidator

	// Channels for communication
	msgChan chan shared.Message
	errChan chan error

	// Control protocol (for streaming mode only)
	protocol        *control.Protocol
	protocolAdapter *ProtocolAdapter

	// Slash commands discovered from the system/init message.
	slashCommands          []control.SlashCommand
	slashCommandsReady     chan struct{}
	slashCommandsReadyOnce sync.Once

	// Control and cleanup
	ctx                   context.Context
	cancel                context.CancelFunc
	processCancel         context.CancelFunc
	processTree           processTree
	cancellationRequested uint32
	watcherStop           chan struct{}
	watcherDone           chan struct{}
	processWaitOnce       sync.Once
	processWaitDone       chan struct{}
	processWaitErr        error
	wg                    sync.WaitGroup
}

// New creates a new subprocess transport.
func New(cliPath string, options *shared.Options, closeStdin bool, entrypoint string) *Transport {
	return &Transport{
		cliPath:    cliPath,
		options:    options,
		closeStdin: closeStdin,
		entrypoint: entrypoint,
		parser:     newParser(options),
		validator:  shared.NewStreamValidator(),
	}
}

// NewWithPrompt creates a new subprocess transport for one-shot queries with prompt as CLI argument.
func NewWithPrompt(cliPath string, options *shared.Options, prompt string) *Transport {
	return &Transport{
		cliPath:    cliPath,
		options:    options,
		closeStdin: true,
		entrypoint: "sdk-go", // Query mode uses sdk-go
		parser:     newParser(options),
		validator:  shared.NewStreamValidator(),
		promptArg:  &prompt,
	}
}

// newParser creates a parser using the buffer size from options, or the default.
func newParser(options *shared.Options) *parser.Parser {
	if options != nil && options.MaxBufferSize != nil {
		return parser.NewWithSize(*options.MaxBufferSize)
	}
	return parser.New()
}

// IsConnected returns whether the transport is currently connected.
func (t *Transport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected && t.cmd != nil && t.cmd.Process != nil
}

// Connect starts the Claude CLI subprocess.
func (t *Transport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	if t.closing {
		return fmt.Errorf("transport is closing")
	}
	if t.connected {
		return fmt.Errorf("transport already connected")
	}
	t.closeDone = make(chan struct{})
	t.closeErr = nil

	// Generate temporary plugin wrappers and MCP config files if requested.
	opts, err := t.prepareRuntimeOptions()
	if err != nil {
		return err
	}
	t.options = opts

	// Build command with all options
	var args []string
	if t.promptArg != nil {
		// One-shot query with prompt as CLI argument
		args = cli.BuildCommandWithPrompt(t.cliPath, opts, *t.promptArg)
	} else {
		// Streaming mode or regular one-shot
		args = cli.BuildCommand(t.cliPath, opts, t.closeStdin)
	}
	processCtx, processCancel := context.WithCancel(context.Background())
	t.processCancel = processCancel
	t.processWaitOnce = sync.Once{}
	t.processWaitDone = make(chan struct{})
	t.processWaitErr = nil
	t.cmd = cli.NewExecCommandContext(processCtx, args)
	configureProcessTree(t.cmd)

	// Set up environment and apply to command
	t.cmd.Env = t.buildEnvironment()

	// Set working directory if specified
	if t.options != nil && t.options.Cwd != nil {
		if err := cli.ValidateWorkingDirectory(*t.options.Cwd); err != nil {
			processCancel()
			t.cleanup()
			return err
		}
		t.cmd.Dir = *t.options.Cwd
	}

	// Check CLI version and warn if outdated (non-blocking)
	t.emitCLIVersionWarning(ctx)

	// Set up I/O pipes
	if err := t.setupIoPipes(); err != nil {
		processCancel()
		t.cleanup()
		return err
	}

	// Start the process
	if err := t.cmd.Start(); err != nil {
		processCancel()
		t.cleanup()
		return shared.NewConnectionError(
			fmt.Sprintf("failed to start Claude CLI: %v", err),
			err,
		)
	}

	tree, err := attachProcessTree(t.cmd)
	if err != nil {
		_ = t.cmd.Process.Kill()
		processCancel()
		_ = t.cmd.Wait()
		t.cleanup()
		return shared.NewConnectionError(
			fmt.Sprintf("failed to own Claude CLI process tree: %v", err),
			err,
		)
	}
	t.processTree = tree

	// Set up context for goroutine management
	t.ctx, t.cancel = context.WithCancel(ctx)
	atomic.StoreUint32(&t.cancellationRequested, 0)

	// Initialize channels
	t.msgChan = make(chan shared.Message, channelBufferSize)
	t.errChan = make(chan error, channelBufferSize)
	t.slashCommands = nil
	t.slashCommandsReady = make(chan struct{})
	t.slashCommandsReadyOnce = sync.Once{}

	// Start I/O handling goroutines
	t.wg.Add(1)
	go t.handleStdout()

	// Start stderr callback goroutine if callback is configured
	if t.stderrPipe != nil && t.options != nil && t.options.StderrCallback != nil {
		t.wg.Add(1)
		go t.handleStderrCallback()
	}

	// Note: Do NOT close stdin here for one-shot mode
	// The CLI still needs stdin to receive the message, even with --print flag
	// stdin will be closed after sending the message in SendMessage()

	// Set up control protocol for streaming mode only
	if err := t.setupControlProtocol(t.ctx); err != nil {
		atomic.StoreUint32(&t.cancellationRequested, 1)
		_ = t.terminateProcess()
		if t.cancel != nil {
			t.cancel()
		}
		if t.protocol != nil {
			_ = t.protocol.Close()
			t.protocol = nil
		}
		if t.protocolAdapter != nil {
			_ = t.protocolAdapter.Close()
			t.protocolAdapter = nil
		}
		t.wg.Wait()
		t.cleanup()
		t.cancel = nil
		t.ctx = nil
		return err
	}

	t.watcherStop = make(chan struct{})
	t.watcherDone = make(chan struct{})
	go t.watchCallerCancellation(
		ctx.Done(),
		t.watcherStop,
		t.watcherDone,
		t.processTree,
		t.cmd,
		t.processCancel,
		t.cancel,
	)

	t.connected = true
	return nil
}

// setupControlProtocol initializes control protocol for streaming mode.
// Returns nil immediately for one-shot mode (closeStdin == true).
func (t *Transport) setupControlProtocol(ctx context.Context) error {
	if t.closeStdin {
		return nil // One-shot mode doesn't need control protocol
	}

	t.protocolAdapter = NewProtocolAdapter(t.stdin)
	t.protocol = control.NewProtocol(t.protocolAdapter, t.buildProtocolOptions()...)

	if err := t.protocol.Start(ctx); err != nil {
		return fmt.Errorf("failed to start control protocol: %w", err)
	}

	// Perform handshake when hooks, permissions, checkpointing, or SDK MCP servers configured
	if t.needsProtocolHandshake() {
		if _, err := t.protocol.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize control protocol: %w", err)
		}
	}

	return nil
}

// needsProtocolHandshake returns true if control protocol handshake is required.
func (t *Transport) needsProtocolHandshake() bool {
	if t.options == nil {
		return false
	}
	return t.options.Hooks != nil ||
		t.options.CanUseTool != nil ||
		t.options.EnableFileCheckpointing ||
		len(t.options.Plugins) > 0 ||
		isSkillsList(t.options.Skills) ||
		t.hasSdkMcpServers()
}

// SendMessage sends a message to the CLI subprocess.
func (t *Transport) SendMessage(ctx context.Context, message shared.StreamMessage) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// For one-shot queries with promptArg, the prompt is already passed as CLI argument
	// so we don't need to send any messages via stdin
	if t.promptArg != nil {
		return nil // No-op for one-shot queries
	}

	if !t.connected || t.stdin == nil {
		return fmt.Errorf("transport not connected or stdin closed")
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Serialize message to JSON
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send with newline
	_, err = t.stdin.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	// For one-shot mode, close stdin after sending the message
	if t.closeStdin {
		_ = t.stdin.Close()
		t.stdin = nil
	}

	return nil
}

// ReceiveMessages returns channels for receiving messages and errors.
func (t *Transport) ReceiveMessages(_ context.Context) (<-chan shared.Message, <-chan error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		// Return closed channels if not connected
		msgChan := make(chan shared.Message)
		errChan := make(chan error)
		close(msgChan)
		close(errChan)
		return msgChan, errChan
	}

	return t.msgChan, t.errChan
}

// Interrupt stops the current turn through the streaming control protocol.
// It does not terminate the CLI process or disconnect the transport.
func (t *Transport) Interrupt(ctx context.Context) error {
	t.mu.RLock()
	if !t.connected || t.cmd == nil || t.cmd.Process == nil {
		t.mu.RUnlock()
		return fmt.Errorf("process not running")
	}
	if t.closeStdin {
		t.mu.RUnlock()
		return fmt.Errorf("interrupt not available in one-shot mode")
	}
	protocol := t.protocol
	transportCtx := t.ctx
	t.mu.RUnlock()

	if transportCtx != nil && transportCtx.Err() != nil {
		return nil
	}
	if protocol == nil {
		return fmt.Errorf("control protocol not initialized")
	}
	err := protocol.Interrupt(ctx)
	if err != nil && atomic.LoadUint32(&t.cancellationRequested) != 0 {
		return nil
	}
	return err
}

// Abort immediately force-stops the subprocess tree and performs the same
// deterministic resource cleanup as Close. It is safe to call while a
// graceful Close is already in progress; the in-progress close is escalated.
func (t *Transport) Abort() error {
	atomic.StoreUint32(&t.cancellationRequested, 1)

	t.mu.RLock()
	closing := t.closing
	tree := t.processTree
	done := t.closeDone
	t.mu.RUnlock()

	if !closing {
		return t.Close()
	}

	var forceErr error
	if tree != nil {
		if err := tree.forceStop(); err != nil && !isProcessAlreadyFinishedError(err) {
			forceErr = fmt.Errorf("force process tree: %w", err)
		}
	}
	if done != nil {
		<-done
	}

	t.mu.RLock()
	closeErr := t.closeErr
	t.mu.RUnlock()
	if closeErr != nil {
		return closeErr
	}
	return forceErr
}

// Close gracefully terminates the subprocess connection. It closes stdin and
// allows the CLI a bounded interval to persist session state before force-stop.
func (t *Transport) Close() error {
	t.mu.Lock()
	if t.closing {
		done := t.closeDone
		t.mu.Unlock()
		if done != nil {
			<-done
		}
		t.mu.RLock()
		err := t.closeErr
		t.mu.RUnlock()
		return err
	}
	if !t.connected {
		done := t.closeDone
		err := t.closeErr
		t.mu.Unlock()
		if done != nil {
			select {
			case <-done:
				return err
			default:
			}
		}
		return nil
	}

	t.connected = false
	t.closing = true

	watcherStop := t.watcherStop
	t.watcherStop = nil
	watcherDone := t.watcherDone
	protocol := t.protocol
	protocolAdapter := t.protocolAdapter
	stdin := t.stdin
	t.stdin = nil
	cancel := t.cancel
	transportCtx := t.ctx
	done := t.closeDone
	t.mu.Unlock()

	if watcherStop != nil {
		close(watcherStop)
	}
	if watcherDone != nil {
		<-watcherDone
	}
	if transportCtx != nil && transportCtx.Err() != nil {
		atomic.StoreUint32(&t.cancellationRequested, 1)
	}
	if stdin != nil {
		_ = stdin.Close()
	}

	err := t.terminateProcess()

	if cancel != nil {
		cancel()
	}
	if protocol != nil {
		_ = protocol.Close()
	}
	if protocolAdapter != nil {
		_ = protocolAdapter.Close()
	}

	// The tree is gone at this point. Closing read handles guarantees that
	// escaped descendants cannot keep the SDK readers blocked.
	t.closeReadPipes()
	t.waitForReaders()
	t.cleanup()

	t.mu.Lock()
	t.protocol = nil
	t.protocolAdapter = nil
	t.cancel = nil
	t.ctx = nil
	t.closeErr = err
	t.closing = false
	if done != nil {
		close(done)
	}
	t.mu.Unlock()
	return err
}

func (t *Transport) closeReadPipes() {
	if t.stdout != nil {
		_ = t.stdout.Close()
	}
	if t.stderrPipe != nil {
		_ = t.stderrPipe.Close()
	}
}

func (t *Transport) waitForReaders() {
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(ioShutdownTimeout):
	}
}
