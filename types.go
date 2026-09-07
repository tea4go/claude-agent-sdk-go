package claudecode

import (
	"context"

	"github.com/tea4go/claude-agent-sdk-go/internal/control"
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

// Message represents any message type in the conversation.
//
// Message 表示对话中的任意消息类型。
type Message = shared.Message

// ContentBlock represents a content block within a message.
//
// ContentBlock 表示消息中的一个内容块。
type ContentBlock = shared.ContentBlock

// UserMessage represents a message from the user.
//
// UserMessage 表示来自用户的消息。
type UserMessage = shared.UserMessage

// AssistantMessage represents a message from the assistant.
//
// AssistantMessage 表示来自助手的消息。
type AssistantMessage = shared.AssistantMessage

// AssistantMessageError represents error types in assistant messages.
//
// AssistantMessageError 表示助手消息中的错误类型。
type AssistantMessageError = shared.AssistantMessageError

// SystemMessage represents a system prompt message.
//
// SystemMessage 表示系统提示消息。
type SystemMessage = shared.SystemMessage

// ResultMessage represents a result or status message.
//
// ResultMessage 表示结果或状态消息。
type ResultMessage = shared.ResultMessage

// TextBlock represents a text content block.
//
// TextBlock 表示文本内容块。
type TextBlock = shared.TextBlock

// ThinkingBlock represents a thinking content block.
//
// ThinkingBlock 表示思考内容块。
type ThinkingBlock = shared.ThinkingBlock

// ToolUseBlock represents a tool usage content block.
//
// ToolUseBlock 表示工具调用内容块。
type ToolUseBlock = shared.ToolUseBlock

// ToolResultBlock represents a tool result content block.
//
// ToolResultBlock 表示工具结果内容块。
type ToolResultBlock = shared.ToolResultBlock

// StreamMessage represents a message in the streaming protocol.
//
// StreamMessage 表示流式协议中的一条消息。
type StreamMessage = shared.StreamMessage

// RateLimitEventMessage is a session heartbeat carrying rate-limit window
// state. Emitted on essentially every CLI session even when nothing is
// constrained — see IsAllowed for the quick "all good" check.
//
// RateLimitEventMessage 是携带限流窗口状态的会话心跳。几乎每个 CLI 会话都会发出（
// 即使并未受限）——可用 IsAllowed 快速判断“一切正常”。
type RateLimitEventMessage = shared.RateLimitEventMessage

// RateLimitInfo is the window state carried by RateLimitEventMessage.
//
// RateLimitInfo 是 RateLimitEventMessage 携带的限流窗口状态。
type RateLimitInfo = shared.RateLimitInfo

// MessageIterator provides iteration over messages.
//
// MessageIterator 提供对消息的迭代能力。
type MessageIterator = shared.MessageIterator

// StreamValidator tracks tool requests and results to detect incomplete streams.
//
// StreamValidator 跟踪工具请求与结果，用于检测不完整的流。
type StreamValidator = shared.StreamValidator

// StreamIssue represents a validation issue found in the stream.
//
// StreamIssue 表示在流中发现的一个校验问题。
type StreamIssue = shared.StreamIssue

// StreamStats provides statistics about the message stream.
//
// StreamStats 提供关于消息流的统计信息。
type StreamStats = shared.StreamStats

// Re-export message type constants
//
// 重新导出消息类型常量。
const (
	MessageTypeUser      = shared.MessageTypeUser
	MessageTypeAssistant = shared.MessageTypeAssistant
	MessageTypeSystem    = shared.MessageTypeSystem
	MessageTypeResult    = shared.MessageTypeResult

	// Control protocol message types
	MessageTypeControlRequest  = shared.MessageTypeControlRequest
	MessageTypeControlResponse = shared.MessageTypeControlResponse

	// Partial message streaming type
	MessageTypeStreamEvent = shared.MessageTypeStreamEvent

	// Session heartbeat carrying rate-limit window state.
	MessageTypeRateLimitEvent = shared.MessageTypeRateLimitEvent
)

// Rate-limit window status constants.
//
// 限流窗口状态常量。
const (
	RateLimitStatusAllowed = shared.RateLimitStatusAllowed
)

// Re-export content block type constants
//
// 重新导出内容块类型常量。
const (
	ContentBlockTypeText       = shared.ContentBlockTypeText
	ContentBlockTypeThinking   = shared.ContentBlockTypeThinking
	ContentBlockTypeToolUse    = shared.ContentBlockTypeToolUse
	ContentBlockTypeToolResult = shared.ContentBlockTypeToolResult
)

// Re-export stream event type constants for Event["type"] discrimination.
//
// 重新导出流事件类型常量，用于对 Event["type"] 进行判别。
const (
	StreamEventTypeContentBlockStart = shared.StreamEventTypeContentBlockStart
	StreamEventTypeContentBlockDelta = shared.StreamEventTypeContentBlockDelta
	StreamEventTypeContentBlockStop  = shared.StreamEventTypeContentBlockStop
	StreamEventTypeMessageStart      = shared.StreamEventTypeMessageStart
	StreamEventTypeMessageDelta      = shared.StreamEventTypeMessageDelta
	StreamEventTypeMessageStop       = shared.StreamEventTypeMessageStop
)

// Re-export AssistantMessageError constants
//
// 重新导出 AssistantMessageError 常量。
const (
	AssistantMessageErrorAuthFailed     = shared.AssistantMessageErrorAuthFailed
	AssistantMessageErrorBilling        = shared.AssistantMessageErrorBilling
	AssistantMessageErrorRateLimit      = shared.AssistantMessageErrorRateLimit
	AssistantMessageErrorInvalidRequest = shared.AssistantMessageErrorInvalidRequest
	AssistantMessageErrorServer         = shared.AssistantMessageErrorServer
	AssistantMessageErrorUnknown        = shared.AssistantMessageErrorUnknown
)

// Re-export stop reason constants
//
// 重新导出停止原因（stop reason）常量。
const (
	StopReasonEndTurn      = shared.StopReasonEndTurn
	StopReasonToolUse      = shared.StopReasonToolUse
	StopReasonStopSequence = shared.StopReasonStopSequence
	StopReasonMaxTokens    = shared.StopReasonMaxTokens
)

// AgentModel represents the model to use for an agent.
//
// AgentModel 表示一个代理（agent）所使用的模型。
type AgentModel = shared.AgentModel

// AgentDefinition defines a programmatic subagent.
//
// AgentDefinition 定义一个编程式子代理（subagent）。
type AgentDefinition = shared.AgentDefinition

// Re-export agent model constants
//
// 重新导出代理模型常量。
const (
	AgentModelSonnet  = shared.AgentModelSonnet
	AgentModelOpus    = shared.AgentModelOpus
	AgentModelHaiku   = shared.AgentModelHaiku
	AgentModelInherit = shared.AgentModelInherit
)

// Transport abstracts the communication layer with Claude Code CLI.
// This interface stays in main package because it's used by client code.
//
// Transport 抽象与 Claude Code CLI 的通信层。
// 该接口保留在主包中，因为它被客户端代码使用。
type Transport interface {
	Connect(ctx context.Context) error
	SendMessage(ctx context.Context, message StreamMessage) error
	ReceiveMessages(ctx context.Context) (<-chan Message, <-chan error)
	Interrupt(ctx context.Context) error
	// SetModel changes the AI model during streaming session.
	SetModel(ctx context.Context, model *string) error
	// SetPermissionMode changes the permission mode during streaming session.
	SetPermissionMode(ctx context.Context, mode PermissionMode) error
	// RewindFiles reverts tracked files to their state at a specific user message.
	// Requires file checkpointing to be enabled and control protocol initialized.
	RewindFiles(ctx context.Context, userMessageID string) error
	// GetMcpStatus returns the connection status of all configured MCP servers.
	GetMcpStatus(ctx context.Context) (*McpStatusResponse, error)
	// GetSlashCommands returns slash commands available in the current session.
	GetSlashCommands(ctx context.Context) ([]SlashCommand, error)
	Close() error
	GetValidator() *StreamValidator
}

// AbortableTransport is an optional transport capability for immediately
// terminating an active connection. Client.Abort uses it when available and
// falls back to Transport.Close for legacy custom transports.
//
// AbortableTransport 是用于立即终止活动连接的可选 transport 能力。
// Client.Abort 在可用时使用它，对旧式自定义 transport 则回退到 Transport.Close。
type AbortableTransport interface {
	Abort() error
}

// RawControlMessage wraps raw control protocol messages for passthrough.
//
// RawControlMessage 包装原始控制协议消息以供直通传递。
type RawControlMessage = shared.RawControlMessage

// RawMessage represents a CLI message with an unrecognized type field.
// Preserves the original type and all data for forward-compatible inspection.
//
// RawMessage 表示具有未识别 type 字段的 CLI 消息；
// 保留原始类型与全部数据以便进行前向兼容的检查。
type RawMessage = shared.RawMessage

// Usage represents token usage information from the Claude API.
// Present on AssistantMessage (per-turn) and ResultMessage (conversation total).
//
// Usage 表示来自 Claude API 的 token 使用信息。
// 出现在 AssistantMessage（每轮）与 ResultMessage（对话总计）上。
type Usage = shared.Usage

// StreamEvent represents a partial message update during streaming.
//
// StreamEvent 表示流式传输过程中的一次部分消息更新。
type StreamEvent = shared.StreamEvent

// Control protocol types for SDK-CLI bidirectional communication.
//
// 以下为用于 SDK↔CLI 双向通信的控制协议类型。

// SDKControlRequest represents a control request sent to the CLI.
//
// SDKControlRequest 表示发送给 CLI 的控制请求。
type SDKControlRequest = control.SDKControlRequest

// SDKControlResponse represents a control response received from the CLI.
//
// SDKControlResponse 表示从 CLI 接收的控制响应。
type SDKControlResponse = control.SDKControlResponse

// ControlResponse is the inner response structure.
//
// ControlResponse 是内层的响应结构。
type ControlResponse = control.Response

// InitializeRequest for control protocol handshake.
//
// InitializeRequest 用于控制协议握手。
type InitializeRequest = control.InitializeRequest

// InitializeResponse from CLI with supported capabilities.
//
// InitializeResponse 是 CLI 返回的、包含所支持能力的响应。
type InitializeResponse = control.InitializeResponse

// InterruptRequest to interrupt current operation via control protocol.
//
// InterruptRequest 用于通过控制协议中断当前操作。
type InterruptRequest = control.InterruptRequest

// SetPermissionModeRequest to change permission mode via control protocol.
//
// SetPermissionModeRequest 用于通过控制协议切换权限模式。
type SetPermissionModeRequest = control.SetPermissionModeRequest

// SetModelRequest to change AI model via control protocol.
//
// SetModelRequest 用于通过控制协议切换 AI 模型。
type SetModelRequest = control.SetModelRequest

// GetMcpStatusRequest to query MCP server status via control protocol.
//
// GetMcpStatusRequest 用于通过控制协议查询 MCP 服务器状态。
type GetMcpStatusRequest = control.GetMcpStatusRequest

// McpServerConnectionStatus represents the connection state of an MCP server.
//
// McpServerConnectionStatus 表示一个 MCP 服务器的连接状态。
type McpServerConnectionStatus = control.McpServerConnectionStatus

// Re-export MCP server connection status constants
//
// 重新导出 MCP 服务器连接状态常量。
const (
	McpServerConnectionStatusConnected = control.McpServerConnectionStatusConnected
	McpServerConnectionStatusFailed    = control.McpServerConnectionStatusFailed
	McpServerConnectionStatusNeedsAuth = control.McpServerConnectionStatusNeedsAuth
	McpServerConnectionStatusPending   = control.McpServerConnectionStatusPending
	McpServerConnectionStatusDisabled  = control.McpServerConnectionStatusDisabled
)

// Re-export MCP server config type constants for McpServerStatusConfig.Type.
//
// 重新导出 MCP 服务器配置类型常量，用于 McpServerStatusConfig.Type。
const (
	McpServerConfigTypeStdio    = control.McpServerConfigTypeStdio
	McpServerConfigTypeSSE      = control.McpServerConfigTypeSSE
	McpServerConfigTypeHTTP     = control.McpServerConfigTypeHTTP
	McpServerConfigTypeSDK      = control.McpServerConfigTypeSDK
	McpServerConfigTypeClaudeAI = control.McpServerConfigTypeClaudeAI
)

// McpServerInfo contains version information about a connected MCP server.
//
// McpServerInfo 包含已连接 MCP 服务器的版本信息。
type McpServerInfo = control.McpServerInfo

// McpToolAnnotations describes behavioral hints for an MCP tool.
//
// McpToolAnnotations 描述一个 MCP 工具的行为提示。
type McpToolAnnotations = control.McpToolAnnotations

// McpToolInfo describes a tool exposed by an MCP server.
//
// McpToolInfo 描述由 MCP 服务器暴露的一个工具。
type McpToolInfo = control.McpToolInfo

// McpServerStatusConfig covers all MCP server config variants, discriminated by Type.
//
// McpServerStatusConfig 涵盖所有 MCP 服务器配置变体，通过 Type 字段判别。
type McpServerStatusConfig = control.McpServerStatusConfig

// McpServerStatus contains the full status of a single MCP server.
//
// McpServerStatus 包含单个 MCP 服务器的完整状态。
type McpServerStatus = control.McpServerStatus

// McpStatusResponse is the response payload for a GetMcpStatus request.
//
// McpStatusResponse 是 GetMcpStatus 请求的响应载荷。
type McpStatusResponse = control.McpStatusResponse

// SlashCommand describes a CLI slash command suitable for input suggestions.
//
// SlashCommand 描述一个适用于输入建议的 CLI 斜杠命令。
type SlashCommand = control.SlashCommand

// ControlProtocol manages bidirectional control communication with CLI.
//
// ControlProtocol 管理与 CLI 的双向控制通信。
type ControlProtocol = control.Protocol

// Re-export control protocol subtype constants
//
// 重新导出控制协议子类型（subtype）常量。
const (
	// Control request subtypes
	// 控制请求子类型。
	SubtypeInterrupt         = control.SubtypeInterrupt
	SubtypeCanUseTool        = control.SubtypeCanUseTool
	SubtypeInitialize        = control.SubtypeInitialize
	SubtypeSetPermissionMode = control.SubtypeSetPermissionMode
	SubtypeSetModel          = control.SubtypeSetModel
	SubtypeHookCallback      = control.SubtypeHookCallback
	SubtypeMcpMessage        = control.SubtypeMcpMessage
	SubtypeGetMcpStatus      = control.SubtypeGetMcpStatus
	SubtypeRewindFiles       = control.SubtypeRewindFiles

	// Control response subtypes
	// 控制响应子类型。
	ResponseSubtypeSuccess = control.ResponseSubtypeSuccess
	ResponseSubtypeError   = control.ResponseSubtypeError
)
