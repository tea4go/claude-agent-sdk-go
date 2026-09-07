package shared

import (
	"sync"
)

// StreamValidator tracks tool requests and results to detect incomplete streams.
//
// StreamValidator 跟踪工具调用请求与结果，用于检测不完整的消息流。
// 典型异常包括：工具被请求但从未收到结果、收到了无对应请求的工具结果、
// 以及流在没有 ResultMessage 的情况下就结束。所有方法均通过读写锁保证并发安全。
type StreamValidator struct {
	mu               sync.RWMutex
	toolsRequested   map[string]bool // 所有被请求的 tool_use ID 集合
	toolsReceived    map[string]bool // 所有已收到的 tool_result ID 集合
	pendingToolsSet  map[string]bool // 尚在等待结果的工具 ID 集合
	hasResultMessage bool            // 是否已见到 ResultMessage
	streamEnded      bool            // 流是否已结束
	issues           []StreamIssue   // 已发现的校验问题
}

// StreamIssue represents a validation issue found in the stream.
//
// StreamIssue 表示在消息流中发现的一个校验问题。
type StreamIssue struct {
	Type        string `json:"type"`                  // 问题类型，如 "missing_tool_result"、"extra_tool_result" 等
	Description string `json:"description"`           // 面向人的可读描述
	ToolUseID   string `json:"tool_use_id,omitempty"` // 相关的工具调用 ID（如适用）
}

// StreamStats provides statistics about the message stream.
//
// StreamStats 提供消息流的统计信息，用于诊断与监控。
type StreamStats struct {
	ToolsRequested int      `json:"tools_requested"` // 被请求的工具总数
	ToolsReceived  int      `json:"tools_received"`  // 已收到的工具结果总数
	PendingTools   []string `json:"pending_tools"`   // 仍在等待结果的工具 ID
	HasResult      bool     `json:"has_result"`      // 是否见到了 ResultMessage
	StreamEnded    bool     `json:"stream_ended"`    // 流是否已结束
}

// NewStreamValidator creates a new stream validator.
//
// NewStreamValidator 创建一个新的流校验器，并初始化内部集合。
func NewStreamValidator() *StreamValidator {
	return &StreamValidator{
		toolsRequested:  make(map[string]bool),
		toolsReceived:   make(map[string]bool),
		pendingToolsSet: make(map[string]bool),
		issues:          []StreamIssue{},
	}
}

// TrackMessage processes a message and updates validation state.
//
// TrackMessage 处理一条消息并更新校验状态：从 AssistantMessage 中提取工具调用请求，
// 从 UserMessage 中提取工具结果并匹配，并在发现无对应请求的结果时记录异常。
func (v *StreamValidator) TrackMessage(msg Message) {
	v.mu.Lock()
	defer v.mu.Unlock()

	switch m := msg.(type) {
	case *AssistantMessage:
		// 记录工具调用请求，并将其加入待完成集合
		for _, block := range m.Content {
			if toolUse, ok := block.(*ToolUseBlock); ok {
				v.toolsRequested[toolUse.ToolUseID] = true
				v.pendingToolsSet[toolUse.ToolUseID] = true
			}
		}

	case *UserMessage:
		// 记录工具结果
		if blocks, ok := m.Content.([]ContentBlock); ok {
			for _, block := range blocks {
				if toolResult, ok := block.(*ToolResultBlock); ok {
					v.toolsReceived[toolResult.ToolUseID] = true
					delete(v.pendingToolsSet, toolResult.ToolUseID)

					// 检查多余的工具结果（没有对应请求的结果）
					if !v.toolsRequested[toolResult.ToolUseID] {
						v.issues = append(v.issues, StreamIssue{
							Type:        "extra_tool_result",
							Description: "Received tool result without corresponding tool request",
							ToolUseID:   toolResult.ToolUseID,
						})
					}
				}
			}
		}

	case *ResultMessage:
		v.hasResultMessage = true
	}
}

// MarkStreamEnd marks the stream as ended and performs final validation.
//
// MarkStreamEnd 将流标记为已结束并执行最终校验：把仍处于待完成集合的工具
// 记为缺失结果，并在有工具请求但未见 ResultMessage 时记录异常。
func (v *StreamValidator) MarkStreamEnd() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.streamEnded = true

	// 检查缺失的工具结果
	for toolID := range v.pendingToolsSet {
		v.issues = append(v.issues, StreamIssue{
			Type:        "missing_tool_result",
			Description: "Tool was requested but result was never received",
			ToolUseID:   toolID,
		})
	}

	// 检查缺失的 ResultMessage
	if len(v.toolsRequested) > 0 && !v.hasResultMessage {
		v.issues = append(v.issues, StreamIssue{
			Type:        "missing_result_message",
			Description: "Stream ended without result message",
		})
	}
}

// GetIssues returns all validation issues found.
//
// GetIssues 返回所有已发现的校验问题（返回副本，防止外部修改内部状态）。
func (v *StreamValidator) GetIssues() []StreamIssue {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// 返回副本以防止外部修改
	issues := make([]StreamIssue, len(v.issues))
	copy(issues, v.issues)
	return issues
}

// GetStats returns current stream statistics.
//
// GetStats 返回当前流的统计快照（请求数、接收数、待完成工具列表等）。
func (v *StreamValidator) GetStats() StreamStats {
	v.mu.RLock()
	defer v.mu.RUnlock()

	pendingTools := make([]string, 0, len(v.pendingToolsSet))
	for toolID := range v.pendingToolsSet {
		pendingTools = append(pendingTools, toolID)
	}

	return StreamStats{
		ToolsRequested: len(v.toolsRequested),
		ToolsReceived:  len(v.toolsReceived),
		PendingTools:   pendingTools,
		HasResult:      v.hasResultMessage,
		StreamEnded:    v.streamEnded,
	}
}

// HasIssues returns whether any validation issues were found.
//
// HasIssues 返回是否发现了任何校验问题。
func (v *StreamValidator) HasIssues() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.issues) > 0
}
