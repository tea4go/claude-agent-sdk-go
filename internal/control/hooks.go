// Package control hook callback handling and registration.
// This file contains hook lifecycle event processing for PreToolUse, PostToolUse, etc.
//
// 本文件处理钩子回调的调用与注册，
// 包含 PreToolUse、PostToolUse 等钩子生命周期事件的处理。
package control

import (
	"context"
	"encoding/json"
	"fmt"
)

// handleHookCallbackRequest processes a hook callback request from CLI.
// Follows the same pattern as handleCanUseToolRequest with panic recovery.
//
// handleHookCallbackRequest 处理来自 CLI 的钩子回调请求，
// 与 handleCanUseToolRequest 采用相同模式并带 panic 恢复。
func (p *Protocol) handleHookCallbackRequest(ctx context.Context, requestID string, request map[string]any) error {
	// Parse callback ID
	// 解析回调 ID
	callbackID, _ := request["callback_id"].(string)
	if callbackID == "" {
		return p.sendErrorResponse(ctx, requestID, "missing callback_id")
	}

	// Parse hook event name from input
	// 从输入中解析钩子事件名
	inputData, _ := request["input"].(map[string]any)
	if inputData == nil {
		inputData = make(map[string]any)
	}

	eventName, _ := inputData["hook_event_name"].(string)
	event := HookEvent(eventName)

	// Parse input based on event type
	// 根据事件类型解析输入
	input := p.parseHookInput(event, inputData)

	// Parse tool_use_id if present
	// 若存在则解析 tool_use_id
	var toolUseID *string
	if id, ok := request["tool_use_id"].(string); ok {
		toolUseID = &id
	}

	// Get callback (thread-safe read)
	// 获取回调（线程安全读取）
	p.hookCallbacksMu.RLock()
	callback, exists := p.hookCallbacks[callbackID]
	p.hookCallbacksMu.RUnlock()

	if !exists {
		return p.sendErrorResponse(ctx, requestID, fmt.Sprintf("callback not found: %s", callbackID))
	}

	// Create hook context
	// 创建钩子上下文
	hookCtx := HookContext{Signal: ctx}

	// Invoke callback with panic recovery (matches permission callback pattern)
	// 调用回调并做 panic 恢复（与权限回调模式一致）
	var result HookJSONOutput
	var callbackErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				callbackErr = fmt.Errorf("hook callback panicked: %v", r)
			}
		}()
		result, callbackErr = callback(ctx, input, toolUseID, hookCtx)
	}()

	if callbackErr != nil {
		return p.sendErrorResponse(ctx, requestID, fmt.Sprintf("callback error: %v", callbackErr))
	}

	return p.sendHookResponse(ctx, requestID, result)
}

