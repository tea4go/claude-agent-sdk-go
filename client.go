package claudecode

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/tea4go/claude-agent-sdk-go/internal/subprocess"
)

const defaultSessionID = "default"

// Client provides bidirectional streaming communication with Claude Code CLI.
//
// Client 提供与 Claude Code CLI 的双向流式通信能力。
type Client interface {
	Connect(ctx context.Context, prompt ...StreamMessage) error
	Disconnect() error
	// Abort immediately terminates the active Claude CLI process tree. Unlike
	// Disconnect, it does not wait for graceful session persistence.
	Abort() error
	Query(ctx context.Context, prompt string) error
	QueryWithSession(ctx context.Context, prompt string, sessionID string) error
	QueryStream(ctx context.Context, messages <-chan StreamMessage) error
	ReceiveMessages(ctx context.Context) <-chan Message
	ReceiveResponse(ctx context.Context) MessageIterator
	Interrupt(ctx context.Context) error
	// SetModel changes the AI model during a streaming session.
	// Pass nil to reset to the default model.
	// Only works in streaming mode (after Connect()).
	SetModel(ctx context.Context, model *string) error
	// SetPermissionMode changes the permission mode during a streaming session.
	// Valid modes: PermissionModeDefault, PermissionModeAcceptEdits,
	// PermissionModePlan, PermissionModeBypassPermissions.
	// Only works in streaming mode (after Connect()).
	SetPermissionMode(ctx context.Context, mode PermissionMode) error
	// RewindFiles reverts tracked files to their state at a specific user message.
	// The messageUUID should be the UUID from a UserMessage received during the session.
	// Requires WithFileCheckpointing() or WithEnableFileCheckpointing(true) option.
	// Only works in streaming mode (after Connect()).
	RewindFiles(ctx context.Context, messageUUID string) error
	// GetMcpStatus returns the connection status of all configured MCP servers.
	// Only works in streaming mode (after Connect()).
	GetMcpStatus(ctx context.Context) (*McpStatusResponse, error)
	// GetSlashCommands returns slash commands available in the current session.
	// Only works in streaming mode (after Connect()).
	GetSlashCommands(ctx context.Context) ([]SlashCommand, error)
	GetStreamIssues() []StreamIssue
	GetStreamStats() StreamStats
	GetServerInfo(ctx context.Context) (map[string]interface{}, error)
}

// ClientImpl implements the Client interface.
//
// ClientImpl 是 Client 接口的具体实现。
type ClientImpl struct {
	mu              sync.RWMutex
	transport       Transport
	customTransport Transport // For testing with WithTransport
	options         *Options
	connected       bool
	msgChan         <-chan Message
	errChan         <-chan error
	streamErrChan   chan error // writable; receives errors from QueryStream goroutine
	injectChan      chan Message
	mergeCancel     context.CancelFunc
	mergeDone       chan struct{}
}

// NewClient creates a new Client with the given options.
//
// NewClient 使用给定的选项创建一个新的 Client。
func NewClient(opts ...Option) Client {
	options := NewOptions(opts...)
	client := &ClientImpl{
		options: options,
	}
	return client
}

// NewClientWithTransport creates a new Client with a custom transport (for testing).
//
// NewClientWithTransport 使用自定义 transport 创建 Client（用于测试）。
func NewClientWithTransport(transport Transport, opts ...Option) Client {
	options := NewOptions(opts...)
	return &ClientImpl{
		customTransport: transport,
		options:         options,
	}
}

