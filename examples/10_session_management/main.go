// Package main demonstrates session management with the Claude Agent SDK for Go.
//
// This example illustrates two distinct session concepts:
//
// Package main 演示 Claude Agent SDK for Go 中的会话管理，
// 并区分同一连接内的上下文会话与跨连接持久化的 CLI 会话。
//
//  1. Context Session ID - Used by QueryWithSession() to organize separate conversation
//     contexts WITHIN a single client connection. This is passed to the CLI via
//     StreamMessage.session_id and enables context isolation (e.g., "math" context
//     remembers 3+3, "default" context remembers 2+2). Not visible in Claude Code UI.
//
//  2. CLI Session UUID - The persistent conversation identifier returned in
//     ResultMessage.SessionID. This appears in Claude Code UI and is used with
//     WithResume() to continue conversations ACROSS client connections.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Session Management Example")
	fmt.Println("==============================================")

	ctx := context.Background()

	if err := runExample(ctx); err != nil {
		if cliErr := claudecode.AsCLINotFoundError(err); cliErr != nil {
			fmt.Printf("Claude CLI not found: %v\n", cliErr)
			fmt.Println("Install with: npm install -g @anthropic-ai/claude-code")
			return
		}
		if connErr := claudecode.AsConnectionError(err); connErr != nil {
			fmt.Printf("Connection failed: %v\n", connErr)
			return
		}
		log.Fatalf("Example failed: %v", err)
	}
}

// runExample walks through context isolation and session resumption.
//
// runExample 依次演示上下文隔离与会话恢复两种能力。
func runExample(ctx context.Context) error {
	// 第一部分：同一客户端连接内的上下文隔离。
	// 这里共享同一个 CLI Session UUID，但使用不同的 Context Session ID。
	var cliSessionUUID string

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		// 1. 默认上下文：对应 Query() 内部使用的 "default" 会话。
		fmt.Println("\n1. Default context with Query()")
		fmt.Println("   Asking: What's 2+2?")
		if err := client.Query(ctx, "Hello! What's 2+2? Reply briefly."); err != nil {
			return fmt.Errorf("default context query: %w", err)
		}
		result, err := streamResponse(ctx, client)
		if err != nil {
			return err
		}
		if result != nil {
			fmt.Printf("   [CLI Session UUID: %s]\n", result.SessionID)
		}

		// 2. 自定义上下文：与默认上下文共享 CLI 会话，但历史彼此隔离。
		fmt.Println("\n2. Custom context with QueryWithSession()")
		fmt.Println("   Asking: What's 3+3?")
		if err := client.QueryWithSession(ctx, "Hello! What's 3+3? Reply briefly.", "math-session"); err != nil {
			return fmt.Errorf("math context query: %w", err)
		}
		mathResult, err := streamResponse(ctx, client)
		if err != nil {
			return err
		}
		if mathResult != nil {
			cliSessionUUID = mathResult.SessionID
			fmt.Printf("   [CLI Session UUID: %s]\n", cliSessionUUID)
		}

		// 3. 用追问验证上下文隔离效果。
		// 两个上下文虽然共用一个 CLI 会话 UUID，但各自维护独立历史。
		fmt.Println("\n3. Context isolation demonstration")

		fmt.Println("   Default context asking about previous question:")
		if err := client.Query(ctx, "What was my previous math question? Reply briefly."); err != nil {
			return fmt.Errorf("default context isolation test: %w", err)
		}
		if _, err := streamResponse(ctx, client); err != nil {
			return err
		}

		fmt.Println("\n   Math context remembers its own history:")
		if err := client.QueryWithSession(ctx, "What was my previous math question? Reply briefly.", "math-session"); err != nil {
			return fmt.Errorf("math context isolation test: %w", err)
		}
		if _, err := streamResponse(ctx, client); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 第二部分：通过 WithResume 在新的客户端连接中恢复会话。
	// 这与上面的上下文隔离不同，恢复的是跨连接持久化的整段会话历史。
	if cliSessionUUID != "" {
		fmt.Println("\n4. Session Resumption with WithResume()")
		fmt.Printf("   Using CLI Session UUID: %s\n", cliSessionUUID)

		err = claudecode.WithClient(ctx, func(client claudecode.Client) error {
			fmt.Println("   Asking resumed session about previous context:")
			// 由于使用了 CLI Session UUID，新连接可以访问旧连接里的完整历史。
			if err := client.Query(ctx, "What math problem did we discuss earlier? Reply briefly."); err != nil {
				return fmt.Errorf("resumed session query: %w", err)
			}
			if _, err := streamResponse(ctx, client); err != nil {
				return err
			}
			return nil
		}, claudecode.WithResume(cliSessionUUID))
		if err != nil {
			return err
		}
	}

	fmt.Println("\nSession management demonstration completed!")
	return nil
}

// streamResponse streams a complete response from the client and returns the ResultMessage.
// This follows established SDK patterns for proper streaming output without messy duplicates.
//
// streamResponse 读取一轮完整响应，并把结果消息一并返回，便于调用方拿到 SessionID。
func streamResponse(ctx context.Context, client claudecode.Client) (*claudecode.ResultMessage, error) {
	msgChan := client.ReceiveMessages(ctx)
	for {
		select {
		case message := <-msgChan:
			if message == nil {
				return nil, nil
			}

			switch msg := message.(type) {
			case *claudecode.AssistantMessage:
				for _, block := range msg.Content {
					if textBlock, ok := block.(*claudecode.TextBlock); ok {
						fmt.Print(textBlock.Text)
					}
				}
			case *claudecode.ResultMessage:
				fmt.Println() // 当前轮回答输出完后补一个换行，便于阅读。
				if msg.IsError {
					if msg.Result != nil {
						return nil, fmt.Errorf("error: %s", *msg.Result)
					}
					return nil, fmt.Errorf("error: unknown error")
				}
				return msg, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
