package shared

import (
	"encoding/json"
)

// Message type constants
//
// 消息类型常量：对应 CLI 线格式中的 "type" 字段，用于判别消息具体类型。
const (
	MessageTypeUser      = "user"
	MessageTypeAssistant = "assistant"
	MessageTypeSystem    = "system"
	MessageTypeResult    = "result"

	// Control protocol message types
	// 控制协议消息类型（请求/响应）
	MessageTypeControlRequest  = "control_request"
	MessageTypeControlResponse = "control_response"

	// Partial message streaming type
	// 部分消息流式推送类型
	MessageTypeStreamEvent = "stream_event"

	// Session heartbeat carrying rate-limit window state. Emitted on
	// essentially every CLI session — even when nothing is constrained.
	// 携带限流窗口状态的会话心跳，几乎每个 CLI 会话都会发送（即使未受限）。
	MessageTypeRateLimitEvent = "rate_limit_event"
)

// Content block type constants
//
// 内容块类型常量：用于判别一条消息内部各内容块的类型。
const (
	ContentBlockTypeText       = "text"
	ContentBlockTypeThinking   = "thinking"
	ContentBlockTypeToolUse    = "tool_use"
	ContentBlockTypeToolResult = "tool_result"
)

// AssistantMessageError represents error types in assistant messages.
//
// AssistantMessageError 表示助手消息中可能携带的错误类型。
type AssistantMessageError string

// AssistantMessageError constants for error type identification.
//
// AssistantMessageError 常量，用于识别具体错误类型（与 Python SDK 保持一致）。
const (
	AssistantMessageErrorAuthFailed     AssistantMessageError = "authentication_failed"
	AssistantMessageErrorBilling        AssistantMessageError = "billing_error"
	AssistantMessageErrorRateLimit      AssistantMessageError = "rate_limit"
	AssistantMessageErrorInvalidRequest AssistantMessageError = "invalid_request"
	AssistantMessageErrorServer         AssistantMessageError = "server_error"
	AssistantMessageErrorUnknown        AssistantMessageError = "unknown"
)

// Stop reason constants for AssistantMessage.StopReason.
// These match the Anthropic API stop_reason values.
//
// AssistantMessage.StopReason 的停止原因常量，与 Anthropic API 的 stop_reason 取值一致。
const (
	StopReasonEndTurn      = "end_turn"
	StopReasonToolUse      = "tool_use"
	StopReasonStopSequence = "stop_sequence"
	StopReasonMaxTokens    = "max_tokens"
)

// Usage represents token usage information from the Claude API.
// Present on AssistantMessage (per-turn) and ResultMessage (conversation total).
//
// Usage 表示来自 Claude API 的 token 用量信息：
// 在 AssistantMessage 上为单轮用量，在 ResultMessage 上为整个对话的累计用量。
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// Message represents any message type in the Claude Code protocol.
//
// Message 是 Claude Code 协议中任意消息类型的统一接口，Type() 返回其类型字符串。
type Message interface {
	Type() string
}

// ContentBlock represents any content block within a message.
//
// ContentBlock 表示消息内部的任意内容块，BlockType() 返回其块类型。
type ContentBlock interface {
	BlockType() string
}

