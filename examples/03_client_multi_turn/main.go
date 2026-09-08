// Package main demonstrates multi-turn conversation with context preservation using WithClient.
//
// Package main 演示使用 WithClient 保持上下文的多轮对话。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Multi-Turn Conversation Example")
	fmt.Println("Building context across multiple related questions")

	ctx := context.Background()

	// 这些问题前后关联，用于观察上下文在多轮对话中的自动保留效果。
	questions := []string{
		"What is a binary search tree?",
		"Can you show me a Go implementation of inserting a node?",
		"What would be the time complexity of that insertion?",
		"How would I implement a search function for the same tree?",
	}

	// WithClient 会在同一连接内自动保留上下文，无需手动维护历史消息。
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("\nConnected! Starting conversation...")

		for i, question := range questions {
			fmt.Printf("\n--- Turn %d ---\n", i+1)
			fmt.Printf("Q: %s\n\n", question)

			if err := client.Query(ctx, question); err != nil {
				return fmt.Errorf("turn %d failed: %w", i+1, err)
			}

			// 当前轮提问后，流式读取完整回答。
			if err := streamFullResponse(ctx, client); err != nil {
				return fmt.Errorf("turn %d streaming failed: %w", i+1, err)
			}
		}

		fmt.Println("\n\nConversation completed!")
		fmt.Println("Notice: Each question built on previous responses automatically")
		return nil
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
		log.Fatalf("Conversation failed: %v", err)
	}
}

// streamFullResponse streams a complete response from the client.
//
// streamFullResponse 从客户端持续读取消息，直到当前轮响应结束或出现错误。
func streamFullResponse(ctx context.Context, client claudecode.Client) error {
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
