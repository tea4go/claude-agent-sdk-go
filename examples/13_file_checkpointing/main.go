// Package main demonstrates the File Checkpointing and Rewind System.
//
// This example shows how to use file checkpointing to track file changes
// during a Claude Code session and rewind them to a previous state.
// File checkpointing enables:
// - Safe experimentation with file modifications
// - Undo/rollback capability for file changes
// - Recovery from unwanted modifications
// - Testing different approaches without permanent changes
//
// Key concepts:
// - WithFileCheckpointing(): Enables file change tracking
// - UserMessage.UUID: Checkpoint identifier for each user message
// - RewindFiles(): Reverts files to their state at a specific checkpoint
//
// NOTE: File checkpointing only works with the Client API (streaming mode).
// It is not available with the Query API (one-shot mode) because the control
// protocol required for rewind operations needs a persistent connection.
//
// Run: go run main.go
//
// Package main 演示文件检查点与回滚（Rewind）机制，重点说明如何捕获 UUID、
// 追踪文件变更，以及在真实场景中调用 RewindFiles。
package main

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	claudecode "github.com/tea4go/claude-agent-sdk-go"
)

// exampleDir returns the directory containing this source file.
//
// exampleDir 返回当前示例源码所在目录。
func exampleDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

func main() {
	fmt.Println("Claude Agent SDK - File Checkpointing Example")
	fmt.Println("=============================================")
	fmt.Println()

	// 示例 1：开启检查点并捕获 UserMessage.UUID。
	fmt.Println("--- Example 1: Capturing Checkpoints ---")
	fmt.Println("Setup: Enable checkpointing and capture UserMessage UUIDs")
	fmt.Println()
	runCheckpointCaptureExample()

	// 示例 2：观察会话过程中可能出现的文件修改。
	fmt.Println()
	fmt.Println("--- Example 2: File Modification Tracking ---")
	fmt.Println("Setup: Track file changes during a session")
	fmt.Println()
	runModificationTrackingExample()

	// 示例 3：串起一次完整的回滚工作流。
	fmt.Println()
	fmt.Println("--- Example 3: Rewind Workflow ---")
	fmt.Println("Setup: Demonstrate the rewind pattern with file operations")
	fmt.Println()
	runRewindWorkflowExample()

	fmt.Println()
	fmt.Println("File checkpointing examples completed!")
}

// CheckpointEntry represents a captured checkpoint.
//
// CheckpointEntry 表示一次已捕获的检查点信息。
type CheckpointEntry struct {
	Timestamp time.Time
	UUID      string
	Query     string
}

// runCheckpointCaptureExample demonstrates capturing UserMessage UUIDs.
//
// runCheckpointCaptureExample 演示如何在流式消息中捕获检查点 UUID。
func runCheckpointCaptureExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 回调里会写共享切片，因此这里用互斥锁保护检查点集合。
	var checkpoints []CheckpointEntry
	var mu sync.Mutex

	fmt.Println("Asking Claude to read a file (capturing checkpoint)...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		// 先发起一轮查询，让 CLI 产出对应的 UserMessage。
		query := "Read the file demo/notes.txt and tell me what version it is."
		if err := client.Query(ctx, query); err != nil {
			return err
		}

		// 从消息流中提取 UUID，并把它和当前查询绑定起来。
		return streamWithCheckpoints(ctx, client, func(uuid, q string) {
			mu.Lock()
			checkpoints = append(checkpoints, CheckpointEntry{
				Timestamp: time.Now(),
				UUID:      uuid,
				Query:     q,
			})
			mu.Unlock()
		}, query)
	}, claudecode.WithFileCheckpointing(), claudecode.WithMaxTurns(3), claudecode.WithCwd(exampleDir()))

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

	// 输出本轮捕获到的检查点摘要。
	fmt.Println("\n--- Captured Checkpoints ---")
	mu.Lock()
	for i, cp := range checkpoints {
		fmt.Printf("  %d. UUID: %s\n", i+1, truncate(cp.UUID, 36))
		fmt.Printf("     Query: %s\n", truncate(cp.Query, 50))
		fmt.Printf("     Time: %s\n", cp.Timestamp.Format("15:04:05"))
	}
	if len(checkpoints) == 0 {
		fmt.Println("  (No checkpoints captured - CLI may not have sent UserMessage)")
	}
	mu.Unlock()
}

