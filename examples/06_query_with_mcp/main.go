// Package main demonstrates Query API with MCP tools (timezone operations).
//
// Package main 演示结合 MCP 工具执行时区相关查询的 Query API 用法。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Query API with MCP Tools Example")
	fmt.Println("Getting current time in multiple timezones using MCP time server")

	ctx := context.Background()
	query := "What time is it in Tokyo and New York?"

	fmt.Printf("\nQuery: %s\n", query)
	fmt.Println("Tools: MCP time server")

	// 通过 uvx 启动 MCP 时间服务器，让模型能调用外部时间工具。
	servers := map[string]claudecode.McpServerConfig{
		"time": &claudecode.McpStdioServerConfig{
			Type:    claudecode.McpServerTypeStdio,
			Command: "uvx",
			Args:    []string{"mcp-server-time"},
		},
	}

	// 为本次查询显式挂上 MCP 服务器和允许的工具列表。
	iterator, err := claudecode.Query(ctx, query,
		claudecode.WithMcpServers(servers),
		claudecode.WithAllowedTools("mcp__time__get_current_time"),
		claudecode.WithSystemPrompt("You are a helpful assistant. Use the MCP time server to get current time in different timezones."),
	)
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
		log.Fatalf("Query failed: %v", err)
	}
	defer iterator.Close()

	fmt.Println("\nResponse:")

	for {
		message, err := iterator.Next(ctx)
		if err != nil {
			if errors.Is(err, claudecode.ErrNoMoreMessages) {
				break
			}
			log.Printf("Message error: %v", err)
			break
		}

		if message == nil {
			break
		}

		// 助手消息输出自然语言结果；用户消息中则可能带有工具回传内容。
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
					fmt.Printf("Error: %s\n", *msg.Result)
				} else {
					fmt.Printf("Error: unknown error\n")
				}
			}
		}
	}

	fmt.Println("\nTimezone query completed!")
}