// WithClient provides Go-idiomatic resource management equivalent to Python SDK's async context manager.
// It automatically connects to Claude Code CLI, executes the provided function, and ensures proper cleanup.
// This eliminates the need for manual Connect/Disconnect calls and prevents resource leaks.
//
// WithClient 提供符合 Go 习惯的资源管理，等价于 Python SDK 的异步上下文管理器：
// 自动连接 Claude Code CLI、执行给定函数并保证正确清理，无需手动调用 Connect/Disconnect，避免资源泄漏。
//
// The function follows Go's established resource management patterns using defer for guaranteed cleanup,
// similar to how database connections, files, and other resources are typically managed in Go.
//
// Example - Basic usage:
//
//	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
//	    return client.Query(ctx, "What is 2+2?")
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example - With configuration options:
//
//	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
//	    if err := client.Query(ctx, "Calculate the area of a circle with radius 5"); err != nil {
//	        return err
//	    }
//
//	    // Process responses
//	    for msg := range client.ReceiveMessages(ctx) {
//	        if assistantMsg, ok := msg.(*claudecode.AssistantMessage); ok {
//	            fmt.Println("Claude:", assistantMsg.Content[0].(*claudecode.TextBlock).Text)
//	        }
//	    }
//	    return nil
//	}, claudecode.WithSystemPrompt("You are a helpful math tutor"),
//	   claudecode.WithAllowedTools("Read", "Write"))
//
// The client will be automatically connected before fn is called and disconnected after fn returns,
// even if fn returns an error or panics. This provides 100% functional parity with Python SDK's
// 'async with ClaudeSDKClient()' pattern while using idiomatic Go resource management.
//
// Parameters:
//   - ctx: Context for connection management and cancellation
//   - fn: Function to execute with the connected client
//   - opts: Optional client configuration options
//
// Returns an error if connection fails or if fn returns an error.
// Disconnect errors are handled gracefully without overriding the original error from fn.
func WithClient(ctx context.Context, fn func(Client) error, opts ...Option) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	client := NewClient(opts...)

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect client: %w", err)
	}

	defer func() {
		// Following Go idiom: cleanup errors don't override the original error
		// This matches patterns in database/sql, os.File, and other stdlib packages
		if disconnectErr := client.Disconnect(); disconnectErr != nil {
			// Log cleanup errors but don't return them to preserve the original error
			// This follows the standard Go pattern for resource cleanup
			_ = disconnectErr // Explicitly acknowledge we're ignoring this error
		}
	}()

	return fn(client)
}

// WithClientTransport provides Go-idiomatic resource management with a custom transport for testing.
// This is the testing-friendly version of WithClient that accepts an explicit transport parameter.
//
// WithClientTransport 是 WithClient 的测试友好版本，接受显式的自定义 transport 参数以便测试。
//
// Usage in tests:
//
//	transport := newClientMockTransport()
//	err := WithClientTransport(ctx, transport, func(client claudecode.Client) error {
//	    return client.Query(ctx, "What is 2+2?")
//	}, opts...)
//
// Parameters:
//   - ctx: Context for connection management and cancellation
//   - transport: Custom transport to use (typically a mock for testing)
//   - fn: Function to execute with the connected client
//   - opts: Optional client configuration options
//
// Returns an error if connection fails or if fn returns an error.
// Disconnect errors are handled gracefully without overriding the original error from fn.
func WithClientTransport(ctx context.Context, transport Transport, fn func(Client) error, opts ...Option) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	client := NewClientWithTransport(transport, opts...)

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect client: %w", err)
	}

	defer func() {
		// Following Go idiom: cleanup errors don't override the original error
		if disconnectErr := client.Disconnect(); disconnectErr != nil {
			// Log cleanup errors but don't return them to preserve the original error
			_ = disconnectErr // Explicitly acknowledge we're ignoring this error
		}
	}()

	return fn(client)
}

