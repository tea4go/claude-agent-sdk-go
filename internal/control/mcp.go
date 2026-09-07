// Package control MCP message routing for SDK MCP servers.
// This file handles JSONRPC method dispatch for tools/list, tools/call, etc.
//
// 本文件处理 SDK MCP 服务器的 MCP 消息路由，
// 负责 tools/list、tools/call 等 JSONRPC 方法的分发。
package control

import (
	"context"
	"encoding/json"
	"fmt"
)

// handleMcpMessageRequest routes MCP JSONRPC messages to SDK servers.
// Follows handleCanUseToolRequest pattern with panic recovery.
//
// handleMcpMessageRequest 将 MCP JSONRPC 消息路由到 SDK 服务器，
// 遵循 handleCanUseToolRequest 模式并带 panic 恢复。
func (p *Protocol) handleMcpMessageRequest(ctx context.Context, requestID string, request map[string]any) error {
	serverName := getString(request, "server_name")
	if serverName == "" {
		return p.sendErrorResponse(ctx, requestID, "missing server_name")
	}

	message, _ := request["message"].(map[string]any)
	if message == nil {
		return p.sendErrorResponse(ctx, requestID, "missing message")
	}

	// Thread-safe server lookup
	// 线程安全地查找服务器
	p.mu.Lock()
	server, exists := p.sdkMcpServers[serverName]
	p.mu.Unlock()

	if !exists {
		return p.sendMcpErrorResponse(ctx, requestID, message, -32601,
			fmt.Sprintf("server '%s' not found", serverName))
	}

	// Route JSONRPC method with panic recovery
	// 路由 JSONRPC 方法并做 panic 恢复
	var mcpResponse map[string]any
	var routeErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				routeErr = fmt.Errorf("MCP handler panicked: %v", r)
			}
		}()
		mcpResponse, routeErr = p.routeMcpMethod(ctx, server, message)
	}()

	if routeErr != nil {
		return p.sendMcpErrorResponse(ctx, requestID, message, -32603, routeErr.Error())
	}

	return p.sendMcpResponse(ctx, requestID, mcpResponse)
}

// routeMcpMethod dispatches JSONRPC methods to server handlers.
//
// routeMcpMethod 将 JSONRPC 方法分发到对应的服务器处理器。
func (p *Protocol) routeMcpMethod(ctx context.Context, server McpServer, msg map[string]any) (map[string]any, error) {
	method := getString(msg, "method")
	params, _ := msg["params"].(map[string]any)
	msgID := msg["id"]

	switch method {
	case "initialize":
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      msgID,
			"result": map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo": map[string]any{
					"name":    server.Name(),
					"version": server.Version(),
				},
			},
		}, nil

	case "tools/list":
		tools, err := server.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		return buildToolsListResult(tools, msgID)

	case "tools/call":
		if params == nil {
			params = make(map[string]any)
		}
		name := getString(params, "name")
		args, _ := params["arguments"].(map[string]any)
		if args == nil {
			args = make(map[string]any)
		}

		result, err := server.CallTool(ctx, name, args)
		if err != nil {
			return nil, err
		}

		content := make([]map[string]any, len(result.Content))
		for i, c := range result.Content {
			item := map[string]any{"type": c.Type}
			switch c.Type {
			case "text":
				item["text"] = c.Text
			case "image":
				item["data"] = c.Data
				item["mimeType"] = c.MimeType
			}
			content[i] = item
		}

		respData := map[string]any{"content": content}
		if result.IsError {
			respData["isError"] = true
		}
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      msgID,
			"result":  respData,
		}, nil

	case "notifications/initialized":
		// Notification - no response required per JSONRPC spec
		// 通知消息——按 JSONRPC 规范无需响应
		return map[string]any{"jsonrpc": "2.0", "result": map[string]any{}}, nil

	default:
		return nil, fmt.Errorf("method '%s' not found", method)
	}
}

// sendMcpResponse sends an MCP success response.
//
// sendMcpResponse 发送一个 MCP 成功响应。
func (p *Protocol) sendMcpResponse(ctx context.Context, requestID string, mcpResp map[string]any) error {
	response := SDKControlResponse{
		Type: MessageTypeControlResponse,
		Response: Response{
			Subtype:   ResponseSubtypeSuccess,
			RequestID: requestID,
			Response:  map[string]any{"mcp_response": mcpResp},
		},
	}
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal MCP response: %w", err)
	}
	return p.transport.Write(ctx, append(data, '\n'))
}

// buildToolsListResult builds the JSONRPC tools/list response payload from a
// slice of tool definitions. Kept standalone from the routeMcpMethod dispatch
// switch so the switch stays within the gocyclo budget, and so additions to
// the response shape do not perturb method-dispatch logic.
//
// buildToolsListResult 根据工具定义切片构建 JSONRPC tools/list 的响应负载。
// 将其从 routeMcpMethod 的分发 switch 中抽离，既使 switch 保持在 gocyclo 预算内，
// 也使响应结构的变更不影响方法分发逻辑。
func buildToolsListResult(tools []McpToolDefinition, msgID any) (map[string]any, error) {
	toolsData := make([]map[string]any, len(tools))
	for i, t := range tools {
		entry := map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		}
		if t.Annotations != nil {
			annMap, err := annotationsToMap(t.Annotations)
			if err != nil {
				return nil, err
			}
			entry["annotations"] = annMap
		}
		toolsData[i] = entry
	}
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      msgID,
		"result":  map[string]any{"tools": toolsData},
	}, nil
}

// annotationsToMap converts a ToolAnnotations value to a map[string]any with
// json `omitempty` honored, so only the fields the caller set appear on the
// wire. Empty fields are omitted from the resulting map.
//
// annotationsToMap 将 ToolAnnotations 值转换为 map[string]any，并尊重 json 的
// `omitempty`，从而只有调用方设置过的字段才会出现在线上；空字段会从结果 map 中省略。
func annotationsToMap(ann *ToolAnnotations) (map[string]any, error) {
	raw, err := json.Marshal(ann)
	if err != nil {
		return nil, fmt.Errorf("marshal annotations: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal annotations: %w", err)
	}
	return out, nil
}

// sendMcpErrorResponse sends an MCP JSONRPC error response.
//
// sendMcpErrorResponse 发送一个 MCP JSONRPC 错误响应。
func (p *Protocol) sendMcpErrorResponse(ctx context.Context, requestID string, msg map[string]any, code int, message string) error {
	errorResp := map[string]any{
		"jsonrpc": "2.0",
		"id":      msg["id"],
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	return p.sendMcpResponse(ctx, requestID, errorResp)
}
