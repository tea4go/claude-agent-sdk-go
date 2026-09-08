// Package main demonstrates basic usage of the Claude Agent SDK Query API.
//
// Package main 演示 Claude Agent SDK Query API 的基础用法。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Query API Example")
	fmt.Println("Asking: What is 2+2?")

	ctx := context.Background()

	// 创建并执行一次性查询。
	iterator, err := claudecode.Query(ctx, "What is 2+2?")
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

	// 逐条遍历返回消息，并按消息类型提取文本或错误结果。
	for {
		message, err := iterator.Next(ctx)
		if err != nil {
			if errors.Is(err, claudecode.ErrNoMoreMessages) {
				break
			}
			log.Fatalf("Failed to get message: %v", err)
		}

		if message == nil {
			break
		}

		// 根据消息类型分别处理正文与结果状态。
		switch msg := message.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range msg.Content {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					fmt.Print(textBlock.Text)
				}
			}
		case *claudecode.ResultMessage:
			if msg.IsError {
				if msg.Result != nil {
					log.Printf("Error: %s", *msg.Result)
				} else {
					log.Printf("Error: unknown error")
				}
			}
		}
	}

	fmt.Println("\nQuery completed!")
}