// runModificationTrackingExample demonstrates file modification tracking.
//
// runModificationTrackingExample 演示如何根据工具调用信息判断是否发生了文件修改。
func runModificationTrackingExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 记录本轮观察到的潜在文件修改。
	var modifications []string
	var mu sync.Mutex

	fmt.Println("Asking Claude to describe a file (read-only operation)...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		// 这里先做只读操作，方便对比“没有修改”的结果。
		if err := client.Query(ctx, "Read demo/notes.txt and describe its contents briefly."); err != nil {
			return err
		}
		if err := streamWithModifications(ctx, client, &modifications, &mu); err != nil {
			return err
		}

		return nil
	}, claudecode.WithFileCheckpointing(), claudecode.WithMaxTurns(5), claudecode.WithCwd(exampleDir()))

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

	// 汇总打印检测到的修改情况。
	fmt.Println("\n--- Modification Summary ---")
	mu.Lock()
	if len(modifications) > 0 {
		for i, mod := range modifications {
			fmt.Printf("  %d. %s\n", i+1, mod)
		}
	} else {
		fmt.Println("  No file modifications detected (read-only operation)")
	}
	mu.Unlock()
}

// runRewindWorkflowExample demonstrates the rewind workflow pattern.
//
// runRewindWorkflowExample 演示一次典型的回滚工作流：捕获 UUID、保存检查点、准备回退。
func runRewindWorkflowExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 这里专门保存一个检查点 UUID，供后面展示 RewindFiles 调用方式。
	var capturedUUID string
	var mu sync.Mutex

	fmt.Println("Demonstrating the rewind workflow pattern...")
	fmt.Println()
	fmt.Println("Step 1: Enable checkpointing and start session")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		// 第 2 步：通过第一次交互拿到可回滚的检查点 UUID。
		fmt.Println("Step 2: Send query and capture checkpoint UUID")

		if err := client.Query(ctx, "Read demo/notes.txt and tell me the version in one sentence."); err != nil {
			return err
		}

		// 这里手动处理消息流：既要拿到 UserMessage.UUID，
		// 也要等到 ResultMessage，确保这一轮真正完成。
		msgChan := client.ReceiveMessages(ctx)
		var streamErr error
		resultReceived := false

		// 一旦拿到 UUID 和结果消息，就进入短暂 drain 阶段，吸收尾部消息。
		drainDeadline := time.After(200 * time.Millisecond)
		draining := false

		for {
			var timeoutChan <-chan time.Time
			if draining {
				timeoutChan = drainDeadline
			}

			select {
			case message := <-msgChan:
				if message == nil {
					// 通道关闭意味着这一轮消息已经全部送达。
					if streamErr != nil {
						return streamErr
					}
					goto processComplete
				}

				switch msg := message.(type) {
				case *claudecode.UserMessage:
					if msg.UUID != nil {
						mu.Lock()
						capturedUUID = *msg.UUID
						mu.Unlock()
						fmt.Printf("         Captured UUID: %s\n", truncate(*msg.UUID, 36))
					}
				case *claudecode.AssistantMessage:
					for _, block := range msg.Content {
						if textBlock, ok := block.(*claudecode.TextBlock); ok {
							text := textBlock.Text
							if len(text) > 100 {
								text = text[:100] + "..."
							}
							fmt.Printf("         Response: %s\n", strings.ReplaceAll(text, "\n", " "))
						}
					}
				case *claudecode.ResultMessage:
					resultReceived = true
					if msg.IsError {
						if msg.Result != nil {
							streamErr = fmt.Errorf("error: %s", *msg.Result)
						} else {
							streamErr = fmt.Errorf("error: unknown error")
						}
					}
				}

				// 核心条件满足后开始短暂 drain，避免遗漏尾部消息。
				if resultReceived && capturedUUID != "" && !draining {
					draining = true
					drainDeadline = time.After(200 * time.Millisecond)
				}

			case <-timeoutChan:
				// 短暂等待后仍无新消息，就认为本轮处理完成。
				if streamErr != nil {
					return streamErr
				}
				goto processComplete

			case <-ctx.Done():
				// 若上下文超时但关键数据已拿到，也允许把流程视作完成。
				mu.Lock()
				hasUUID := capturedUUID != ""
				mu.Unlock()
				if hasUUID && resultReceived {
					goto processComplete
				}
				return ctx.Err()
			}
		}
	processComplete:

		// 第 3 步：展示真实业务里应如何调用 RewindFiles。
		fmt.Println()
		fmt.Println("Step 3: RewindFiles usage pattern")

		mu.Lock()
		uuid := capturedUUID
		mu.Unlock()

		if uuid != "" {
			fmt.Println("         To rewind files to this checkpoint:")
			fmt.Printf("         err := client.RewindFiles(ctx, %q)\n", truncate(uuid, 36))
			fmt.Println()
			fmt.Println("         This would revert all file changes made after this point.")

			// 这里不真正执行回滚，原因有两个：
			// 1. 当前示例里没有真实文件改动；
			// 2. CLI 需要存在实际修改后才有回滚意义。
			// 如需真实演示，可取消下面的注释：
			// if err := client.RewindFiles(ctx, uuid); err != nil {
			//     fmt.Printf("         Rewind error: %v\n", err)
			// }
		} else {
			fmt.Println("         (No UUID captured - CLI may not have sent UserMessage)")
		}

		return nil
	}, claudecode.WithFileCheckpointing(), claudecode.WithMaxTurns(3), claudecode.WithCwd(exampleDir()))

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

	fmt.Println()
	fmt.Println("Workflow complete. In a real scenario, you would:")
	fmt.Println("  1. Enable checkpointing with WithFileCheckpointing()")
	fmt.Println("  2. Capture UUIDs from UserMessage during streaming")
	fmt.Println("  3. Call RewindFiles(ctx, uuid) to revert file changes")
}

