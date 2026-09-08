// Package main demonstrates Partial Streaming for real-time updates.
//
// This example shows how to receive streaming updates as Claude generates
// responses, enabling real-time UI updates and progress indicators.
// Partial streaming enables:
// - Character-by-character or chunk-by-chunk text display
// - Real-time typing indicators and progress bars
// - Early processing of response content
// - Building responsive chat interfaces
//
// Key components:
// - WithPartialStreaming: Enable partial message streaming
// - WithIncludePartialMessages: Explicit control over partial messages
// - StreamEvent: Message type for streaming events
// - StreamEventType* constants: Event type identifiers
//
// Stream event types:
// - content_block_start: A new content block is starting
// - content_block_delta: Incremental content update
// - content_block_stop: Content block completed
// - message_start: Message generation beginning
// - message_delta: Message-level update
// - message_stop: Message generation complete
//
// Run: go run main.go
//
// Package main 演示部分流式输出（Partial Streaming），帮助理解如何用 StreamEvent
// 做逐步渲染、输入中提示和实时状态更新。
package main

import (
	"context"
	"fmt"
	"time"

	claudecode "github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Partial Streaming Example")
	fmt.Println("=============================================")
	fmt.Println()

	// 示例 1：先把可用的事件类型常量列出来。
	fmt.Println("--- Example 1: Stream Event Types ---")
	fmt.Println("Available stream event type constants:")
	showStreamEventTypes()

	// 示例 2：展示最基础的部分流式接收。
	fmt.Println()
	fmt.Println("--- Example 2: Basic Partial Streaming ---")
	fmt.Println("Enabling partial streaming to receive real-time updates...")
	runPartialStreamingExample()

	// 示例 3：把流事件进一步组合成“正在输入”指示器。
	fmt.Println()
	fmt.Println("--- Example 3: Typing Indicator Pattern ---")
	fmt.Println("Building a typing indicator with stream events...")
	demonstrateTypingIndicator()

	fmt.Println()
	fmt.Println("Partial streaming example completed!")
}

// showStreamEventTypes displays all available stream event type constants.
//
// showStreamEventTypes 输出所有可用的流事件类型常量。
func showStreamEventTypes() {
	fmt.Printf("  ContentBlockStart: %q\n", claudecode.StreamEventTypeContentBlockStart)
	fmt.Printf("  ContentBlockDelta: %q\n", claudecode.StreamEventTypeContentBlockDelta)
	fmt.Printf("  ContentBlockStop:  %q\n", claudecode.StreamEventTypeContentBlockStop)
	fmt.Printf("  MessageStart:      %q\n", claudecode.StreamEventTypeMessageStart)
	fmt.Printf("  MessageDelta:      %q\n", claudecode.StreamEventTypeMessageDelta)
	fmt.Printf("  MessageStop:       %q\n", claudecode.StreamEventTypeMessageStop)
}

// runPartialStreamingExample demonstrates receiving StreamEvent messages.
//
// runPartialStreamingExample 演示如何消费 StreamEvent 并拼接增量文本。
func runPartialStreamingExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var eventCount int
	var deltaCount int
	var textBuffer string

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		// 提问尽量简短，这样更容易看清楚每个事件的到达顺序。
		if err := client.Query(ctx, "Count from 1 to 5, putting each number on its own line."); err != nil {
			return err
		}

		msgChan := client.ReceiveMessages(ctx)

		for {
			select {
			case message := <-msgChan:
				if message == nil {
					return nil
				}

				switch msg := message.(type) {
				case *claudecode.StreamEvent:
					eventCount++
					eventType, _ := msg.Event["type"].(string)

					switch eventType {
					case claudecode.StreamEventTypeContentBlockStart:
						fmt.Printf("[%s] Content block starting...\n", eventType)

					case claudecode.StreamEventTypeContentBlockDelta:
						deltaCount++
						// 从 delta 事件里提取文本增量，并手动累计。
						if delta, ok := msg.Event["delta"].(map[string]any); ok {
							if text, ok := delta["text"].(string); ok {
								textBuffer += text
								// 每次增量到达就立即输出，模拟实时渲染。
								fmt.Printf("[delta %d] %q\n", deltaCount, text)
							}
						}

					case claudecode.StreamEventTypeContentBlockStop:
						fmt.Printf("[%s] Content block complete\n", eventType)

					case claudecode.StreamEventTypeMessageStart:
						fmt.Printf("[%s] Message generation starting\n", eventType)

					case claudecode.StreamEventTypeMessageStop:
						fmt.Printf("[%s] Message generation complete\n", eventType)

					default:
						fmt.Printf("[%s] Event received\n", eventType)
					}

				case *claudecode.AssistantMessage:
					// 即便开启部分流式，最终仍会收到一条完整的 AssistantMessage。
					for _, block := range msg.Content {
						if textBlock, ok := block.(*claudecode.TextBlock); ok {
							fmt.Printf("\nFull message: %s\n", textBlock.Text)
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
	},
		claudecode.WithPartialStreaming(),
		claudecode.WithMaxTurns(1),
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
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total stream events: %d\n", eventCount)
	fmt.Printf("  Content deltas: %d\n", deltaCount)
	fmt.Printf("  Accumulated text length: %d chars\n", len(textBuffer))
}

// demonstrateTypingIndicator shows how to build a typing indicator.
//
// demonstrateTypingIndicator 演示如何基于 StreamEvent 搭一个简易“输入中”效果。
func demonstrateTypingIndicator() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("Simulating a chat interface with typing indicator...")
	fmt.Println()

	isTyping := false
	var charCount int

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.Query(ctx, "Say hello in exactly 10 words."); err != nil {
			return err
		}

		msgChan := client.ReceiveMessages(ctx)

		for {
			select {
			case message := <-msgChan:
				if message == nil {
					return nil
				}

				switch msg := message.(type) {
				case *claudecode.StreamEvent:
					eventType, _ := msg.Event["type"].(string)

					switch eventType {
					case claudecode.StreamEventTypeContentBlockStart:
						if !isTyping {
							isTyping = true
							fmt.Print("Claude is typing")
						}

					case claudecode.StreamEventTypeContentBlockDelta:
						// 每收到一定数量的增量，就补一个点，模拟输入动画。
						charCount++
						if charCount%5 == 0 {
							fmt.Print(".")
						}

					case claudecode.StreamEventTypeMessageStop:
						if isTyping {
							fmt.Println(" done!")
							isTyping = false
						}
					}

				case *claudecode.AssistantMessage:
					fmt.Print("\nClaude: ")
					for _, block := range msg.Content {
						if textBlock, ok := block.(*claudecode.TextBlock); ok {
							fmt.Println(textBlock.Text)
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
	},
		claudecode.WithPartialStreaming(),
		claudecode.WithMaxTurns(1),
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
		fmt.Printf("Error: %v\n", err)
	}
}