// UserMessage represents a message from the user.
//
// UserMessage 表示来自用户的消息。Content 可为纯文本字符串，也可为 []ContentBlock；
// ToolUseResult 携带工具调用的富元数据（如文件路径、结构化补丁、diff 等）。
type UserMessage struct {
	MessageType     string         `json:"type"`
	Content         interface{}    `json:"content"` // string 或 []ContentBlock
	UUID            *string        `json:"uuid,omitempty"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
	ToolUseResult   map[string]any `json:"tool_use_result,omitempty"`
}

// Type returns the message type for UserMessage.
//
// Type 返回 UserMessage 的消息类型。
func (m *UserMessage) Type() string {
	return MessageTypeUser
}

// GetUUID returns the UUID or empty string if nil.
//
// GetUUID 返回消息 UUID，为 nil 时返回空字符串。
func (m *UserMessage) GetUUID() string {
	if m.UUID != nil {
		return *m.UUID
	}
	return ""
}

// GetParentToolUseID returns the parent tool use ID or empty string if nil.
//
// GetParentToolUseID 返回父工具调用 ID，为 nil 时返回空字符串。
func (m *UserMessage) GetParentToolUseID() string {
	if m.ParentToolUseID != nil {
		return *m.ParentToolUseID
	}
	return ""
}

// GetToolUseResult returns the tool use result metadata or nil if not present.
//
// GetToolUseResult 返回工具调用结果元数据，不存在时返回 nil。
func (m *UserMessage) GetToolUseResult() map[string]any {
	return m.ToolUseResult
}

// HasToolUseResult returns true if tool use result metadata is present and non-empty.
//
// HasToolUseResult 在工具调用结果元数据存在且非空时返回 true（访问前先用它判断）。
func (m *UserMessage) HasToolUseResult() bool {
	return len(m.ToolUseResult) > 0
}

// MarshalJSON implements custom JSON marshaling for UserMessage
//
// MarshalJSON 为 UserMessage 实现自定义 JSON 序列化，确保输出固定的 type 字段。
func (m *UserMessage) MarshalJSON() ([]byte, error) {
	type userMessage UserMessage
	temp := struct {
		Type string `json:"type"`
		*userMessage
	}{
		Type:        MessageTypeUser,
		userMessage: (*userMessage)(m),
	}
	return json.Marshal(temp)
}

// AssistantMessage represents a message from the assistant.
//
// AssistantMessage 表示来自助手（Claude）的消息。Content 为多个内容块（文本/
// 思考/工具调用等）；Error 非空表示本轮出错；ParentToolUseID 非空表示该消息
// 产生于子代理内部。
type AssistantMessage struct {
	MessageType     string                 `json:"type"`
	ID              *string                `json:"id,omitempty"`
	Content         []ContentBlock         `json:"content"`
	Model           string                 `json:"model"`
	Usage           *Usage                 `json:"usage,omitempty"`
	StopReason      *string                `json:"stop_reason,omitempty"`
	Error           *AssistantMessageError `json:"error,omitempty"`
	ParentToolUseID *string                `json:"parent_tool_use_id,omitempty"`
}

// Type returns the message type for AssistantMessage.
//
// Type 返回 AssistantMessage 的消息类型。
func (m *AssistantMessage) Type() string {
	return MessageTypeAssistant
}

// HasUsage returns true if per-turn token usage is available.
//
// HasUsage 在存在单轮 token 用量时返回 true。
func (m *AssistantMessage) HasUsage() bool {
	return m.Usage != nil
}

// GetID returns the Claude message ID or empty string if nil.
//
// GetID 返回 Claude 消息 ID，为 nil 时返回空字符串。
func (m *AssistantMessage) GetID() string {
	if m.ID != nil {
		return *m.ID
	}
	return ""
}

// GetStopReason returns the stop reason or empty string if nil.
//
// GetStopReason 返回停止原因，为 nil 时返回空字符串。
func (m *AssistantMessage) GetStopReason() string {
	if m.StopReason != nil {
		return *m.StopReason
	}
	return ""
}

// IsToolUse returns true if the message stopped because Claude wants to use a tool.
//
// IsToolUse 在消息因 Claude 需要调用工具而停止时返回 true。
func (m *AssistantMessage) IsToolUse() bool {
	return m.StopReason != nil && *m.StopReason == StopReasonToolUse
}

// GetParentToolUseID returns the parent tool use ID or empty string if nil.
// On assistant messages produced inside a subagent (Agent/Task tool), this
// identifies the orchestrator tool_use_id that spawned the subagent.
//
// GetParentToolUseID 返回父工具调用 ID，为 nil 时返回空字符串。对于在子代理
// （Agent/Task 工具）内部产生的助手消息，它标识创建该子代理的编排器 tool_use_id。
func (m *AssistantMessage) GetParentToolUseID() string {
	if m.ParentToolUseID != nil {
		return *m.ParentToolUseID
	}
	return ""
}

// HasError returns true if the message contains an error.
//
// HasError 在消息包含错误时返回 true。
func (m *AssistantMessage) HasError() bool {
	return m.Error != nil
}

// GetError returns the error type or empty string if nil.
//
// GetError 返回错误类型，为 nil 时返回空字符串。
func (m *AssistantMessage) GetError() AssistantMessageError {
	if m.Error != nil {
		return *m.Error
	}
	return ""
}

// IsRateLimited returns true if the error is a rate limit error.
//
// IsRateLimited 在错误为限流错误时返回 true。
func (m *AssistantMessage) IsRateLimited() bool {
	return m.Error != nil && *m.Error == AssistantMessageErrorRateLimit
}

// MarshalJSON implements custom JSON marshaling for AssistantMessage
//
// MarshalJSON 为 AssistantMessage 实现自定义 JSON 序列化，确保输出固定的 type 字段。
func (m *AssistantMessage) MarshalJSON() ([]byte, error) {
	type assistantMessage AssistantMessage
	temp := struct {
		Type string `json:"type"`
		*assistantMessage
	}{
		Type:             MessageTypeAssistant,
		assistantMessage: (*assistantMessage)(m),
	}
	return json.Marshal(temp)
}

// SystemMessage represents a system message.
//
// SystemMessage 表示系统消息。Subtype 区分具体子类型；Data 以 map 形式完整
// 保留原始数据（不参与自动 JSON 字段映射）。
type SystemMessage struct {
	MessageType string         `json:"type"`
	Subtype     string         `json:"subtype"`
	Data        map[string]any `json:"-"` // 保留全部原始数据
}

// Type returns the message type for SystemMessage.
//
// Type 返回 SystemMessage 的消息类型。
func (m *SystemMessage) Type() string {
	return MessageTypeSystem
}

// MarshalJSON implements custom JSON marshaling for SystemMessage
//
// MarshalJSON 为 SystemMessage 实现自定义 JSON 序列化：将 Data 展开并补上 type/subtype。
func (m *SystemMessage) MarshalJSON() ([]byte, error) {
	data := make(map[string]any)
	for k, v := range m.Data {
		data[k] = v
	}
	data["type"] = MessageTypeSystem
	data["subtype"] = m.Subtype
	return json.Marshal(data)
}

// ResultMessage represents the final result of a conversation turn.
//
// ResultMessage 表示一个对话轮次的最终结果，携带耗时、轮数、会话 ID、
// 累计费用与用量、最终文本及结构化输出等；IsError 标识本轮是否以错误告终。
type ResultMessage struct {
	MessageType      string   `json:"type"`
	Subtype          string   `json:"subtype"`
	DurationMs       int      `json:"duration_ms"`
	DurationAPIMs    int      `json:"duration_api_ms"`
	IsError          bool     `json:"is_error"`
	Errors           []string `json:"errors,omitempty"`
	NumTurns         int      `json:"num_turns"`
	SessionID        string   `json:"session_id"`
	TotalCostUSD     *float64 `json:"total_cost_usd,omitempty"`
	Usage            *Usage   `json:"usage,omitempty"`
	Result           *string  `json:"result,omitempty"`
	StructuredOutput any      `json:"structured_output,omitempty"`
}

// Type returns the message type for ResultMessage.
//
// Type 返回 ResultMessage 的消息类型。
func (m *ResultMessage) Type() string {
	return MessageTypeResult
}

// HasUsage returns true if conversation-level token usage is available.
//
// HasUsage 在存在对话级别的 token 用量时返回 true。
func (m *ResultMessage) HasUsage() bool {
	return m.Usage != nil
}

// MarshalJSON implements custom JSON marshaling for ResultMessage
//
// MarshalJSON 为 ResultMessage 实现自定义 JSON 序列化，确保输出固定的 type 字段。
func (m *ResultMessage) MarshalJSON() ([]byte, error) {
	type resultMessage ResultMessage
	temp := struct {
		Type string `json:"type"`
		*resultMessage
	}{
		Type:          MessageTypeResult,
		resultMessage: (*resultMessage)(m),
	}
	return json.Marshal(temp)
}

// TextBlock represents text content.
//
// TextBlock 表示纯文本内容块。
type TextBlock struct {
	MessageType string `json:"type"`
	Text        string `json:"text"`
}

// BlockType returns the content block type for TextBlock.
//
// BlockType 返回 TextBlock 的内容块类型。
func (b *TextBlock) BlockType() string {
	return ContentBlockTypeText
}

// ThinkingBlock represents thinking content with signature.
//
// ThinkingBlock 表示带签名的思考内容块。
type ThinkingBlock struct {
	MessageType string `json:"type"`
	Thinking    string `json:"thinking"`
	Signature   string `json:"signature"`
}

// BlockType returns the content block type for ThinkingBlock.
//
// BlockType 返回 ThinkingBlock 的内容块类型。
func (b *ThinkingBlock) BlockType() string {
	return ContentBlockTypeThinking
}

// ToolUseBlock represents a tool use request.
//
// ToolUseBlock 表示一次工具调用请求；Name 为工具名，Input 为调用参数。
type ToolUseBlock struct {
	MessageType string         `json:"type"`
	ToolUseID   string         `json:"tool_use_id"`
	Name        string         `json:"name"`
	Input       map[string]any `json:"input"`
}

// BlockType returns the content block type for ToolUseBlock.
//
// BlockType 返回 ToolUseBlock 的内容块类型。
func (b *ToolUseBlock) BlockType() string {
	return ContentBlockTypeToolUse
}

// ToolResultBlock represents the result of a tool use.
//
// ToolResultBlock 表示一次工具调用的结果；Content 可为字符串或结构化数据，
// IsError 非空且为 true 时表示工具执行失败。
type ToolResultBlock struct {
	MessageType string      `json:"type"`
	ToolUseID   string      `json:"tool_use_id"`
	Content     interface{} `json:"content"` // 字符串或结构化数据
	IsError     *bool       `json:"is_error,omitempty"`
}

// BlockType returns the content block type for ToolResultBlock.
//
// BlockType 返回 ToolResultBlock 的内容块类型。
func (b *ToolResultBlock) BlockType() string {
	return ContentBlockTypeToolResult
}

// RawControlMessage wraps raw control protocol messages for passthrough to the control handler.
// Control messages are not parsed into typed structs by the parser - they are routed directly
// to the control protocol handler which performs its own parsing.
//
// RawControlMessage 封装原始控制协议消息，用于直接透传给控制处理器。
// 解析器不会把控制消息解析为强类型结构，而是直接路由给控制协议处理器自行解析。
type RawControlMessage struct {
	MessageType string
	Data        map[string]any
}

// Type returns the message type for RawControlMessage.
//
// Type 返回 RawControlMessage 的消息类型。
func (m *RawControlMessage) Type() string {
	return m.MessageType
}

// RawMessage represents a CLI message with an unrecognized type field.
// It preserves the original type string and all data so consumers can inspect
// future CLI message types without SDK upgrades.
//
// RawMessage 表示具有未知 type 字段的 CLI 消息。它保留原始 type 字符串与全部
// 数据，使调用方无需升级 SDK 即可检视未来新增的 CLI 消息类型。
type RawMessage struct {
	MessageType string         `json:"type"`
	Data        map[string]any `json:"-"`
}

// Type returns the original message type string.
//
// Type 返回原始的消息类型字符串。
func (m *RawMessage) Type() string {
	return m.MessageType
}

// MarshalJSON implements custom JSON marshaling for RawMessage.
//
// MarshalJSON 为 RawMessage 实现自定义 JSON 序列化：展开 Data 并补上原始 type。
func (m *RawMessage) MarshalJSON() ([]byte, error) {
	data := make(map[string]any)
	for k, v := range m.Data {
		data[k] = v
	}
	data["type"] = m.MessageType
	return json.Marshal(data)
}

// Rate-limit window status constants. Status carries one of these strings;
// "allowed" means the session is fine and the message is informational only.
//
// 限流窗口状态常量。Status 取其中之一；"allowed" 表示会话正常，该消息仅为告知性质。
const (
	RateLimitStatusAllowed = "allowed"
)

// RateLimitInfo carries the rate-limit window state from a rate_limit_event
// message. The Claude CLI emits one of these as a per-session heartbeat
// regardless of whether the user is actually constrained — check Status to
// decide if action is needed.
//
// RateLimitInfo 携带来自 rate_limit_event 消息的限流窗口状态。无论用户是否真正
// 受限，Claude CLI 都会作为会话心跳发送一条——通过 Status 判断是否需要采取行动。
type RateLimitInfo struct {
	Status          string `json:"status"`
	ResetsAt        int64  `json:"resetsAt"`
	RateLimitType   string `json:"rateLimitType"`
	OverageStatus   string `json:"overageStatus,omitempty"`
	OverageResetsAt int64  `json:"overageResetsAt,omitempty"`
	IsUsingOverage  bool   `json:"isUsingOverage,omitempty"`
}

// RateLimitEventMessage is a session heartbeat from the CLI announcing the
// current rate-limit window. Emitted on essentially every session even when
// nothing is constrained — most consumers can simply ignore the message
// unless RateLimitInfo.Status differs from RateLimitStatusAllowed.
//
// See https://github.com/tea4go/claude-agent-sdk-go/issues/126.
//
// RateLimitEventMessage 是 CLI 发出的会话心跳，用于告知当前限流窗口。几乎每个
// 会话都会发送（即使未受限），除非 RateLimitInfo.Status 不等于 RateLimitStatusAllowed，
// 否则大多数调用方可直接忽略。
type RateLimitEventMessage struct {
	MessageType   string        `json:"type"`
	RateLimitInfo RateLimitInfo `json:"rate_limit_info"`
	UUID          string        `json:"uuid,omitempty"`
	SessionID     string        `json:"session_id,omitempty"`
}

// Type returns the message type for RateLimitEventMessage.
//
// Type 返回 RateLimitEventMessage 的消息类型。
func (m *RateLimitEventMessage) Type() string {
	return MessageTypeRateLimitEvent
}

// IsAllowed returns true when the rate-limit window is healthy — i.e. the
// heartbeat is informational and the consumer can keep going.
//
// IsAllowed 在限流窗口健康时返回 true——即心跳仅为告知性质，调用方可继续。
func (m *RateLimitEventMessage) IsAllowed() bool {
	return m.RateLimitInfo.Status == RateLimitStatusAllowed
}

// Stream event type constants for Event["type"] discrimination.
// Use these when type-switching on StreamEvent.Event to handle different event types.
//
// 用于 Event["type"] 判别的流事件类型常量；在对 StreamEvent.Event 做类型分支时使用。
const (
	StreamEventTypeContentBlockStart = "content_block_start"
	StreamEventTypeContentBlockDelta = "content_block_delta"
	StreamEventTypeContentBlockStop  = "content_block_stop"
	StreamEventTypeMessageStart      = "message_start"
	StreamEventTypeMessageDelta      = "message_delta"
	StreamEventTypeMessageStop       = "message_stop"
)

// StreamEvent represents a partial message update during streaming.
// Emitted when IncludePartialMessages is enabled in Options.
//
// The Event field contains varying structure depending on event type:
//   - content_block_start: {"type": "content_block_start", "index": <int>, "content_block": {...}}
//   - content_block_delta: {"type": "content_block_delta", "index": <int>, "delta": {...}}
//   - content_block_stop: {"type": "content_block_stop", "index": <int>}
//   - message_start: {"type": "message_start", "message": {...}}
//   - message_delta: {"type": "message_delta", "delta": {...}, "usage": {...}}
//   - message_stop: {"type": "message_stop"}
//
// Consumer code should type-switch on Event["type"] to handle different event types:
//
//	switch event.Event["type"] {
//	case shared.StreamEventTypeContentBlockDelta:
//	    // Handle content delta
//	case shared.StreamEventTypeMessageStop:
//	    // Handle message completion
//	}
type StreamEvent struct {
	UUID            string         `json:"uuid"`
	SessionID       string         `json:"session_id"`
	Event           map[string]any `json:"event"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

// Type returns the message type for StreamEvent.
//
// Type 返回 StreamEvent 的消息类型。
func (m *StreamEvent) Type() string {
	return MessageTypeStreamEvent
}
