// Package main demonstrates streaming with Client API using automatic resource management.
//
// Package main 演示结合自动资源管理的 Client API 流式响应。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Client Streaming Example")
	fmt.Println("Asking: Explain Go goroutines with a simple example")

	ctx := context.Background()
	question := "Explain what Go goroutines are and show a simple example"

	// WithClient 自动管理连接生命周期，适合示例和常规业务代码。
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("\nConnected! Streaming response:")

		if err := client.Query(ctx, question); err != nil {
			return fmt.Errorf("query failed: %w", err)
		}

		// 实时消费消息流，边收到边输出。
		msgChan := client.ReceiveMessages(ctx)
		for {
			select {
			case message := <-msgChan:
				if message == nil {
					return nil // 消息流结束。
				}

				switch msg := message.(type) {
				case *claudecode.AssistantMessage:
					// 助手消息可能按块到达，这里直接流式打印文本块。
					for _, block := range msg.Content {
						if textBlock, ok := block.(*claudecode.TextBlock); ok {
							fmt.Print(textBlock.Text)
						}
					}
				case *claudecode.ResultMessage:
					if msg.IsError {
						if msg.Result != nil {
							return fmt.Errorf("error: %s", *msg.Result)
						}
						return fmt.Errorf("error: unknown error")
					}
					return nil // 成功收尾，当前轮响应完成。
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
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
		log.Fatalf("Streaming failed: %v", err)
	}

	fmt.Println("\n\nStreaming completed!")
}