// streamWithCheckpoints processes messages and captures UserMessage UUIDs.
//
// streamWithCheckpoints 处理消息流，并在遇到 UserMessage 时提取检查点 UUID。
func streamWithCheckpoints(
	ctx context.Context,
	client claudecode.Client,
	onCheckpoint func(uuid, query string),
	currentQuery string,
) error {
	msgChan := client.ReceiveMessages(ctx)

	for {
		select {
		case message := <-msgChan:
			if message == nil {
				return nil
			}

			switch msg := message.(type) {
			case *claudecode.UserMessage:
				// 每条用户消息都可能成为后续回滚的检查点。
				if msg.UUID != nil {
					onCheckpoint(*msg.UUID, currentQuery)
					fmt.Printf("  [CHECKPOINT] UUID captured: %s\n", truncate(*msg.UUID, 24))
				}
			case *claudecode.AssistantMessage:
				for _, block := range msg.Content {
					if textBlock, ok := block.(*claudecode.TextBlock); ok {
						text := textBlock.Text
						if len(text) > 150 {
							text = text[:150] + "..."
						}
						fmt.Printf("  Response: %s\n", strings.ReplaceAll(text, "\n", " "))
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

// streamWithModifications processes messages and tracks tool use for modifications.
//
// streamWithModifications 通过观察 ToolUseBlock 来推断潜在的文件修改行为。
func streamWithModifications(
	ctx context.Context,
	client claudecode.Client,
	modifications *[]string,
	mu *sync.Mutex,
) error {
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
					switch b := block.(type) {
					case *claudecode.TextBlock:
						text := b.Text
						if len(text) > 150 {
							text = text[:150] + "..."
						}
						fmt.Printf("  Response: %s\n", strings.ReplaceAll(text, "\n", " "))
					case *claudecode.ToolUseBlock:
						// 这里只把 Write / Edit 视为可能修改文件的工具。
						if b.Name == "Write" || b.Name == "Edit" {
							mu.Lock()
							*modifications = append(*modifications, fmt.Sprintf("%s tool used", b.Name))
							mu.Unlock()
							fmt.Printf("  [MODIFY] %s tool invoked\n", b.Name)
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

// truncate shortens a string to maxLen characters.
//
// truncate 截断过长字符串，方便在终端里展示 UUID 和查询摘要。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
