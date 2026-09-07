package shared

import "context"

// StreamMessage represents messages sent to the CLI for streaming communication.
//
// StreamMessage 表示通过标准输入发送给 CLI 的流式协议消息。
// 它是一个「联合体」结构：既承载普通对话消息（Type 为 user/assistant 等，
// 内容放在 Message 字段），也承载控制协议消息（Type 为 control_request/
// control_response，内容放在 Request/Response 字段）。
// 除 Type 外全部字段都带 omitempty，因此同一结构体可以序列化出多种线格式。
type StreamMessage struct {
	// Type 是消息类型判别字段，决定其余字段中哪些有效。
	Type string `json:"type"`
	// Message 承载普通对话消息体，通常是 map[string]interface{}。
	Message interface{} `json:"message,omitempty"`
	// ParentToolUseID 非空表示该消息产生于子代理（Agent/Task 工具）内部。
	ParentToolUseID *string `json:"parent_tool_use_id,omitempty"`
	// SessionID 为会话标识，用于多会话场景下区分来源。
	SessionID string `json:"session_id,omitempty"`
	// RequestID 是控制请求的唯一标识，用于把响应与请求配对。
	RequestID string `json:"request_id,omitempty"`
	// Request 承载控制请求负载（Type == control_request 时有效）。
	Request map[string]interface{} `json:"request,omitempty"`
	// Response 承载控制响应负载（Type == control_response 时有效）。
	Response map[string]interface{} `json:"response,omitempty"`
}

// MessageIterator provides an iterator pattern for streaming messages.
//
// MessageIterator 为流式消息提供迭代器抽象。
// Next 阻塞直到下一条消息就绪、流结束或 ctx 被取消；
// 流正常结束时返回 io.EOF 语义的错误，调用方需要通过 Close 释放底层资源。
type MessageIterator interface {
	// Next 返回下一条消息；流结束或出错时返回非 nil error。
	Next(ctx context.Context) (Message, error)
	// Close 释放迭代器占用的资源，可安全重复调用。
	Close() error
}
