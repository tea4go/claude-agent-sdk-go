// Package control provides the SDK control protocol for bidirectional communication with Claude CLI.
// This package enables features like tool permission callbacks, hook callbacks, and MCP message routing.
//
// control 包实现 SDK 与 Claude CLI 之间双向通信的控制协议，支持工具权限回调、
// 钩子回调以及 MCP 消息路由等能力。
package control

import (
	"context"

	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

// Message type constants for control protocol discrimination.
//
// 控制协议消息类型常量，用于区分请求与响应。
const (
	// MessageTypeControlRequest is sent TO the CLI to request an action.
	// MessageTypeControlRequest 发送给 CLI 以请求执行某个动作。
	MessageTypeControlRequest = "control_request"
	// MessageTypeControlResponse is received FROM the CLI as a response.
	// MessageTypeControlResponse 从 CLI 接收，作为响应。
	MessageTypeControlResponse = "control_response"
)

// Request subtype constants.
//
// 控制请求子类型常量。
const (
	// SubtypeInterrupt requests interruption of current operation.
	// SubtypeInterrupt 请求中断当前操作。
	SubtypeInterrupt = "interrupt"
	// SubtypeCanUseTool requests permission to use a tool.
	// SubtypeCanUseTool 请求使用某个工具的权限。
	SubtypeCanUseTool = "can_use_tool"
	// SubtypeInitialize performs the control protocol handshake.
	// SubtypeInitialize 执行控制协议握手。
	SubtypeInitialize = "initialize"
	// SubtypeSetPermissionMode changes the permission mode at runtime.
	// SubtypeSetPermissionMode 在运行时更改权限模式。
	SubtypeSetPermissionMode = "set_permission_mode"
	// SubtypeSetModel changes the AI model at runtime.
	// SubtypeSetModel 在运行时更换 AI 模型。
	SubtypeSetModel = "set_model"
	// SubtypeHookCallback invokes a registered hook callback.
	// SubtypeHookCallback 调用已注册的钩子回调。
	SubtypeHookCallback = "hook_callback"
	// SubtypeMcpMessage routes an MCP message to an SDK MCP server.
	// SubtypeMcpMessage 将 MCP 消息路由到 SDK MCP 服务器。
	SubtypeMcpMessage = "mcp_message"
	// SubtypeRewindFiles requests file rewind to a specific user message state.
	// SubtypeRewindFiles 请求将文件回卷到某条用户消息时的状态。
	SubtypeRewindFiles = "rewind_files"
)

// Response subtype constants for control responses.
//
// 控制响应的子类型常量。
const (
	// ResponseSubtypeSuccess indicates the request succeeded.
	// ResponseSubtypeSuccess 表示请求成功。
	ResponseSubtypeSuccess = "success"
	// ResponseSubtypeError indicates the request failed.
	// ResponseSubtypeError 表示请求失败。
	ResponseSubtypeError = "error"
)

// SDKControlRequest represents a control request sent TO the CLI.
// This is the envelope that wraps all control request types.
//
// SDKControlRequest 表示发送给 CLI 的控制请求，是包裹所有控制请求类型的信封。
type SDKControlRequest struct {
	// Type is always MessageTypeControlRequest.
	Type string `json:"type"`
	// RequestID is a unique identifier for request/response correlation.
	// Format: req_{counter}_{random_hex}
	RequestID string `json:"request_id"`
	// Request contains the actual request payload (InterruptRequest, InitializeRequest, etc.).
	Request any `json:"request"`
}

// SDKControlResponse represents a control response received FROM the CLI.
// This is the envelope that wraps all control response types.
//
// SDKControlResponse 表示从 CLI 接收的控制响应，是包裹所有控制响应类型的信封。
type SDKControlResponse struct {
	// Type is always MessageTypeControlResponse.
	Type string `json:"type"`
	// Response contains the actual response data.
	Response Response `json:"response"`
}

// Response is the inner response structure within SDKControlResponse.
//
// Response 是 SDKControlResponse 内部的响应结构，成功时填 Response，失败时填 Error。
type Response struct {
	// Subtype is either ResponseSubtypeSuccess or ResponseSubtypeError.
	Subtype string `json:"subtype"`
	// RequestID matches the request that this response is for.
	RequestID string `json:"request_id"`
	// Response contains the response data (only for success).
	Response any `json:"response,omitempty"`
	// Error contains the error message (only for error).
	Error string `json:"error,omitempty"`
}

// InterruptRequest requests interruption of the current operation.
//
// InterruptRequest 请求中断当前操作。
type InterruptRequest struct {
	// Subtype is always SubtypeInterrupt.
	Subtype string `json:"subtype"`
}

// InitializeRequest performs the control protocol handshake.
// This must be sent before any other control requests in streaming mode.
//
// InitializeRequest 执行控制协议握手，在流式模式下必须先于其他控制请求发送。
type InitializeRequest struct {
	// Subtype is always SubtypeInitialize.
	Subtype string `json:"subtype"`
	// Hooks contains hook registrations keyed by event type.
	// Format: {"PreToolUse": [...], "PostToolUse": [...]}
	Hooks map[string][]HookMatcherConfig `json:"hooks,omitempty"`
	// Plugins contains local plugin directories to expose in streaming sessions.
	Plugins []shared.SdkPluginConfig `json:"plugins,omitempty"`
	// Skills filters which discovered Skills are loaded for this session.
	Skills *[]string `json:"skills,omitempty"`
}

// InitializeResponse contains the CLI's response to initialization.
//
// InitializeResponse 包含 CLI 对初始化的响应（列出支持的控制命令）。
type InitializeResponse struct {
	// SupportedCommands lists the control commands supported by this CLI version.
	SupportedCommands []string `json:"supported_commands,omitempty"`
}

// SetPermissionModeRequest changes the permission mode at runtime.
//
// SetPermissionModeRequest 在运行时更改权限模式。
type SetPermissionModeRequest struct {
	// Subtype is always SubtypeSetPermissionMode.
	Subtype string `json:"subtype"`
	// Mode is the new permission mode to set.
	Mode string `json:"mode"`
}

// SetModelRequest changes the AI model at runtime.
//
// SetModelRequest 在运行时更换 AI 模型，Model 为 nil 时重置为默认模型。
type SetModelRequest struct {
	// Subtype is always SubtypeSetModel.
	Subtype string `json:"subtype"`
	// Model is the new model to use. Use nil to reset to default.
	// Examples: "claude-sonnet-4-5", "claude-opus-4-1-20250805"
	Model *string `json:"model"`
}

// RewindFilesRequest requests rewinding files to a specific user message state.
//
// RewindFilesRequest 请求将文件回卷到某条用户消息时的状态。
type RewindFilesRequest struct {
	// Subtype is always SubtypeRewindFiles ("rewind_files").
	Subtype string `json:"subtype"`
	// UserMessageID is the UUID of the user message to rewind to.
	// This should be obtained from UserMessage.UUID received during the session.
	UserMessageID string `json:"user_message_id"`
}

// PermissionUpdateType specifies the type of permission update.
//
// PermissionUpdateType 指定权限更新的类型。
type PermissionUpdateType string

const (
	// PermissionUpdateTypeAddRules adds new permission rules.
	PermissionUpdateTypeAddRules PermissionUpdateType = "addRules"
	// PermissionUpdateTypeReplaceRules replaces all permission rules.
	PermissionUpdateTypeReplaceRules PermissionUpdateType = "replaceRules"
	// PermissionUpdateTypeRemoveRules removes specified permission rules.
	PermissionUpdateTypeRemoveRules PermissionUpdateType = "removeRules"
	// PermissionUpdateTypeSetMode sets the permission mode.
	PermissionUpdateTypeSetMode PermissionUpdateType = "setMode"
	// PermissionUpdateTypeAddDirectories adds directories to allowed list.
	PermissionUpdateTypeAddDirectories PermissionUpdateType = "addDirectories"
	// PermissionUpdateTypeRemoveDirectories removes directories from allowed list.
	PermissionUpdateTypeRemoveDirectories PermissionUpdateType = "removeDirectories"
)

// PermissionRuleValue represents a permission rule.
// JSON tags use camelCase to match CLI protocol.
//
// PermissionRuleValue 表示一条权限规则；JSON 标签采用驼峰命名以匹配 CLI 协议。
type PermissionRuleValue struct {
	// ToolName is the name of the tool this rule applies to.
	ToolName string `json:"toolName"`
	// RuleContent is the optional rule content (e.g., path pattern).
	RuleContent *string `json:"ruleContent,omitempty"`
}

// PermissionUpdate represents a dynamic permission rule update.
//
// PermissionUpdate 表示一次动态权限规则更新。
type PermissionUpdate struct {
	// Type is the kind of permission update.
	Type PermissionUpdateType `json:"type"`
	// Rules are the permission rules to add/replace/remove.
	Rules []PermissionRuleValue `json:"rules,omitempty"`
	// Behavior is the permission behavior (allow/deny).
	Behavior *string `json:"behavior,omitempty"`
	// Mode is the permission mode to set.
	Mode *string `json:"mode,omitempty"`
	// Directories are the directories to add/remove.
	Directories []string `json:"directories,omitempty"`
	// Destination specifies where the update applies (session/user/project).
	Destination *string `json:"destination,omitempty"`
}

// ToolPermissionContext provides context for permission callbacks.
//
// ToolPermissionContext 为权限回调提供上下文（包括 CLI 提供的权限建议）。
type ToolPermissionContext struct {
	// Signal is reserved for future abort signal support (currently unused).
	Signal any `json:"-"`
	// Suggestions contains permission suggestions from CLI.
	Suggestions []PermissionUpdate `json:"suggestions,omitempty"`
}

// PermissionResult is the interface for permission callback results.
// Go idiom: unexported marker method for sealed interface pattern.
//
// PermissionResult 是权限回调结果的接口；采用未导出标记方法实现封闭接口模式，
// 仅允许本包内的 Allow/Deny 类型实现它。
type PermissionResult interface {
	permissionResult() // 标记方法 - 未导出（小写）
}

// PermissionResultAllow permits tool execution with optional modifications.
// Behavior field is always "allow" - this is the discriminator for CLI.
//
// PermissionResultAllow 允许工具执行，并可选地修改输入与权限。Behavior 固定为 "allow"，
// 是 CLI 用于判别的字段。
type PermissionResultAllow struct {
	// Behavior is always "allow".
	Behavior string `json:"behavior"`
	// UpdatedInput contains the modified tool input (optional).
	UpdatedInput map[string]any `json:"updatedInput,omitempty"`
	// UpdatedPermissions contains dynamic permission updates (optional).
	UpdatedPermissions []PermissionUpdate `json:"updatedPermissions,omitempty"`
}

// permissionResult implements PermissionResult marker interface.
func (PermissionResultAllow) permissionResult() {}

// NewPermissionResultAllow creates an Allow result with proper defaults.
// Go idiom: constructor functions for types with required fields.
//
// NewPermissionResultAllow 创建带正确默认值（Behavior="allow"）的 Allow 结果。
func NewPermissionResultAllow() PermissionResultAllow {
	return PermissionResultAllow{Behavior: "allow"}
}

// PermissionResultDeny prevents tool execution.
// Behavior field is always "deny" - this is the discriminator for CLI.
//
// PermissionResultDeny 阻止工具执行。Behavior 固定为 "deny"，是 CLI 用于判别的字段。
type PermissionResultDeny struct {
	// Behavior is always "deny".
	Behavior string `json:"behavior"`
	// Message is the reason for denial.
	Message string `json:"message,omitempty"`
	// Interrupt indicates whether to interrupt the session.
	Interrupt bool `json:"interrupt,omitempty"`
}

// permissionResult implements PermissionResult marker interface.
func (PermissionResultDeny) permissionResult() {}

// NewPermissionResultDeny creates a Deny result with proper defaults.
//
// NewPermissionResultDeny 创建带正确默认值（Behavior="deny"）的 Deny 结果。
func NewPermissionResultDeny(message string) PermissionResultDeny {
	return PermissionResultDeny{Behavior: "deny", Message: message}
}

// CanUseToolCallback is invoked when CLI requests permission to use a tool.
// Go idiom: context.Context as first parameter, (result, error) return.
// The callback must be thread-safe as it may be invoked concurrently.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - toolName: Name of the tool being requested (e.g., "Read", "Write", "Bash")
//   - input: Tool input parameters as a map
//   - permCtx: Context with permission suggestions from CLI
//
// Returns:
//   - PermissionResult: Either PermissionResultAllow or PermissionResultDeny
//   - error: Non-nil if the callback encounters an error
//
// CanUseToolCallback 在 CLI 请求使用某个工具的权限时被调用。
// 遵循 Go 惯例：首个参数为 context.Context，返回 (结果, 错误)。
// 由于可能被并发调用，该回调必须是线程安全的。
//
// 参数：
//   - ctx: 用于取消与超时的上下文
//   - toolName: 被请求的工具名（如 "Read"、"Write"、"Bash"）
//   - input: 以 map 形式表示的工具输入参数
//   - permCtx: 携带 CLI 权限建议的上下文
//
// 返回：
//   - PermissionResult: PermissionResultAllow 或 PermissionResultDeny
//   - error: 回调发生错误时非 nil
type CanUseToolCallback func(
	ctx context.Context,
	toolName string,
	input map[string]any,
	permCtx ToolPermissionContext,
) (PermissionResult, error)

// SubtypeGetMcpStatus is the control request subtype for querying MCP server status.
// Wire value: {"subtype": "mcp_status"}.
//
// SubtypeGetMcpStatus 是查询 MCP 服务器状态的控制请求子类型，线格式为 {"subtype": "mcp_status"}。
const SubtypeGetMcpStatus = "mcp_status"

// GetMcpStatusRequest requests the status of all configured MCP servers.
//
// GetMcpStatusRequest 请求获取所有已配置 MCP 服务器的状态。
type GetMcpStatusRequest struct {
	Subtype string `json:"subtype"`
}

// NewGetMcpStatusRequest creates a properly initialized GetMcpStatusRequest.
//
// NewGetMcpStatusRequest 创建一个正确初始化（已设置 Subtype）的 GetMcpStatusRequest。
func NewGetMcpStatusRequest() GetMcpStatusRequest {
	return GetMcpStatusRequest{Subtype: SubtypeGetMcpStatus}
}

// SlashCommand describes a CLI slash command suitable for input suggestions.
//
// SlashCommand 描述一个可用于输入建议的 CLI 斜杠命令。
type SlashCommand struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// McpServerConnectionStatus represents the connection state of an MCP server.
//
// McpServerConnectionStatus 表示 MCP 服务器的连接状态。
type McpServerConnectionStatus string

const (
	// McpServerConnectionStatusConnected indicates the server is connected and ready.
	// McpServerConnectionStatusConnected 表示服务器已连接且就绪。
	McpServerConnectionStatusConnected McpServerConnectionStatus = "connected"
	// McpServerConnectionStatusFailed indicates the server failed to connect.
	// McpServerConnectionStatusFailed 表示服务器连接失败。
	McpServerConnectionStatusFailed McpServerConnectionStatus = "failed"
	// McpServerConnectionStatusNeedsAuth indicates the server requires authentication.
	// McpServerConnectionStatusNeedsAuth 表示服务器需要认证。
	McpServerConnectionStatusNeedsAuth McpServerConnectionStatus = "needs-auth"
	// McpServerConnectionStatusPending indicates the server connection is in progress.
	// McpServerConnectionStatusPending 表示服务器正在连接中。
	McpServerConnectionStatusPending McpServerConnectionStatus = "pending"
	// McpServerConnectionStatusDisabled indicates the server is disabled.
	// McpServerConnectionStatusDisabled 表示服务器已被禁用。
	McpServerConnectionStatusDisabled McpServerConnectionStatus = "disabled"
)

// McpServerInfo contains version information about a connected MCP server.
//
// McpServerInfo 包含已连接 MCP 服务器的版本信息。
type McpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// McpToolAnnotations describes behavioral hints for an MCP tool.
// All fields are optional (pointer) to distinguish "not set" from false.
//
// McpToolAnnotations 描述 MCP 工具的行为提示。所有字段均为可选（指针），
// 以便区分“未设置”与 false。
type McpToolAnnotations struct {
	ReadOnly    *bool `json:"readOnly,omitempty"`
	Destructive *bool `json:"destructive,omitempty"`
	OpenWorld   *bool `json:"openWorld,omitempty"`
}

// McpToolInfo describes a tool exposed by an MCP server.
//
// McpToolInfo 描述 MCP 服务器暴露的一个工具。
type McpToolInfo struct {
	Name        string              `json:"name"`
	Description *string             `json:"description,omitempty"`
	Annotations *McpToolAnnotations `json:"annotations,omitempty"`
}

// McpServerStatusConfig is a flat struct covering all server config variants
// (stdio/sse/http/sdk/claudeai-proxy), discriminated by Type.
//
// McpServerStatusConfig 是一个扁平结构，覆盖所有服务器配置变体
// （stdio/sse/http/sdk/claudeai-proxy），通过 Type 字段判别。
type McpServerStatusConfig struct {
	Type    string            `json:"type"`
	Command *string           `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     *string           `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Name    *string           `json:"name,omitempty"`
	ID      *string           `json:"id,omitempty"`
}

// MCP server config type constants for McpServerStatusConfig.Type.
//
// 用于 McpServerStatusConfig.Type 的 MCP 服务器配置类型常量。
const (
	McpServerConfigTypeStdio    = "stdio"
	McpServerConfigTypeSSE      = "sse"
	McpServerConfigTypeHTTP     = "http"
	McpServerConfigTypeSDK      = "sdk"
	McpServerConfigTypeClaudeAI = "claudeai-proxy"
)

// McpServerStatus contains the full status of a single MCP server.
//
// McpServerStatus 包含单个 MCP 服务器的完整状态。
type McpServerStatus struct {
	Name   string                    `json:"name"`
	Status McpServerConnectionStatus `json:"status"`
	// ServerInfo contains version info. Only non-nil when Status is McpServerConnectionStatusConnected.
	ServerInfo *McpServerInfo `json:"serverInfo,omitempty"`
	// Error contains the error message. Only non-nil when Status is McpServerConnectionStatusFailed.
	Error  *string                `json:"error,omitempty"`
	Config *McpServerStatusConfig `json:"config,omitempty"`
	Scope  *string                `json:"scope,omitempty"`
	// Tools lists tools exposed by this server. Only populated when Status is McpServerConnectionStatusConnected.
	Tools []McpToolInfo `json:"tools,omitempty"`
}

// McpStatusResponse is the response payload for a GetMcpStatus request.
//
// McpStatusResponse 是 GetMcpStatus 请求的响应负载。
type McpStatusResponse struct {
	McpServers []McpServerStatus `json:"mcpServers"`
}

// Type aliases for MCP types from shared package.
// Using type aliases (not type definitions) ensures interface compatibility:
// shared.McpServer and control.McpServer are the same type, so transport can
// pass shared.McpServer to control.WithSdkMcpServers().
//
// 来自 shared 包的 MCP 类型别名。使用类型别名（而非类型定义）可保证接口兼容：
// shared.McpServer 与 control.McpServer 是同一类型，因此 transport 可将
// shared.McpServer 传给 control.WithSdkMcpServers()。
type (
	// McpServer is the interface for in-process SDK MCP servers.
	McpServer = shared.McpServer
	// McpToolDefinition describes a tool exposed by an MCP server.
	McpToolDefinition = shared.McpToolDefinition
	// McpToolResult represents the result of a tool call.
	McpToolResult = shared.McpToolResult
	// McpContent represents content returned by a tool.
	McpContent = shared.McpContent
	// ToolAnnotations carries MCP-spec behavioral hints attached to an SDK MCP tool.
	// Distinct from McpToolAnnotations above, which is the CLI-stripped response
	// shape returned by GetMcpStatus.
	ToolAnnotations = shared.ToolAnnotations
)
