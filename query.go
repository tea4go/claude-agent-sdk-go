package claudecode

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/tea4go/claude-agent-sdk-go/internal/cli"
	"github.com/tea4go/claude-agent-sdk-go/internal/subprocess"
)

// ErrNoMoreMessages indicates the message iterator has no more messages.
//
// ErrNoMoreMessages 表示消息迭代器已没有更多消息。
var ErrNoMoreMessages = errors.New("no more messages")

// Query executes a one-shot query with automatic cleanup.
// This follows the Python SDK pattern but uses dependency injection for transport.
//
// Query 执行一次性查询并自动清理资源。该实现遵循 Python SDK 模式，
// 但采用依赖注入方式提供 transport。
func Query(ctx context.Context, prompt string, opts ...Option) (MessageIterator, error) {
	options := NewOptions(opts...)

	if iter, ok := tryRunSkill(ctx, prompt, options, defaultSessionID); ok {
		return iter, nil
	}

	// For one-shot queries, create a transport that passes prompt as CLI argument
	// This matches the Python SDK behavior where prompt is passed via --print flag
	transport, err := createQueryTransport(prompt, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create query transport: %w", err)
	}

	return queryWithTransportAndOptions(ctx, prompt, transport, options)
}

// QueryWithTransport executes a query with a custom transport.
// The transport parameter is required and must not be nil.
//
// QueryWithTransport 使用自定义 transport 执行查询。transport 参数必需提供且不得为 nil。
func QueryWithTransport(
	ctx context.Context,
	prompt string,
	transport Transport,
	opts ...Option,
) (MessageIterator, error) {
	if transport == nil {
		return nil, fmt.Errorf("transport is required")
	}

	options := NewOptions(opts...)

	if iter, ok := tryRunSkill(ctx, prompt, options, defaultSessionID); ok {
		return iter, nil
	}
	return queryWithTransportAndOptions(ctx, prompt, transport, options)
}

// Internal helper functions
//
// queryWithTransportAndOptions 是内部辅助函数，创建管理 transport 生命周期的查询迭代器。
func queryWithTransportAndOptions(
	ctx context.Context,
	prompt string,
	transport Transport,
	options *Options,
) (MessageIterator, error) {
	if transport == nil {
		return nil, fmt.Errorf("transport is required")
	}

	// Create iterator that manages the transport lifecycle
	return &queryIterator{
		transport: transport,
		prompt:    prompt,
		ctx:       ctx,
		options:   options,
	}, nil
}

// queryIterator implements MessageIterator for simple queries
//
// queryIterator 实现 MessageIterator，用于简单的一次性查询。
type queryIterator struct {
	transport Transport
	prompt    string
	ctx       context.Context
	options   *Options
	started   bool
	msgChan   <-chan Message
	errChan   <-chan error
	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

func (qi *queryIterator) Next(_ context.Context) (Message, error) {
	qi.mu.Lock()
	if qi.closed {
		qi.mu.Unlock()
		return nil, ErrNoMoreMessages
	}

	// Initialize on first call
	if !qi.started {
		if err := qi.start(); err != nil {
			qi.mu.Unlock()
			return nil, err
		}
		qi.started = true
	}
	qi.mu.Unlock()

	// Read from message channels
	select {
	case msg, ok := <-qi.msgChan:
		if !ok {
			select {
			case err, ok := <-qi.errChan:
				if ok && err != nil {
					qi.mu.Lock()
					qi.closed = true
					qi.mu.Unlock()
					return nil, err
				}
			default:
			}

			qi.mu.Lock()
			qi.closed = true
			qi.mu.Unlock()
			return nil, ErrNoMoreMessages
		}
		return msg, nil
	case err := <-qi.errChan:
		qi.mu.Lock()
		qi.closed = true
		qi.mu.Unlock()
		return nil, err
	case <-qi.ctx.Done():
		qi.mu.Lock()
		qi.closed = true
		qi.mu.Unlock()
		return nil, qi.ctx.Err()
	}
}

func (qi *queryIterator) Close() error {
	var err error
	qi.closeOnce.Do(func() {
		qi.mu.Lock()
		qi.closed = true
		qi.mu.Unlock()
		if qi.transport != nil {
			err = qi.transport.Close()
		}
	})
	return err
}

func (qi *queryIterator) start() error {
	// Connect to transport
	if err := qi.transport.Connect(qi.ctx); err != nil {
		return fmt.Errorf("failed to connect transport: %w", err)
	}

	// Get message channels
	msgChan, errChan := qi.transport.ReceiveMessages(qi.ctx)
	qi.msgChan = msgChan
	qi.errChan = errChan

	// Send the prompt
	userMsg := &UserMessage{Content: qi.prompt}
	streamMsg := StreamMessage{
		Type:    "request",
		Message: userMsg,
	}

	if err := qi.transport.SendMessage(qi.ctx, streamMsg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// createQueryTransport creates a transport for one-shot queries with prompt as CLI argument.
//
// createQueryTransport 为一次性查询创建 transport，将 prompt 作为 CLI 参数传入。
//
// If the caller supplied a CLI path via WithCLIPath, that path is used directly
// and CLI auto-discovery is skipped. This matches the documented behaviour of
// the option (and the Python SDK's `cli_path` parameter) and lets callers
// bypass exec.LookPath when, for example, the npm shim on their platform
// mishandles command-line arguments.
func createQueryTransport(prompt string, options *Options) (Transport, error) {
	cliPath, err := resolveCLIPath(options)
	if err != nil {
		return nil, err
	}

	// Create subprocess transport with prompt as CLI argument
	return subprocess.NewWithPrompt(cliPath, options, prompt), nil
}

// resolveCLIPath returns the CLI path the transport should invoke. When
// options.CLIPath is set and non-empty, it wins over auto-discovery — the
// caller has explicitly opted out of FindCLI's PATH/well-known-location
// search.
//
// resolveCLIPath 返回 transport 应调用的 CLI 路径。当 options.CLIPath 已设置且非空时，
// 它优先于自动发现——调用方已显式选择跳过 FindCLI 对 PATH 与常见位置的搜索。
func resolveCLIPath(options *Options) (string, error) {
	if options != nil && options.CLIPath != nil && *options.CLIPath != "" {
		return *options.CLIPath, nil
	}
	return cli.FindCLI()
}
