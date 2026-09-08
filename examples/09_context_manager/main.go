// Package main demonstrates WithClient context manager pattern.
//
// Package main 演示 WithClient 上下文管理模式，以及它与手动连接方式的区别。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - WithClient Context Manager")
	fmt.Println("Automatic resource management vs manual pattern")

	ctx := context.Background()
	question := "What are the benefits of using context managers in programming?"

	// 先展示推荐写法：由 WithClient 负责连接和清理。
	fmt.Println("\n--- WithClient Pattern (Recommended) ---")
	fmt.Println("[+] Automatic connect/disconnect")
	fmt.Println("[+] Guaranteed cleanup on errors")

	if err := demonstrateWithClient(ctx, question); err != nil {
		if cliErr := claudecode.AsCLINotFoundError(err); cliErr != nil {
			fmt.Printf("Claude CLI not found: %v\n", cliErr)
			fmt.Println("Install with: npm install -g @anthropic-ai/claude-code")
			return
		}
		if connErr := claudecode.AsConnectionError(err); connErr != nil {
			fmt.Printf("Connection failed: %v\n", connErr)
			return
		}
		log.Printf("WithClient failed: %v", err)
	}

	// 再展示手动写法，便于对比样板代码和出错点。
	fmt.Println("\n--- Manual Pattern (Still Supported) ---")
	fmt.Println("[!] Manual connect/disconnect required")
	fmt.Println("[!] Easy to forget cleanup")

	if err := demonstrateManualPattern(ctx, question); err != nil {
		if cliErr := claudecode.AsCLINotFoundError(err); cliErr != nil {
			fmt.Printf("Claude CLI not found: %v\n", cliErr)
			fmt.Println("Install with: npm install -g @anthropic-ai/claude-code")
			return
		}
		if connErr := claudecode.AsConnectionError(err); connErr != nil {
			fmt.Printf("Connection failed: %v\n", connErr)
			return
		}
		log.Printf("Manual pattern failed: %v", err)
	}

	// 最后补一个错误处理演示，看看清理是否仍然可靠。
	fmt.Println("\n--- Error Handling ---")
	if err := demonstrateErrorScenarios(ctx); err != nil {
		log.Printf("Error demo failed: %v", err)
	}

	fmt.Println("\nRecommendation: Use WithClient for automatic resource management")
}

// demonstrateWithClient shows the recommended automatic resource-management pattern.
//
// demonstrateWithClient 展示推荐的自动资源管理写法。
func demonstrateWithClient(ctx context.Context, question string) error {
	fmt.Println("Using WithClient for automatic resource management...")

	return claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("Connected! Client managed automatically")

		if err := client.Query(ctx, question); err != nil {
			return fmt.Errorf("query failed: %w", err)
		}

		fmt.Println("\nResponse (first lines):")
		if err := showFirstLines(ctx, client, 3, 80); err != nil {
			return err
		}
		fmt.Println("[+] WithClient will handle cleanup automatically")
		return nil
	})
}

// demonstrateManualPattern shows the equivalent manual Connect/Disconnect flow.
//
// demonstrateManualPattern 展示等价的手动 Connect / Disconnect 流程。
func demonstrateManualPattern(ctx context.Context, question string) error {
	fmt.Println("Using manual Connect/Disconnect pattern...")

	client := claudecode.NewClient()

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	defer func() {
		fmt.Println("Manual cleanup...")
		if err := client.Disconnect(); err != nil {
			log.Printf("Disconnect warning: %v", err)
		}
		fmt.Println("[+] Manual cleanup completed")
	}()

	fmt.Println("Connected manually")

	if err := client.Query(ctx, question); err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	fmt.Println("\nResponse (first lines):")
	if err := showFirstLines(ctx, client, 3, 80); err != nil {
		return err
	}
	return nil
}

// demonstrateErrorScenarios verifies how WithClient behaves under common failures.
//
// demonstrateErrorScenarios 演示 WithClient 在常见失败场景下的行为。
func demonstrateErrorScenarios(ctx context.Context) error {
	fmt.Println("Testing WithClient error handling...")

	// 测试上下文取消时，WithClient 是否会正确返回并完成清理。
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	err := claudecode.WithClient(cancelCtx, func(client claudecode.Client) error {
		return client.Query(cancelCtx, "This will be cancelled")
	})
	if err != nil {
		fmt.Printf("[+] WithClient handled cancellation: %v\n", err)
	}

	// 测试业务函数主动返回错误时，连接是否同样会被自动关闭。
	err = claudecode.WithClient(ctx, func(client claudecode.Client) error {
		return fmt.Errorf("simulated application error")
	})
	if err != nil {
		fmt.Printf("[+] WithClient propagated error: %v\n", err)
		fmt.Println("   Connection was still cleaned up automatically")
	}

	return nil
}

// showFirstLines displays first lines of response from client.
//
// showFirstLines 只展示前几行响应，避免示例输出过长。
func showFirstLines(ctx context.Context, client claudecode.Client, maxLines, maxWidth int) error {
	msgChan := client.ReceiveMessages(ctx)
	linesShown := 0

	for linesShown < maxLines {
		select {
		case message := <-msgChan:
			if message == nil {
				return nil
			}

			switch msg := message.(type) {
			case *claudecode.AssistantMessage:
				for _, block := range msg.Content {
					if textBlock, ok := block.(*claudecode.TextBlock); ok {
						if linesShown < maxLines {
							text := textBlock.Text
							if len(text) > maxWidth {
								text = text[:maxWidth] + "..."
							}
							fmt.Printf("  %s\n", text)
							linesShown++
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

	// 读够目标行数后，把剩余消息清掉，避免影响下一轮查询。
	drainMessages(msgChan)
	return nil
}

// drainMessages consumes remaining messages from a channel.
//
// drainMessages 清空通道里残留的消息，直到通道暂时无数据或结束。
func drainMessages(msgChan <-chan claudecode.Message) {
	for {
		select {
		case message := <-msgChan:
			if message == nil {
				return
			}
		default:
			return
		}
	}
}
