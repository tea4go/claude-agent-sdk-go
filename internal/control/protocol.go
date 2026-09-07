package control

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

// DefaultInitTimeout is the default timeout for the Initialize handshake.
//
// DefaultInitTimeout 是 Initialize 握手的默认超时时间。
const DefaultInitTimeout = 60 * time.Second

// Transport abstracts the I/O operations for the control protocol.
// This allows testing with mock transports.
//
// Transport 抽象控制协议的 I/O 操作，从而可用模拟 transport 进行测试。
type Transport interface {
	// Write sends data to the CLI stdin.
	Write(ctx context.Context, data []byte) error
	// Read returns a channel that receives data from CLI stdout.
	Read(ctx context.Context) <-chan []byte
	// Close closes the transport.
	Close() error
}

// Protocol manages the bidirectional control protocol with Claude CLI.
// It handles request/response correlation, message routing, and initialization.
//
// Protocol 管理与 Claude CLI 的双向控制协议，负责请求/响应关联、
// 消息路由以及初始化。
type Protocol struct {
	mu        sync.Mutex
	transport Transport

	// Request correlation
	pendingRequests map[string]chan *Response
	requestCounter  int64

	// Message routing
	messageStream chan map[string]any

	// State
	initialized  bool
	initOnce     sync.Once
	initErr      error
	initResponse *InitializeResponse
	initErrChan  chan error
	closed       bool
	started      bool

	// Configuration
	initTimeout time.Duration

	// Permission callback
	canUseToolCallback CanUseToolCallback

	// Hook callbacks
	hooks            map[HookEvent][]HookMatcher
	hookCallbacks    map[string]HookCallback
	hookCallbacksMu  sync.RWMutex
	nextHookCallback int64

	// Session initialization config
	plugins []shared.SdkPluginConfig
	skills  *[]string

	// SDK MCP servers for in-process tool handling
	sdkMcpServers map[string]McpServer

	// Background goroutine management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ProtocolOption configures Protocol behavior.
//
// ProtocolOption 用于配置 Protocol 的行为。
type ProtocolOption func(*Protocol)

// WithInitTimeout sets the initialization timeout.
//
// WithInitTimeout 设置初始化超时时间。
func WithInitTimeout(timeout time.Duration) ProtocolOption {
	return func(p *Protocol) {
		p.initTimeout = timeout
	}
}

// WithCanUseToolCallback sets the permission callback for tool usage requests.
// The callback is invoked when CLI requests permission to use a tool.
//
// WithCanUseToolCallback 设置工具使用请求的权限回调，
// 当 CLI 请求使用某个工具时会调用该回调。
func WithCanUseToolCallback(callback CanUseToolCallback) ProtocolOption {
	return func(p *Protocol) {
		p.canUseToolCallback = callback
	}
}

// WithHooks sets the hook configuration for lifecycle events.
// Hooks are registered during initialization and invoked by the CLI.
//
// WithHooks 设置生命周期事件的钩子配置，钩子在初始化时注册并由 CLI 调用。
func WithHooks(hooks map[HookEvent][]HookMatcher) ProtocolOption {
	return func(p *Protocol) {
		p.hooks = hooks
	}
}

// WithHookCallbacks sets pre-registered hook callbacks by ID.
// This is primarily used for testing.
//
// WithHookCallbacks 按 ID 设置预注册的钩子回调，主要用于测试。
func WithHookCallbacks(callbacks map[string]HookCallback) ProtocolOption {
	return func(p *Protocol) {
		p.hookCallbacks = callbacks
	}
}

// WithSdkMcpServers configures SDK MCP servers for in-process tool handling.
// The servers map is keyed by server name.
//
// WithSdkMcpServers 配置用于进程内工具处理的 SDK MCP 服务器，map 以服务器名为键。
func WithSdkMcpServers(servers map[string]McpServer) ProtocolOption {
	return func(p *Protocol) {
		p.sdkMcpServers = servers
	}
}

// WithSessionConfig forwards session-scoped plugin and Skill filters during
// the streaming initialize handshake.
//
// WithSessionConfig 在流式 initialize 握手期间转发会话级的插件与 Skill 过滤器。
func WithSessionConfig(plugins []shared.SdkPluginConfig, skills any) ProtocolOption {
	return func(p *Protocol) {
		if len(plugins) > 0 {
			p.plugins = append([]shared.SdkPluginConfig(nil), plugins...)
		}
		if names, ok := skills.([]string); ok {
			copied := append([]string(nil), names...)
			p.skills = &copied
		}
	}
}

// NewProtocol creates a new control protocol handler.
//
// NewProtocol 创建一个新的控制协议处理器。
func NewProtocol(transport Transport, opts ...ProtocolOption) *Protocol {
	p := &Protocol{
		transport:       transport,
		pendingRequests: make(map[string]chan *Response),
		messageStream:   make(chan map[string]any, 100),
		initTimeout:     DefaultInitTimeout,
		initErrChan:     make(chan error, 1),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Start begins the message reading goroutine.
// This must be called before sending any control requests.
//
// Start 启动消息读取 goroutine，必须在发送任何控制请求之前调用。
func (p *Protocol) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return nil
	}

	p.ctx, p.cancel = context.WithCancel(ctx)
	p.started = true

	// Start background message reader
	p.wg.Add(1)
	go p.readLoop()

	return nil
}

// readLoop continuously reads from transport and routes messages.
//
// readLoop 持续从 transport 读取并路由消息。
func (p *Protocol) readLoop() {
	defer p.wg.Done()

	readChan := p.transport.Read(p.ctx)

	for {
		select {
		case <-p.ctx.Done():
			return
		case data, ok := <-readChan:
			if !ok {
				return
			}

			// Parse the incoming message
			var msg map[string]any
			if err := json.Unmarshal(data, &msg); err != nil {
				fmt.Fprintf(os.Stderr, "claude-agent-sdk: failed to parse control message: %v\n", err)
				continue
			}

			// Route the message
			if err := p.HandleIncomingMessage(p.ctx, msg); err != nil {
				fmt.Fprintf(os.Stderr, "claude-agent-sdk: failed to route control message: %v\n", err)
				continue
			}
		}
	}
}

// generateRequestID creates a unique request ID matching Python SDK format.
// Format: req_{counter}_{random_hex}
//
// generateRequestID 生成与 Python SDK 格式一致的唯一请求 ID。
// 格式：req_{计数器}_{随机十六进制}
func (p *Protocol) generateRequestID() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.requestCounter++

	// Generate 4 random bytes as hex
	randomBytes := make([]byte, 4)
	_, _ = rand.Read(randomBytes)

	return fmt.Sprintf("req_%d_%x", p.requestCounter, randomBytes)
}

