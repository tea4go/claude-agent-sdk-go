// Package main demonstrates Client API with file tools using WithClient for automatic resource management.
// Also showcases dynamic permission mode switching with SetPermissionMode and tool_use_result metadata
// for rich edit information (file paths, diffs, patches).
//
// Package main 演示结合文件工具的 Client API 用法，同时展示 WithClient 自动资源管理、
// SetPermissionMode 动态权限切换，以及 tool_use_result 元数据的读取方式。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Client API with Tools Example")
	fmt.Println("Interactive file operations with context preservation")

	// 准备演示文件，让后续 Read / Edit / Write 都有真实目标可操作。
	if err := setupFiles(); err != nil {
		log.Fatalf("Setup failed: %v", err)
	}
	defer os.RemoveAll("demo")

	if err := os.Chdir("demo"); err != nil {
		log.Fatalf("Failed to change to demo directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(".."); err != nil {
			log.Printf("Warning: Failed to change back to parent directory: %v", err)
		}
	}()

	ctx := context.Background()

	// 通过 WithClient 开启多轮交互，观察上下文与工具调用信息如何自动保留。
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		fmt.Println("\nConnected! Starting interactive session...")

		// 第 1 轮：先分析项目，为下一轮修改建立上下文。
		fmt.Println("\n--- Turn 1: Project Analysis ---")
		query1 := "Read all files in this project and give me an overview of what it does"

		if err := client.Query(ctx, query1); err != nil {
			return fmt.Errorf("turn 1 query failed: %w", err)
		}

		if err := streamResponse(ctx, client); err != nil {
			return fmt.Errorf("turn 1 failed: %w", err)
		}

		// 在准备修改文件前切换到自动接受编辑模式，
		// 演示运行时动态调整权限策略的效果。
		fmt.Println("\n--- Switching to auto-accept edits mode ---")
		if err := client.SetPermissionMode(ctx, claudecode.PermissionModeAcceptEdits); err != nil {
			// 权限模式切换不是强依赖，失败时保留提示但继续演示主流程。
			fmt.Printf("Note: Permission mode switch failed (may not be supported): %v\n", err)
		} else {
			fmt.Println("Now auto-accepting file edits!")
		}

		// 第 2 轮：基于第一轮上下文执行实际改动。
		fmt.Println("\n--- Turn 2: Code Improvements (auto-accept mode) ---")
		query2 := "Based on what you learned, improve the main.go file with better error handling and create a README.md"

		if err := client.Query(ctx, query2); err != nil {
			return fmt.Errorf("turn 2 query failed: %w", err)
		}

		if err := streamResponse(ctx, client); err != nil {
			return fmt.Errorf("turn 2 failed: %w", err)
		}

		fmt.Println("\nInteractive session completed!")
		fmt.Println("Context was preserved across both turns automatically")
		return nil
	}, claudecode.WithAllowedTools("Read", "Write", "Edit"),
		claudecode.WithSystemPrompt("You are a helpful software development assistant."))
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
		log.Fatalf("Session failed: %v", err)
	}
}

// streamResponse reads one round of client messages and prints both model output
// and tool feedback.
//
// streamResponse 读取一轮会话消息，并同时打印模型回复与工具反馈。
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
			case *claudecode.UserMessage:
				// tool_use_result 会附带文件路径、结构化补丁等丰富编辑信息，
				// 适合做 UI 展示或审计输出。
				if msg.HasToolUseResult() {
					result := msg.GetToolUseResult()
					if filePath, ok := result["filePath"].(string); ok {
						fmt.Printf("[Edit] %s", filePath)
						// 如果结构化补丁存在，顺手展示首个补丁的位置信息。
						if patches, ok := result["structuredPatch"].([]any); ok && len(patches) > 0 {
							if patch, ok := patches[0].(map[string]any); ok {
								if oldStart, ok := patch["oldStart"].(float64); ok {
									if lines, ok := patch["lines"].([]any); ok {
										fmt.Printf(" (line %d, %d changes)", int(oldStart), len(lines))
									}
								}
							}
						}
						fmt.Println()
					}
				}

				// 额外打印工具结果正文，便于看到文件读写的直接反馈。
				if blocks, ok := msg.Content.([]claudecode.ContentBlock); ok {
					for _, block := range blocks {
						if toolResult, ok := block.(*claudecode.ToolResultBlock); ok {
							if content, ok := toolResult.Content.(string); ok {
								if len(content) > 100 {
									fmt.Printf("[File] %s...\n", content[:100])
								} else {
									fmt.Printf("[File] %s\n", content)
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

// setupFiles creates a small demo project for the tool workflow.
//
// setupFiles 创建一个小型演示项目，供工具工作流示例使用。
func setupFiles() error {
	if err := os.MkdirAll("demo", 0o755); err != nil {
		return err
	}

	packageJSON := `{
  "name": "demo-web-server",
  "version": "1.0.0",
  "description": "Simple Go web server demo",
  "main": "main.go",
  "keywords": ["go", "web", "demo"],
  "author": "Demo"
}`

	mainGo := `package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})
	
	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", nil)
}`

	files := map[string]string{
		filepath.Join("demo", "package.json"): packageJSON,
		filepath.Join("demo", "main.go"):      mainGo,
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}

	return nil
}