// prepareOptions applies defaults and validates the client configuration options.
//
// prepareOptions 应用默认值并校验客户端配置选项。
func (c *ClientImpl) prepareOptions() error {
	if c.options == nil {
		return nil // Nil options are acceptable (use defaults)
	}

	// Auto-configure PermissionPromptToolName when CanUseTool callback is set.
	// This tells CLI to route permission prompts through stdio (control protocol).
	if c.options.CanUseTool != nil && c.options.PermissionPromptToolName == nil {
		stdio := "stdio"
		c.options.PermissionPromptToolName = &stdio
	}

	// Validate working directory
	if c.options.Cwd != nil {
		if _, err := os.Stat(*c.options.Cwd); os.IsNotExist(err) {
			return fmt.Errorf("working directory does not exist: %s", *c.options.Cwd)
		}
	}

	// Validate max turns
	if c.options.MaxTurns < 0 {
		return fmt.Errorf("max_turns must be non-negative, got: %d", c.options.MaxTurns)
	}

	// Validate permission mode
	if c.options.PermissionMode != nil {
		validModes := map[PermissionMode]bool{
			PermissionModeDefault:           true,
			PermissionModeAcceptEdits:       true,
			PermissionModePlan:              true,
			PermissionModeBypassPermissions: true,
		}
		if !validModes[*c.options.PermissionMode] {
			return fmt.Errorf("invalid permission mode: %s", string(*c.options.PermissionMode))
		}
	}

	return nil
}

// Connect establishes a connection to the Claude Code CLI.
//
// Connect 建立与 Claude Code CLI 的连接。
func (c *ClientImpl) Connect(ctx context.Context, _ ...StreamMessage) error {
	// Check context before acquiring lock
	if ctx.Err() != nil {
		return ctx.Err()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check context again after acquiring lock
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Validate configuration before connecting
	if err := c.prepareOptions(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Use custom transport if provided, otherwise create default
	if c.customTransport != nil {
		c.transport = c.customTransport
	} else {
		// Honor WithCLIPath when set, otherwise fall back to auto-discovery.
		cliPath, err := resolveCLIPath(c.options)
		if err != nil {
			return fmt.Errorf("claude CLI not found: %w", err)
		}

		// Create subprocess transport for streaming mode (closeStdin=false)
		c.transport = subprocess.New(cliPath, c.options, false, "sdk-go-client")
	}

	// Connect the transport
	if err := c.transport.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect transport: %w", err)
	}

	// Get message channels
	c.msgChan, c.errChan = c.transport.ReceiveMessages(ctx)
	c.streamErrChan = make(chan error, 1)
	transportMsgChan, transportErrChan := c.transport.ReceiveMessages(ctx)

	c.errChan = transportErrChan
	c.injectChan = make(chan Message, 100)
	mergedChan := make(chan Message, 100)

	mergeCtx, cancel := context.WithCancel(context.Background())
	c.mergeCancel = cancel
	c.mergeDone = make(chan struct{})
	go mergeMessageChannels(mergeCtx, mergedChan, transportMsgChan, c.injectChan, c.mergeDone)

	c.msgChan = mergedChan

	c.connected = true
	return nil
}

// Disconnect closes the connection to the Claude Code CLI.
//
// Disconnect 关闭与 Claude Code CLI 的连接。
func (c *ClientImpl) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.mergeCancel != nil {
		c.mergeCancel()
		c.mergeCancel = nil
	}
	// Do not close injectChan here. Skill handlers keep a snapshot of the
	// channel while running and may still publish their final messages as
	// Disconnect begins. Cancelling the merge goroutine is sufficient; the
	// abandoned buffered channel is reclaimed after those senders finish.
	if c.injectChan != nil {
		c.injectChan = nil
	}

	if c.transport != nil && c.connected {
		if err := c.transport.Close(); err != nil {
			return fmt.Errorf("failed to close transport: %w", err)
		}
	}
	c.connected = false
	c.transport = nil
	c.msgChan = nil
	c.errChan = nil
	c.streamErrChan = nil
	c.mergeDone = nil
	return nil
}