// parseHookInput creates the appropriate typed input based on event type.
// Returns the strongly-typed input struct for the callback.
//
// parseHookInput 根据事件类型创建对应的强类型输入结构，供回调使用。
func (p *Protocol) parseHookInput(event HookEvent, inputData map[string]any) any {
	// Parse base fields
	// 解析基础字段
	base := BaseHookInput{
		SessionID:      getString(inputData, "session_id"),
		TranscriptPath: getString(inputData, "transcript_path"),
		Cwd:            getString(inputData, "cwd"),
		PermissionMode: getString(inputData, "permission_mode"),
	}

	switch event {
	case HookEventPreToolUse:
		return &PreToolUseHookInput{
			BaseHookInput: base,
			HookEventName: "PreToolUse",
			ToolName:      getString(inputData, "tool_name"),
			ToolInput:     getMap(inputData, "tool_input"),
			ToolUseID:     getString(inputData, "tool_use_id"),
		}
	case HookEventPostToolUse:
		return &PostToolUseHookInput{
			BaseHookInput: base,
			HookEventName: "PostToolUse",
			ToolName:      getString(inputData, "tool_name"),
			ToolInput:     getMap(inputData, "tool_input"),
			ToolResponse:  inputData["tool_response"],
			ToolUseID:     getString(inputData, "tool_use_id"),
		}
	case HookEventPostToolUseFailure:
		return &PostToolUseFailureHookInput{
			BaseHookInput: base,
			HookEventName: "PostToolUseFailure",
			ToolName:      getString(inputData, "tool_name"),
			ToolInput:     getMap(inputData, "tool_input"),
			ToolUseID:     getString(inputData, "tool_use_id"),
			Error:         getString(inputData, "error"),
			IsInterrupt:   getBoolPtr(inputData, "is_interrupt"),
		}
	case HookEventUserPromptSubmit:
		return &UserPromptSubmitHookInput{
			BaseHookInput: base,
			HookEventName: "UserPromptSubmit",
			Prompt:        getString(inputData, "prompt"),
		}
	case HookEventStop:
		return &StopHookInput{
			BaseHookInput:  base,
			HookEventName:  "Stop",
			StopHookActive: getBool(inputData, "stop_hook_active"),
		}
	case HookEventSubagentStop:
		return &SubagentStopHookInput{
			BaseHookInput:       base,
			HookEventName:       "SubagentStop",
			StopHookActive:      getBool(inputData, "stop_hook_active"),
			AgentID:             getString(inputData, "agent_id"),
			AgentTranscriptPath: getString(inputData, "agent_transcript_path"),
			AgentType:           getString(inputData, "agent_type"),
		}
	case HookEventPreCompact:
		return &PreCompactHookInput{
			BaseHookInput:      base,
			HookEventName:      "PreCompact",
			Trigger:            getString(inputData, "trigger"),
			CustomInstructions: getStringPtr(inputData, "custom_instructions"),
		}
	case HookEventNotification:
		return &NotificationHookInput{
			BaseHookInput:    base,
			HookEventName:    "Notification",
			Message:          getString(inputData, "message"),
			Title:            getStringPtr(inputData, "title"),
			NotificationType: getString(inputData, "notification_type"),
		}
	case HookEventSubagentStart:
		return &SubagentStartHookInput{
			BaseHookInput: base,
			HookEventName: "SubagentStart",
			AgentID:       getString(inputData, "agent_id"),
			AgentType:     getString(inputData, "agent_type"),
		}
	case HookEventPermissionRequest:
		return &PermissionRequestHookInput{
			BaseHookInput:         base,
			HookEventName:         "PermissionRequest",
			ToolName:              getString(inputData, "tool_name"),
			ToolInput:             getMap(inputData, "tool_input"),
			PermissionSuggestions: getAnySlice(inputData, "permission_suggestions"),
		}
	default:
		// Forward compatibility - return raw input for unknown events
		// 向前兼容——对未知事件返回原始输入
		return inputData
	}
}

