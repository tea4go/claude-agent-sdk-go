// Package main demonstrates Client API with MCP time tools using WithClient for multi-turn time workflow.
// Also demonstrates GetMcpStatus() to inspect MCP server connection state after connecting.
//
// Package main 演示结合 MCP 时间工具的 Client API 多轮工作流，
// 同时展示连接建立后如何通过 GetMcpStatus() 检查 MCP 服务状态。
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Client API with MCP Time Tools Example")
	fmt.Println("Multi-turn time workflow with context preservation")

	ctx := context.Background()

	// 设计两步连续问题，验证第二步能否复用第一步的上下文结果。
	steps := []string{
		"What time is it in London?",
		"Convert that time to Tokyo timezone",
	}

	// 配置 MCP 时间服务器，供查询当前时间和时区转换使用。
	servers := map[string]claudecode.McpServerConfig{
		"time": &claudecode.McpStdioServerConfig{
			Type:    claudecode.McpServerTypeStdio,
			Command: "uvx",
			Args:    []string{"mcp-server-time"},
		},
	}

	// WithClient 自动维持上下文，适合这种多步时间处理流程。
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("\nConnected!")

		// 先检查 MCP 服务器是否就绪，避免把后续失败混淆成业务问题。
		fmt.Println("\n--- MCP Server Status ---")
		status, err := client.GetMcpStatus(ctx)
		if err != nil {
			fmt.Printf("  GetMcpStatus error: %v\n", err)
		} else {
			for _, srv := range status.McpServers {
				fmt.Printf("  Server: %s  Status: %s\n", srv.Name, srv.Status)
				if srv.Error != nil {
					fmt.Printf("    Error: %s\n", *srv.Error)
				}
				if srv.ServerInfo != nil {
					fmt.Printf("    Version: %s\n", srv.ServerInfo.Version)
				}
				for _, tool := range srv.Tools {
					desc := ""
					if tool.Description != nil {
						desc = *tool.Description
					}
					fmt.Printf("    Tool: %s  %s\n", tool.Name, desc)
				}
			}
		}

		fmt.Println("\nStarting multi-turn time workflow...")
		for i, step := range steps {
			fmt.Printf("\n--- Step %d ---\n", i+1)
			fmt.Printf("Query: %s\n", step)

			if err := client.Query(ctx, step); err != nil {
				return fmt.Errorf("step %d failed: %w", i+1, err)
			}

			if err := streamTimeResponse(ctx, client); err != nil {
				return fmt.Errorf("step %d response failed: %w", i+1, err)
			}
		}

		fmt.Println("\nTime workflow completed!")
		fmt.Println("Context was preserved - step 2 referenced the time from step 1")
		return nil
	}, claudecode.WithMcpServers(servers),
		claudecode.WithAllowedTools(
			"mcp__time__get_current_time",
			"mcp__time__convert_time"),
		claudecode.WithSystemPrompt("You are a helpful assistant. Use the MCP time server to get and convert times between timezones."))
	if err != nil {
		if cliErr := claudecode.AsCLINotFoundError(err); cliErr != nil {
			fmt.Printf("Claude CLI not found: %v\n", cliErr)
			fmt.Println("Install with: npm install -g @anthropic-ai/claude-code")
			return
		}
		if connErr := claudecode.AsConnectionError(err); connErr != nil {
			fmt.Printf("Connection failed: %v\n", connErr)
			return
		}
		log.Fatalf("Time workflow failed: %v", err)
	}
}

// streamTimeResponse reads one round of time-related responses and tool output.
//
// streamTimeResponse 读取当前轮与时间工具相关的响应内容。
func streamTimeResponse(ctx context.Context, client claudecode.Client) error {
	fmt.Println("\nResponse:")

	msgChan := client.ReceiveMessages(ctx)
	for {
		select {
		case message := <-msgChan:
			if message == nil {
				return nil
			}

			switch msg := message.(type) {
			case *claudecode.AssistantMessage:
				for _, block := range msg.Content {
					if textBlock, ok := block.(*claudecode.TextBlock); ok {
						fmt.Print(textBlock.Text)
					}
				}
			case *claudecode.UserMessage:
				if blocks, ok := msg.Content.([]claudecode.ContentBlock); ok {
					for _, block := range blocks {
						if toolResult, ok := block.(*claudecode.ToolResultBlock); ok {
							if content, ok := toolResult.Content.(string); ok {
								if strings.Contains(content, "tool_use_error") {
									fmt.Printf("Time Tool Error: %s\n", content)
								} else if len(content) > 150 {
									fmt.Printf("Time Result: %s...\n", content[:150])
								} else {
									fmt.Printf("Time Result: %s\n", content)
								}
							}
						}
					}
				}
			case *claudecode.ResultMessage:
				if msg.IsError {
					if msg.Result != nil {
						return fmt.Errorf("error: %s", *msg.Result)
					}
					return fmt.Errorf("error: unknown error")
				}
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