// SendControlRequest sends a control request and waits for the response.
// It uses the request ID for correlation with the matching response.
//
// SendControlRequest 发送一个控制请求并等待响应，
// 通过请求 ID 将响应与对应请求关联。
func (p *Protocol) SendControlRequest(ctx context.Context, request any, timeout time.Duration) (any, error) {
	requestID := p.generateRequestID()

	// Create response channel
	responseChan := make(chan *Response, 1)

	p.mu.Lock()
	p.pendingRequests[requestID] = responseChan
	p.mu.Unlock()

	// Cleanup on exit
	defer func() {
		p.mu.Lock()
		delete(p.pendingRequests, requestID)
		p.mu.Unlock()
	}()

	// Build control request envelope
	controlReq := SDKControlRequest{
		Type:      MessageTypeControlRequest,
		RequestID: requestID,
		Request:   request,
	}

	// Serialize and send
	data, err := json.Marshal(controlReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal control request: %w", err)
	}

	// Add newline for JSON lines protocol
	data = append(data, '\n')

	if err := p.transport.Write(ctx, data); err != nil {
		return nil, fmt.Errorf("failed to send control request: %w", err)
	}

	// Wait for response with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case response := <-responseChan:
		if response.Subtype == ResponseSubtypeError {
			return nil, fmt.Errorf("control request error: %s", response.Error)
		}
		return response.Response, nil

	case err := <-p.initErrChan:
		return nil, err

	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("control request timeout: %w", timeoutCtx.Err())
	}
}

