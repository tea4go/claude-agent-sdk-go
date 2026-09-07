package claudecode

import (
	"context"
	"fmt"
	"sync"

	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

// Type aliases for MCP types from shared package.
// This provides a clean public API while keeping types in shared for internal use.
//
// 以下为 shared 包中 MCP 类型的类型别名：
// 在保持类型位于 shared 包供内部使用的同时，对外提供整洁的公开 API。
type (
	// McpToolResult represents the result of a tool call.
	McpToolResult = shared.McpToolResult
	// McpContent represents content returned by a tool.
	McpContent = shared.McpContent
	// McpToolDefinition describes a tool exposed by an MCP server.
	McpToolDefinition = shared.McpToolDefinition
	// McpSdkServerConfig configures an in-process SDK MCP server.
	McpSdkServerConfig = shared.McpSdkServerConfig
	// ToolAnnotations carries MCP-spec behavioral hints (title, readOnlyHint,
	// destructiveHint, idempotentHint, openWorldHint) attached to an SDK MCP tool.
	ToolAnnotations = shared.ToolAnnotations
)

// McpServerTypeSdk represents an in-process SDK MCP server.
//
// McpServerTypeSdk 表示进程内的 SDK MCP 服务器。
const McpServerTypeSdk = shared.McpServerTypeSdk

// McpToolHandler is the function signature for tool handlers.
// Context-first per Go idioms, explicit error return.
//
// McpToolHandler 是工具处理函数的签名；遵循 Go 习惯的 context 优先与显式错误返回。
//
// Example:
//
//	handler := func(ctx context.Context, args map[string]any) (*McpToolResult, error) {
//	    a, _ := args["a"].(float64)
//	    b, _ := args["b"].(float64)
//	    return &McpToolResult{
//	        Content: []McpContent{{Type: "text", Text: fmt.Sprintf("%f", a+b)}},
//	    }, nil
//	}
type McpToolHandler func(ctx context.Context, args map[string]any) (*McpToolResult, error)

// McpTool represents a tool for SDK MCP servers.
//
// Create tools using NewTool() for proper initialization.
//
// McpTool 表示 SDK MCP 服务器的一个工具；请使用 NewTool() 创建以保证正确初始化。
type McpTool struct {
	name        string
	description string
	inputSchema map[string]any
	handler     McpToolHandler
	annotations *ToolAnnotations
}

// ToolOption configures optional fields on an McpTool at construction time.
// Used as variadic args to NewTool to extend the tool without breaking the
// positional signature. See WithToolAnnotations for the canonical example.
//
// ToolOption 在构造时配置 McpTool 的可选字段；作为 NewTool 的可变参数，
// 在不破坏位置参数签名的前提下扩展工具。典型用法参见 WithToolAnnotations。
type ToolOption func(*McpTool)

// WithToolAnnotations attaches MCP-spec behavioral hints to a tool.
// The provided pointer is stored as-is and surfaced unchanged via
// McpTool.Annotations() and McpToolDefinition.Annotations.
//
// WithToolAnnotations 为工具附加 MCP 规范的行为提示；传入的指针原样存储，
// 并通过 McpTool.Annotations() 与 McpToolDefinition.Annotations 原样暴露。
//
// Example:
//
//	tool := claudecode.NewTool(
//	    "read_file", "Read a file from disk", schema, handler,
//	    claudecode.WithToolAnnotations(&claudecode.ToolAnnotations{
//	        ReadOnlyHint: ptrTo(true),
//	        OpenWorldHint: ptrTo(false),
//	    }),
//	)
func WithToolAnnotations(ann *ToolAnnotations) ToolOption {
	return func(t *McpTool) { t.annotations = ann }
}

// NewTool creates a new MCP tool definition.
//
// NewTool 创建一个新的 MCP 工具定义。
//
// Example:
//
//	addTool := claudecode.NewTool(
//	    "add",
//	    "Add two numbers together",
//	    map[string]any{
//	        "type": "object",
//	        "properties": map[string]any{
//	            "a": map[string]any{"type": "number"},
//	            "b": map[string]any{"type": "number"},
//	        },
//	        "required": []string{"a", "b"},
//	    },
//	    func(ctx context.Context, args map[string]any) (*claudecode.McpToolResult, error) {
//	        a, _ := args["a"].(float64)
//	        b, _ := args["b"].(float64)
//	        return &claudecode.McpToolResult{
//	            Content: []claudecode.McpContent{
//	                {Type: "text", Text: fmt.Sprintf("%.2f + %.2f = %.2f", a, b, a+b)},
//	            },
//	        }, nil
//	    },
//	)
func NewTool(name, description string, inputSchema map[string]any, handler McpToolHandler, opts ...ToolOption) *McpTool {
	t := &McpTool{
		name:        name,
		description: description,
		inputSchema: inputSchema,
		handler:     handler,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool's name.
//
// Name 返回工具名称。
func (t *McpTool) Name() string {
	return t.name
}

// Description returns the tool's description.
//
// Description 返回工具描述。
func (t *McpTool) Description() string {
	return t.description
}

// InputSchema returns the tool's input JSON schema.
//
// InputSchema 返回工具的输入 JSON schema。
func (t *McpTool) InputSchema() map[string]any {
	return t.inputSchema
}

// Annotations returns the tool's MCP-spec annotations, or nil if none
// were attached via WithToolAnnotations.
//
// Annotations 返回工具的 MCP 规范注解；若未通过 WithToolAnnotations 附加则返回 nil。
func (t *McpTool) Annotations() *ToolAnnotations {
	return t.annotations
}

// Call executes the tool handler with the given context and arguments.
// Returns an error if no handler is set.
//
// Call 使用给定的 context 与参数执行工具处理函数；未设置处理函数时返回错误。
func (t *McpTool) Call(ctx context.Context, args map[string]any) (*McpToolResult, error) {
	if t.handler == nil {
		return nil, fmt.Errorf("tool '%s' has no handler", t.name)
	}
	return t.handler(ctx, args)
}

// SdkMcpServer implements the McpServer interface for in-process tools.
// It is thread-safe and can handle concurrent tool calls.
//
// SdkMcpServer 为进程内工具实现 McpServer 接口；它是线程安全的，可处理并发工具调用。
type SdkMcpServer struct {
	name    string
	version string
	mu      sync.RWMutex
	tools   map[string]*McpTool
}

// CreateSDKMcpServer creates an in-process MCP server with the given tools.
// This is the Go equivalent of Python's create_sdk_mcp_server().
//
// CreateSDKMcpServer 使用给定工具创建一个进程内 MCP 服务器，
// 等价于 Python 的 create_sdk_mcp_server()。
//
// Example:
//
//	calculator := claudecode.CreateSDKMcpServer("calculator", "1.0.0", addTool, sqrtTool)
//
//	client := claudecode.NewClient(
//	    claudecode.WithSdkMcpServer("calc", calculator),
//	    claudecode.WithAllowedTools("mcp__calc__add", "mcp__calc__sqrt"),
//	)
func CreateSDKMcpServer(name, version string, tools ...*McpTool) *McpSdkServerConfig {
	server := &SdkMcpServer{
		name:    name,
		version: version,
		tools:   make(map[string]*McpTool),
	}
	for _, tool := range tools {
		if tool != nil {
			server.tools[tool.Name()] = tool
		}
	}
	return &McpSdkServerConfig{
		Type:     McpServerTypeSdk,
		Name:     name,
		Instance: server,
	}
}

// Name returns the server name.
//
// Name 返回服务器名称。
func (s *SdkMcpServer) Name() string {
	return s.name
}

// Version returns the server version.
//
// Version 返回服务器版本。
func (s *SdkMcpServer) Version() string {
	return s.version
}

// ListTools returns all registered tools.
// This method is thread-safe.
//
// ListTools 返回所有已注册的工具。该方法是线程安全的。
func (s *SdkMcpServer) ListTools(_ context.Context) ([]McpToolDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	defs := make([]McpToolDefinition, 0, len(s.tools))
	for _, tool := range s.tools {
		defs = append(defs, McpToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
			Annotations: tool.Annotations(),
		})
	}
	return defs, nil
}

// CallTool executes a tool by name with the given arguments.
// Returns an error if the tool is not found.
// This method is thread-safe.
//
// CallTool 按名称使用给定参数执行工具；工具不存在时返回错误。该方法是线程安全的。
func (s *SdkMcpServer) CallTool(ctx context.Context, name string, args map[string]any) (*McpToolResult, error) {
	s.mu.RLock()
	tool, exists := s.tools[name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	return tool.Call(ctx, args)
}
