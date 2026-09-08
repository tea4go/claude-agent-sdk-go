// Package main demonstrates advanced Client API features with WithClient,
// dynamic model switching, and error handling.
//
// Package main 演示 Client API 的进阶能力，包括 WithClient、动态模型切换与错误处理。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Advanced Client Features Example")
	fmt.Println("WithClient with dynamic model switching and error handling")

	ctx := context.Background()

	// 组合自定义 system prompt、模型切换和错误分类处理，模拟更贴近真实业务的调用方式。
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("\nConnected with custom configuration!")

		// 第一问使用默认模型，作为切换前的基线响应。
		fmt.Println("\n--- Question 1 (default model) ---")
		question1 := "Explain Go concurrency patterns for web crawlers with goroutine management"
		fmt.Printf("Q: %s\n", question1)

		if err := client.Query(ctx, question1); err != nil {
			return fmt.Errorf("query 1 failed: %w", err)
		}
		if err := streamResponse(ctx, client); err != nil {
			return fmt.Errorf("response 1 failed: %w", err)
		}

		// 在同一会话中切换模型，观察上下文保留与控制请求的效果。
		fmt.Println("\n--- Switching model to claude-sonnet-4-5 ---")
		sonnetModel := "claude-sonnet-4-5"
		if err := client.SetModel(ctx, &sonnetModel); err != nil {
			// 模型切换失败不阻断主演示流程，只给出提示。
			fmt.Printf("Note: Model switch failed (may not be supported): %v\n", err)
		} else {
			fmt.Println("Model switched successfully!")
		}

		// 第二问复用同一会话，但改用新的模型继续回答。
		fmt.Println("\n--- Question 2 (after model switch) ---")
		question2 := "Review this Go code for race conditions: func processItems(items []Item) error { var wg sync.WaitGroup; for _, item := range items { go func() { processItem(item) }() } }"
		fmt.Printf("Q: %s\n", question2)

		if err := client.Query(ctx, question2); err != nil {
			return fmt.Errorf("query 2 failed: %w", err)
		}
		if err := streamResponse(ctx, client); err != nil {
			return fmt.Errorf("response 2 failed: %w", err)
		}

		// 演示结束前恢复默认模型，避免后续请求继续使用临时配置。
		fmt.Println("\n--- Resetting to default model ---")
		if err := client.SetModel(ctx, nil); err != nil {
			fmt.Printf("Note: Model reset failed: %v\n", err)
		} else {
			fmt.Println("Model reset to default!")
		}

		fmt.Println("\nAdvanced session completed!")
		return nil
	},
		// 这些选项展示了 WithClient 在复杂配置下的常见搭配。
		claudecode.WithSystemPrompt("You are a senior Go developer providing code reviews and architectural guidance."),
		claudecode.WithAllowedTools("Read", "Write"), // Optional tools
	)
	// 通过 As* 辅助函数做细粒度错误分类，便于给出更准确的提示。
	if err != nil {
		// 按错误类型分别处理 CLI 缺失、连接失败和其他未知错误。
		if cliError := claudecode.AsCLINotFoundError(err); cliError != nil {
			fmt.Printf("[Error] Claude CLI not installed: %v\n", cliError)
			fmt.Println("Install: npm install -g @anthropic-ai/claude-code")
			return
		}

		if connError := claudecode.AsConnectionError(err); connError != nil {
			fmt.Printf("[Warning] Connection failed: %v\n", connError)
			fmt.Println("WithClient handled cleanup automatically")
			return
		}

		log.Fatalf("Advanced features failed: %v", err)
	}

	fmt.Println("\nAdvanced features demonstration completed!")
}

// streamResponse drains one response round from the client.
//
// streamResponse 读取并输出当前轮响应，直到收到结果消息或上下文结束。
func streamResponse(ctx context.Context, client claudecode.Client) error {
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