// HandleControlInitErr reports an initialization error back to any pending
// SendControlRequest, unblocking it when the CLI returns an error result
// instead of a control protocol response (e.g., invalid session ID).
//
// HandleControlInitErr 将初始化错误上报给任何处于等待中的 SendControlRequest；
// 当 CLI 返回错误结果而非控制协议响应时（如无效的会话 ID）解除其阻塞。
func (p *Protocol) HandleControlInitErr(err error) {
	select {
	case p.initErrChan <- err:
	default:
	}
}

// HandleIncomingMessage routes incoming messages based on their type.
// Control messages are handled internally, regular messages are forwarded to the stream.
//
// HandleIncomingMessage 根据消息类型路由传入的消息：
// 控制消息在内部处理，普通消息转发到消息流。
func (p *Protocol) HandleIncomingMessage(ctx context.Context, msg map[string]any) error {
	msgType, ok := msg["type"].(string)
	if !ok {
		// No type field - forward to stream for compatibility
		// 无 type 字段——为兼容转发到消息流
		return p.forwardToStream(ctx, msg)
	}

	switch msgType {
	case MessageTypeControlResponse:
		return p.handleControlResponse(ctx, msg)
	case MessageTypeControlRequest:
		// Incoming control request from CLI (e.g., hook callback, permission check)
		// 来自 CLI 的传入控制请求（如钩子回调、权限检查）
		return p.handleIncomingControlRequest(ctx, msg)
	default:
		// Regular SDK message - forward to stream
		// 普通 SDK 消息——转发到消息流
		return p.forwardToStream(ctx, msg)
	}
}

// handleIncomingControlRequest routes incoming control requests from CLI.
//
// handleIncomingControlRequest 路由来自 CLI 的传入控制请求。
func (p *Protocol) handleIncomingControlRequest(ctx context.Context, msg map[string]any) error {
	request, ok := msg["request"].(map[string]any)
	if !ok {
		return fmt.Errorf("invalid control request: missing request field")
	}

	subtype, _ := request["subtype"].(string)
	requestID, _ := msg["request_id"].(string)

	switch subtype {
	case SubtypeCanUseTool:
		return p.handleCanUseToolRequest(ctx, requestID, request)
	case SubtypeHookCallback:
		return p.handleHookCallbackRequest(ctx, requestID, request)
	case SubtypeMcpMessage:
		return p.handleMcpMessageRequest(ctx, requestID, request)
	default:
		// Unknown subtype - ignore for forward compatibility
		// 未知子类型——为向前兼容而忽略
		return nil
	}
}

// handleControlResponse routes a control response to the waiting request.
//
// handleControlResponse 将控制响应路由到等待中的请求。
func (p *Protocol) handleControlResponse(_ context.Context, msg map[string]any) error {
	responseData, ok := msg["response"].(map[string]any)
	if !ok {
		return fmt.Errorf("invalid control response: missing response field")
	}

	requestID, ok := responseData["request_id"].(string)
	if !ok {
		return fmt.Errorf("invalid control response: missing request_id")
	}

	p.mu.Lock()
	responseChan, exists := p.pendingRequests[requestID]
	p.mu.Unlock()

	if !exists {
		// Response for unknown request - ignore (could be stale or from another session)
		// 未知请求的响应——忽略（可能是陈旧消息或来自另一会话）
		return nil
	}

	response := &Response{
		RequestID: requestID,
	}

	if subtype, ok := responseData["subtype"].(string); ok {
		response.Subtype = subtype
	}

	if response.Subtype == ResponseSubtypeError {
		if errMsg, ok := responseData["error"].(string); ok {
			response.Error = errMsg
		}
	} else {
		response.Response = responseData["response"]
	}

	// Send response to waiting goroutine (non-blocking)
	// 将响应发送给等待中的 goroutine（非阻塞）
	select {
	case responseChan <- response:
	default:
		fmt.Fprintf(os.Stderr, "claude-agent-sdk: response channel full or closed, dropping response for request %s\n", requestID)
	}

	return nil
}