// Abort immediately terminates the connection to the Claude Code CLI.
// Custom transports can implement AbortableTransport for native force-stop
// behavior. Legacy transports fall back to Close.
//
// Abort 立即终止与 Claude Code CLI 的连接。自定义 transport 可实现 AbortableTransport
// 以获得原生强制停止行为；旧式 transport 则回退到 Close。
func (c *ClientImpl) Abort() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.mergeCancel != nil {
		c.mergeCancel()
		c.mergeCancel = nil
	}
	if c.injectChan != nil {
		c.injectChan = nil
	}

	var abortErr error
	if c.transport != nil && c.connected {
		if transport, ok := c.transport.(AbortableTransport); ok {
			abortErr = transport.Abort()
		} else {
			abortErr = c.transport.Close()
		}
	}

	c.connected = false
	c.transport = nil
	c.msgChan = nil
	c.errChan = nil
	c.streamErrChan = nil
	c.mergeDone = nil
	if abortErr != nil {
		return fmt.Errorf("failed to abort transport: %w", abortErr)
	}
	return nil
}

// Query sends a simple text query using the default session.
// This is equivalent to QueryWithSession(ctx, prompt, "default").
//
// Query 使用默认会话发送一条简单的文本查询，等价于 QueryWithSession(ctx, prompt, "default")。
//
// Example:
//
//	client.Query(ctx, "What is Go?")
func (c *ClientImpl) Query(ctx context.Context, prompt string) error {
	return c.queryWithSession(ctx, prompt, defaultSessionID)
}

// QueryWithSession sends a simple text query using the specified session ID.
// Each session maintains its own conversation context, allowing for isolated
// conversations within the same client connection.
//
// QueryWithSession 使用指定的会话 ID 发送文本查询。每个会话维护自己的对话上下文，
// 从而在同一客户端连接内实现相互隔离的多个对话。
//
// If sessionID is empty, it defaults to "default".
//
// Example:
//
//	client.QueryWithSession(ctx, "Remember this", "my-session")
//	client.QueryWithSession(ctx, "What did I just say?", "my-session") // Remembers context
//	client.Query(ctx, "What did I just say?")                          // Won't remember, different session
func (c *ClientImpl) QueryWithSession(ctx context.Context, prompt string, sessionID string) error {
	// Use default session if empty session ID provided
	if sessionID == "" {
		sessionID = defaultSessionID
	}
	return c.queryWithSession(ctx, prompt, sessionID)
}

// queryWithSession is the internal implementation for sending queries with session management.
//
// queryWithSession 是带会话管理的查询发送内部实现。
func (c *ClientImpl) queryWithSession(ctx context.Context, prompt string, sessionID string) error {
	// Check context before proceeding
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	// Check context again after acquiring connection info
	if ctx.Err() != nil {
		return ctx.Err()
	}

	c.mu.RLock()
	injectChan := c.injectChan
	c.mu.RUnlock()

	name, args, ok := parseSkillCommand(prompt)
	if ok {
		if handler, exists := getSkillHandler(c.options, name); exists {
			go func() {
				output, err := handler(ctx, args)

				model := "sdk-skill"

				if err != nil {
					errText := err.Error()
					enqueueInjectedMessage(ctx, injectChan, &AssistantMessage{
						Content: []ContentBlock{&TextBlock{Text: errText}},
						Model:   model,
					})
					enqueueInjectedMessage(ctx, injectChan, &ResultMessage{
						Subtype:       "error",
						DurationMs:    0,
						DurationAPIMs: 0,
						IsError:       true,
						Errors:        []string{errText},
						NumTurns:      0,
						SessionID:     sessionID,
						Result:        &errText,
					})
					return
				}

				enqueueInjectedMessage(ctx, injectChan, &AssistantMessage{
					Content: []ContentBlock{&TextBlock{Text: output}},
					Model:   model,
				})
				enqueueInjectedMessage(ctx, injectChan, &ResultMessage{
					Subtype:       "success",
					DurationMs:    0,
					DurationAPIMs: 0,
					IsError:       false,
					NumTurns:      0,
					SessionID:     sessionID,
				})
			}()
			return nil
		}
	}

	// Create user message in Python SDK compatible format
	streamMsg := StreamMessage{
		Type: "user",
		Message: map[string]interface{}{
			"role":    "user",
			"content": prompt,
		},
		ParentToolUseID: nil,
		SessionID:       sessionID,
	}

	// Send message via transport (without holding mutex to avoid blocking other operations)
	return transport.SendMessage(ctx, streamMsg)
}