// sendHookResponse sends a hook callback response back to CLI.
//
// sendHookResponse 将钩子回调的响应回送给 CLI。
func (p *Protocol) sendHookResponse(ctx context.Context, requestID string, result HookJSONOutput) error {
	// Build response data from HookJSONOutput
	// 从 HookJSONOutput 构建响应数据
	responseData := make(map[string]any)

	if result.Continue != nil {
		responseData["continue"] = *result.Continue
	}
	if result.SuppressOutput != nil {
		responseData["suppressOutput"] = *result.SuppressOutput
	}
	if result.StopReason != nil {
		responseData["stopReason"] = *result.StopReason
	}
	if result.Decision != nil {
		responseData["decision"] = *result.Decision
	}
	if result.SystemMessage != nil {
		responseData["systemMessage"] = *result.SystemMessage
	}
	if result.Reason != nil {
		responseData["reason"] = *result.Reason
	}
	if result.HookSpecificOutput != nil {
		responseData["hookSpecificOutput"] = result.HookSpecificOutput
	}

	response := SDKControlResponse{
		Type: MessageTypeControlResponse,
		Response: Response{
			Subtype:   ResponseSubtypeSuccess,
			RequestID: requestID,
			Response:  responseData,
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal hook response: %w", err)
	}

	return p.transport.Write(ctx, append(data, '\n'))
}

// generateHookRegistrations creates hook registrations for initialization.
// This builds the hooks config to send to CLI during initialize.
//
// generateHookRegistrations 为初始化创建钩子注册项，构建在 initialize 时发送给 CLI 的钩子配置。
func (p *Protocol) generateHookRegistrations() []HookRegistration {
	var registrations []HookRegistration

	if p.hooks == nil {
		return registrations
	}

	// Initialize callback map if needed
	// 必要时初始化回调 map
	p.hookCallbacksMu.Lock()
	if p.hookCallbacks == nil {
		p.hookCallbacks = make(map[string]HookCallback)
	}

	for _, matchers := range p.hooks {
		for _, matcher := range matchers {
			for _, callback := range matcher.Hooks {
				// Generate callback ID matching Python SDK format
				// 生成与 Python SDK 格式一致的回调 ID
				callbackID := fmt.Sprintf("hook_%d", p.nextHookCallback)
				p.nextHookCallback++

				// Store callback for later lookup
				// 存储回调以便后续查找
				p.hookCallbacks[callbackID] = callback

				registrations = append(registrations, HookRegistration{
					CallbackID: callbackID,
					Matcher:    matcher.Matcher,
					Timeout:    matcher.Timeout,
				})
			}
		}
	}
	p.hookCallbacksMu.Unlock()

	return registrations
}

// buildHooksConfig creates the hooks config for the initialize request.
// Format: {"PreToolUse": [{"matcher": "Bash", "hookCallbackIds": ["hook_0"]}], ...}
//
// buildHooksConfig 为 initialize 请求构建钩子配置。
// 格式：{"PreToolUse": [{"matcher": "Bash", "hookCallbackIds": ["hook_0"]}], ...}
func (p *Protocol) buildHooksConfig() map[string][]HookMatcherConfig {
	if p.hooks == nil {
		return nil
	}

	config := make(map[string][]HookMatcherConfig)

	// Initialize callback map if needed
	// 必要时初始化回调 map
	p.hookCallbacksMu.Lock()
	if p.hookCallbacks == nil {
		p.hookCallbacks = make(map[string]HookCallback)
	}

	for event, matchers := range p.hooks {
		eventName := string(event)
		var matcherConfigs []HookMatcherConfig

		for _, matcher := range matchers {
			// Generate callback IDs for each callback in this matcher
			var callbackIDs []string
			for _, callback := range matcher.Hooks {
				callbackID := fmt.Sprintf("hook_%d", p.nextHookCallback)
				p.nextHookCallback++

				// Store callback for later lookup
				p.hookCallbacks[callbackID] = callback
				callbackIDs = append(callbackIDs, callbackID)
			}

			matcherConfigs = append(matcherConfigs, HookMatcherConfig{
				Matcher:         matcher.Matcher,
				HookCallbackIDs: callbackIDs,
				Timeout:         matcher.Timeout,
			})
		}

		if len(matcherConfigs) > 0 {
			config[eventName] = matcherConfigs
		}
	}
	p.hookCallbacksMu.Unlock()

	return config
}

// Helper functions for parsing hook input fields
//
// 用于解析钩子输入字段的辅助函数

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStringPtr(m map[string]any, key string) *string {
	if v, ok := m[key].(string); ok {
		return &v
	}
	return nil
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getBoolPtr(m map[string]any, key string) *bool {
	if v, ok := m[key].(bool); ok {
		return &v
	}
	return nil
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return make(map[string]any)
}

// getAnySlice returns m[key] as a []any if present, or nil if absent or wrong type.
// Returns nil (not empty slice) to preserve Python's NotRequired absent-state semantics.
//
// getAnySlice 在 m[key] 存在时将其作为 []any 返回，缺失或类型不符时返回 nil。
// 返回 nil（而非空切片）以保留 Python NotRequired 的“字段缺失”语义。
func getAnySlice(m map[string]any, key string) []any {
	if v, ok := m[key].([]any); ok {
		return v
	}
	return nil
}