// forwardToStream sends a message to the regular message stream.
// Returns an error if the buffer is full rather than blocking readLoop.
//
// forwardToStream 将消息发送到普通消息流；缓冲区满时返回错误而非阻塞 readLoop。
func (p *Protocol) forwardToStream(ctx context.Context, msg map[string]any) error {
	select {
	case p.messageStream <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("message stream buffer full, dropping message")
	}
}

// sendErrorResponse sends an error response back to CLI.
// This is a shared utility used by hooks, MCP, and permissions handlers.
//
// sendErrorResponse 将错误响应回送给 CLI，是钩子、MCP 与权限处理器共用的工具方法。
func (p *Protocol) sendErrorResponse(ctx context.Context, requestID string, errMsg string) error {
	response := SDKControlResponse{
		Type: MessageTypeControlResponse,
		Response: Response{
			Subtype:   ResponseSubtypeError,
			RequestID: requestID,
			Error:     errMsg,
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal error response: %w", err)
	}

	return p.transport.Write(ctx, append(data, '\n'))
}

// Initialize performs the control protocol handshake with the CLI.
// This must be called in streaming mode before other control operations.
// The result is cached via sync.Once - concurrent and subsequent calls return the cached
// response. If the first call fails, the error is also cached permanently; subsequent
// calls return the same error and will not retry even with a fresh context.
//
// Initialize 执行与 CLI 的控制协议握手。在流式模式下必须先于其他控制操作调用。
// 结果通过 sync.Once 缓存——并发及后续调用均返回缓存的响应。若首次调用失败，
// 错误也会被永久缓存；后续调用返回相同错误，即使使用新的 context 也不会重试。
func (p *Protocol) Initialize(ctx context.Context) (*InitializeResponse, error) {
	p.initOnce.Do(func() {
		// Build initialize request with hooks configuration
		initReq := InitializeRequest{
			Subtype: SubtypeInitialize,
		}
		if len(p.plugins) > 0 {
			initReq.Plugins = append([]shared.SdkPluginConfig(nil), p.plugins...)
		}
		if p.skills != nil {
			skills := append([]string(nil), (*p.skills)...)
			initReq.Skills = &skills
		}

		// Generate hook registrations and build hooks config
		if p.hooks != nil {
			initReq.Hooks = p.buildHooksConfig()
		}

		// Send initialize request
		result, err := p.SendControlRequest(ctx, initReq, p.initTimeout)
		if err != nil {
			p.initErr = fmt.Errorf("initialize failed: %w", err)
			return
		}

		// Parse response
		var initResp InitializeResponse
		if resultMap, ok := result.(map[string]any); ok {
			if cmds, ok := resultMap["supported_commands"].([]any); ok {
				for _, cmd := range cmds {
					if cmdStr, ok := cmd.(string); ok {
						initResp.SupportedCommands = append(initResp.SupportedCommands, cmdStr)
					}
				}
			}
		}

		p.mu.Lock()
		p.initialized = true
		p.initResponse = &initResp
		p.mu.Unlock()
	})

	p.mu.Lock()
	resp := p.initResponse
	err := p.initErr
	p.mu.Unlock()

	return resp, err
}

// Interrupt sends an interrupt control request to the CLI.
//
// Interrupt 向 CLI 发送一个中断控制请求。
func (p *Protocol) Interrupt(ctx context.Context) error {
	_, err := p.SendControlRequest(ctx, InterruptRequest{
		Subtype: SubtypeInterrupt,
	}, 5*time.Second)

	return err
}

// SetModel changes the AI model during a streaming session.
// Pass nil to reset to the default model.
// Returns error if the control request fails or times out.
//
// SetModel 在流式会话中更换 AI 模型。传 nil 可重置为默认模型。
// 控制请求失败或超时时返回错误。
func (p *Protocol) SetModel(ctx context.Context, model *string) error {
	_, err := p.SendControlRequest(ctx, SetModelRequest{
		Subtype: SubtypeSetModel,
		Model:   model,
	}, 5*time.Second)

	return err
}

// SetPermissionMode changes the permission mode during a streaming session.
// Valid modes: "default", "accept_edits", "plan", "bypass_permissions"
// Returns error if the control request fails or times out.
//
// SetPermissionMode 在流式会话中更改权限模式。
// 有效模式："default"、"accept_edits"、"plan"、"bypass_permissions"。
// 控制请求失败或超时时返回错误。
func (p *Protocol) SetPermissionMode(ctx context.Context, mode string) error {
	_, err := p.SendControlRequest(ctx, SetPermissionModeRequest{
		Subtype: SubtypeSetPermissionMode,
		Mode:    mode,
	}, 5*time.Second)

	return err
}

// GetMcpStatus returns the connection status of all configured MCP servers.
//
// GetMcpStatus 返回所有已配置 MCP 服务器的连接状态。
func (p *Protocol) GetMcpStatus(ctx context.Context) (*McpStatusResponse, error) {
	result, err := p.SendControlRequest(ctx, NewGetMcpStatusRequest(), 5*time.Second)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("mcp status response: CLI returned empty response")
	}
	// SendControlRequest returns Response.Response as any (map[string]any from JSON).
	// Re-marshal + unmarshal into typed struct - necessary because SendControlRequest
	// returns any and there is no generic typed variant.
	// SendControlRequest 以 any（来自 JSON 的 map[string]any）返回 Response.Response。
	// 重新 marshal + unmarshal 为强类型结构——因 SendControlRequest 返回 any 且无泛型版本，故必需如此。
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp status response: %w", err)
	}
	var resp McpStatusResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal mcp status response: %w", err)
	}
	return &resp, nil
}