// mergeMessageChannels 将 transport 消息通道与注入消息通道合并到同一个输出通道；
// 任一上游通道关闭后置 nil，直至两者均关闭或 ctx 取消时退出。
func mergeMessageChannels(
	ctx context.Context,
	out chan<- Message,
	transportMsgChan <-chan Message,
	injectChan <-chan Message,
	done chan<- struct{},
) {
	defer close(done)
	defer close(out)

	for transportMsgChan != nil || injectChan != nil {
		select {
		case msg, ok := <-transportMsgChan:
			if !ok {
				transportMsgChan = nil
				continue
			}
			select {
			case out <- msg:
			case <-ctx.Done():
				return
			}
		case msg, ok := <-injectChan:
			if !ok {
				injectChan = nil
				continue
			}
			select {
			case out <- msg:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// enqueueInjectedMessage 将一条注入消息发送到通道；通道为 nil 或 ctx 取消时返回 false。
func enqueueInjectedMessage(ctx context.Context, ch chan<- Message, msg Message) bool {
	if ch == nil {
		return false
	}
	select {
	case ch <- msg:
		return true
	case <-ctx.Done():
		return false
	}
}

// QueryStream sends a stream of messages.
//
// QueryStream 发送一个消息流（在后台 goroutine 中依次发送）。
func (c *ClientImpl) QueryStream(ctx context.Context, messages <-chan StreamMessage) error {
	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	streamErrChan := c.streamErrChan
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	// Send messages from channel in a goroutine
	go func() {
		for {
			select {
			case msg, ok := <-messages:
				if !ok {
					return // Channel closed
				}
				if err := transport.SendMessage(ctx, msg); err != nil {
					fmt.Fprintf(os.Stderr, "claude-agent-sdk: QueryStream send error: %v\n", err)
					select {
					case streamErrChan <- fmt.Errorf("stream send error: %w", err):
					default:
					}
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// ReceiveMessages returns a channel of incoming messages.
//
// ReceiveMessages 返回一个接收入站消息的通道；未连接时返回一个已关闭的通道。
func (c *ClientImpl) ReceiveMessages(_ context.Context) <-chan Message {
	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	msgChan := c.msgChan
	c.mu.RUnlock()

	if !connected || msgChan == nil {
		// Return closed channel if not connected
		closedChan := make(chan Message)
		close(closedChan)
		return closedChan
	}

	// Return the transport's message channel directly
	return msgChan
}

// ReceiveResponse returns an iterator for the response messages.
//
// ReceiveResponse 返回用于遍历响应消息的迭代器；未连接时返回包裹已关闭通道的迭代器（非 nil）。
func (c *ClientImpl) ReceiveResponse(_ context.Context) MessageIterator {
	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	msgChan := c.msgChan
	errChan := c.errChan
	streamErrChan := c.streamErrChan
	c.mu.RUnlock()

	if !connected || msgChan == nil {
		closed := make(chan Message)
		close(closed)
		return &clientIterator{msgChan: closed, errChan: make(chan error)}
	}

	return &clientIterator{
		msgChan:       msgChan,
		errChan:       errChan,
		streamErrChan: streamErrChan,
	}
}

// Interrupt sends an interrupt signal to stop the current operation.
//
// Interrupt 发送中断信号以停止当前操作。
func (c *ClientImpl) Interrupt(ctx context.Context) error {
	// Check context before proceeding
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Check connection status with read lock
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	return transport.Interrupt(ctx)
}

// SetModel changes the AI model during a streaming session.
// Pass nil to reset to the default model.
// Returns error if not connected or if the control request fails.
//
// SetModel 在流式会话期间切换 AI 模型；传 nil 则重置为默认模型。
// 未连接或控制请求失败时返回错误。
//
// Example - Change to a specific model:
//
//	model := "claude-sonnet-4-5"
//	err := client.SetModel(ctx, &model)
//
// Example - Reset to default model:
//
//	err := client.SetModel(ctx, nil)
func (c *ClientImpl) SetModel(ctx context.Context, model *string) error {
	// Check context before proceeding (Go idiom: fail fast)
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Check connection status with read lock (minimize lock duration)
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	return transport.SetModel(ctx, model)
}

// SetPermissionMode changes the permission mode during a streaming session.
// Valid modes: PermissionModeDefault, PermissionModeAcceptEdits,
// PermissionModePlan, PermissionModeBypassPermissions.
// Returns error if not connected or if the control request fails.
//
// SetPermissionMode 在流式会话期间切换权限模式。
// 有效模式：PermissionModeDefault、PermissionModeAcceptEdits、
// PermissionModePlan、PermissionModeBypassPermissions。未连接或控制请求失败时返回错误。
//
// Example - Enable auto-accept for edits:
//
//	err := client.SetPermissionMode(ctx, claudecode.PermissionModeAcceptEdits)
//
// Example - Switch to plan mode:
//
//	err := client.SetPermissionMode(ctx, claudecode.PermissionModePlan)
func (c *ClientImpl) SetPermissionMode(ctx context.Context, mode PermissionMode) error {
	// Check context before proceeding (Go idiom: fail fast)
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Check connection status with read lock (minimize lock duration)
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	return transport.SetPermissionMode(ctx, mode)
}

// RewindFiles reverts tracked files to their state at a specific user message.
// The messageUUID should be the UUID from a UserMessage received during the session.
// Requires file checkpointing to be enabled via WithFileCheckpointing() option.
// Returns error if not connected or the request fails.
//
// RewindFiles 将被跟踪的文件回滚到某条用户消息时的状态。
// messageUUID 应为会话期间收到的 UserMessage 的 UUID，
// 需通过 WithFileCheckpointing() 选项启用文件检查点。未连接或请求失败时返回错误。
//
// Example:
//
//	client := claudecode.NewClient(claudecode.WithFileCheckpointing())
//	// ... connect and receive messages, capture UUID from UserMessage
//	if msg, ok := receivedMsg.(*claudecode.UserMessage); ok && msg.UUID != nil {
//	    err := client.RewindFiles(ctx, *msg.UUID)
//	}
func (c *ClientImpl) RewindFiles(ctx context.Context, messageUUID string) error {
	// Check context before proceeding (Go idiom: fail fast)
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Check connection status with read lock (minimize lock duration)
	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return fmt.Errorf("client not connected")
	}

	return transport.RewindFiles(ctx, messageUUID)
}

// GetMcpStatus returns the connection status of all configured MCP servers.
// Returns error if not connected or if the control request fails.
//
// GetMcpStatus 返回所有已配置 MCP 服务器的连接状态。未连接或控制请求失败时返回错误。
func (c *ClientImpl) GetMcpStatus(ctx context.Context) (*McpStatusResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return nil, fmt.Errorf("client not connected")
	}

	return transport.GetMcpStatus(ctx)
}

// GetSlashCommands returns slash commands available in the current session.
// Returns error if not connected or if the control request fails.
//
// GetSlashCommands 返回当前会话中可用的斜杠命令。未连接或控制请求失败时返回错误。
func (c *ClientImpl) GetSlashCommands(ctx context.Context) ([]SlashCommand, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	c.mu.RLock()
	connected := c.connected
	transport := c.transport
	c.mu.RUnlock()

	if !connected || transport == nil {
		return nil, fmt.Errorf("client not connected")
	}

	return transport.GetSlashCommands(ctx)
}

// clientIterator implements MessageIterator for client message reception
//
// clientIterator 实现 MessageIterator，用于客户端消息接收。
type clientIterator struct {
	msgChan       <-chan Message
	errChan       <-chan error
	streamErrChan <-chan error
	mu            sync.Mutex
	closed        bool
}

func (ci *clientIterator) Next(ctx context.Context) (Message, error) {
	ci.mu.Lock()
	if ci.closed {
		ci.mu.Unlock()
		return nil, ErrNoMoreMessages
	}
	ci.mu.Unlock()

	select {
	case msg, ok := <-ci.msgChan:
		if !ok {
			select {
			case err, ok := <-ci.errChan:
				if ok && err != nil {
					ci.mu.Lock()
					ci.closed = true
					ci.mu.Unlock()
					return nil, err
				}
			case err, ok := <-ci.streamErrChan:
				if ok && err != nil {
					ci.mu.Lock()
					ci.closed = true
					ci.mu.Unlock()
					return nil, err
				}
			default:
			}

			ci.mu.Lock()
			ci.closed = true
			ci.mu.Unlock()
			return nil, ErrNoMoreMessages
		}
		return msg, nil
	case err, ok := <-ci.errChan:
		if !ok {
			ci.mu.Lock()
			ci.closed = true
			ci.mu.Unlock()
			return nil, ErrNoMoreMessages
		}
		ci.mu.Lock()
		ci.closed = true
		ci.mu.Unlock()
		return nil, err
	case err, ok := <-ci.streamErrChan:
		if !ok {
			ci.mu.Lock()
			ci.closed = true
			ci.mu.Unlock()
			return nil, ErrNoMoreMessages
		}
		ci.mu.Lock()
		ci.closed = true
		ci.mu.Unlock()
		return nil, err
	case <-ctx.Done():
		ci.mu.Lock()
		ci.closed = true
		ci.mu.Unlock()
		return nil, ctx.Err()
	}
}

func (ci *clientIterator) Close() error {
	ci.mu.Lock()
	ci.closed = true
	ci.mu.Unlock()
	return nil
}

// GetStreamIssues returns validation issues found in the message stream.
// This can help diagnose problems like missing tool results or incomplete streams.
//
// GetStreamIssues 返回消息流中发现的校验问题，可用于诊断如工具结果缺失或流不完整等问题。
func (c *ClientImpl) GetStreamIssues() []StreamIssue {
	c.mu.RLock()
	transport := c.transport
	c.mu.RUnlock()

	if transport == nil {
		return nil
	}

	validator := transport.GetValidator()
	if validator == nil {
		return nil
	}

	return validator.GetIssues()
}

// GetStreamStats returns statistics about the message stream.
// This includes counts of tools requested/received and pending tools.
//
// GetStreamStats 返回消息流的统计信息，包括已请求/已接收的工具数以及待处理工具数。
func (c *ClientImpl) GetStreamStats() StreamStats {
	c.mu.RLock()
	transport := c.transport
	c.mu.RUnlock()

	if transport == nil {
		return StreamStats{}
	}

	validator := transport.GetValidator()
	if validator == nil {
		return StreamStats{}
	}

	return validator.GetStats()
}

// GetServerInfo returns diagnostic information about the client and its connection.
// This provides useful information for debugging, health checks, and support scenarios.
//
// GetServerInfo 返回关于客户端及其连接的诊断信息，便于调试、健康检查与技术支持。
//
// This method is thread-safe and can be called concurrently from multiple goroutines.
//
// Returns a map containing:
//   - "connected": bool - Whether the client is currently connected
//   - "transport_type": string - The type of transport being used (e.g., "subprocess")
//
// Returns an error if the client is not connected.
//
// Example:
//
//	info, err := client.GetServerInfo(ctx)
//	if err != nil {
//	    log.Printf("Client not connected: %v", err)
//	    return
//	}
//	fmt.Printf("Connected: %v, Transport: %s\n",
//	    info["connected"], info["transport_type"])
func (c *ClientImpl) GetServerInfo(_ context.Context) (map[string]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.transport == nil {
		return nil, fmt.Errorf("client not connected")
	}

	info := map[string]interface{}{
		"connected":      true,
		"transport_type": "subprocess",
	}

	return info, nil
}