// RewindFiles reverts tracked files to their state at a specific user message.
// The userMessageID should be the UUID from a UserMessage received during the session.
// Requires EnableFileCheckpointing to be set when creating the client.
// Returns error if the control request fails or times out.
//
// RewindFiles 将被跟踪的文件回卷到某条用户消息时的状态。
// userMessageID 应为会话期间收到的 UserMessage 的 UUID。
// 需在创建客户端时设置 EnableFileCheckpointing。控制请求失败或超时时返回错误。
func (p *Protocol) RewindFiles(ctx context.Context, userMessageID string) error {
	_, err := p.SendControlRequest(ctx, RewindFilesRequest{
		Subtype:       SubtypeRewindFiles,
		UserMessageID: userMessageID,
	}, 5*time.Second)

	return err
}

// ReceiveMessages returns a channel for receiving regular (non-control) messages.
//
// ReceiveMessages 返回一个用于接收普通（非控制）消息的 channel。
func (p *Protocol) ReceiveMessages() <-chan map[string]any {
	return p.messageStream
}

// IsClosed returns whether the protocol has been closed.
//
// IsClosed 返回协议是否已关闭。
func (p *Protocol) IsClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}

// Close shuts down the protocol handler.
//
// Close 关闭协议处理器。
func (p *Protocol) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Cancel background goroutines
	// 取消后台 goroutine
	if p.cancel != nil {
		p.cancel()
	}

	// Wait for goroutines to finish
	// 等待 goroutine 结束
	p.wg.Wait()

	// Close message stream
	// 关闭消息流
	close(p.messageStream)

	return nil
}

// setPendingRequest adds a pending request for testing purposes.
//
// setPendingRequest 为测试目的添加一个待处理请求。
func (p *Protocol) setPendingRequest(requestID string, responseChan chan *Response) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pendingRequests[requestID] = responseChan
}
