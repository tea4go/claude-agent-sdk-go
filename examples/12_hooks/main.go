// Package main demonstrates the Hook System for lifecycle events.
//
// This example shows how to use hooks to intercept and respond to
// lifecycle events during Claude Code CLI execution. Hooks enable:
// - Logging and auditing of tool usage (PreToolUse, PostToolUse)
// - Blocking dangerous commands before execution
// - Adding context to tool responses
// - Monitoring session lifecycle events
//
// Hook events supported:
// - PreToolUse: Before a tool executes (can block or modify input)
// - PostToolUse: After a tool executes (can add context)
// - PostToolUseFailure: After a tool fails (can inject recovery context)
// - UserPromptSubmit: When user submits a prompt
// - Stop: When session is stopping
// - SubagentStop: When a subagent is stopping
// - PreCompact: Before context compaction
// - Notification: When the CLI emits a notification
// - SubagentStart: When a subagent starts
// - PermissionRequest: When a permission is requested
//
// NOTE: Hooks are invoked when the CLI sends hook callback requests
// to the SDK. The callbacks demonstrate the correct API usage pattern
// for handling these lifecycle events.
//
// Run: go run main.go
//
// Package main 演示 Hook 系统，覆盖工具执行前后、失败恢复、通知观察等生命周期事件。
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
	fmt.Println("Claude Agent SDK - Hook System Example")
	fmt.Println("======================================")
	fmt.Println()

	// 示例 1：记录工具执行前后的 Hook 触发情况。
	fmt.Println("--- Example 1: Tool Logging Hooks ---")
	fmt.Println("Hook: Log all tool usage before and after execution")
	fmt.Println()
	runToolLoggingExample()

	// 示例 2：在执行前拦截危险命令。
	fmt.Println()
	fmt.Println("--- Example 2: Command Blocking Hook ---")
	fmt.Println("Hook: Block dangerous bash commands before execution")
	fmt.Println()
	runBlockingExample()

	// 示例 3：在工具执行后给 Claude 注入补充上下文。
	fmt.Println()
	fmt.Println("--- Example 3: Context Injection Hook ---")
	fmt.Println("Hook: Add timing information after tool execution")
	fmt.Println()
	runContextInjectionExample()

	// 示例 4：工具失败后注入恢复建议。
	fmt.Println()
	fmt.Println("--- Example 4: Tool Failure Recovery Hook ---")
	fmt.Println("Hook: Inject recovery context when a Bash command fails")
	fmt.Println()
	runFailureRecoveryExample()

	// 示例 5：观察 CLI 发出的通知事件。
	fmt.Println()
	fmt.Println("--- Example 5: Notification Hook ---")
	fmt.Println("Hook: Observe CLI-emitted notifications via the generic WithHook API")
	fmt.Println()
	runNotificationExample()

	fmt.Println()
	fmt.Println("Hook system examples completed!")
}

// runToolLoggingExample demonstrates logging tool usage with hooks.
//
// runToolLoggingExample 演示如何用 Hook 记录工具执行前后的日志。
func runToolLoggingExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Hook 可能并发触发，因此日志切片需要加锁保护。
	var toolLog []ToolLogEntry
	var logMu sync.Mutex

	// PreToolUse：在工具真正执行前记录一条日志。
	preToolHook := claudecode.WithPreToolUseHook("", func(
		_ context.Context,
		input any,
		_ *string,
		_ claudecode.HookContext,
	) (claudecode.HookJSONOutput, error) {
		preInput, ok := input.(*claudecode.PreToolUseHookInput)
		if !ok {
			return claudecode.HookJSONOutput{}, nil
		}

		entry := ToolLogEntry{
			Timestamp: time.Now(),
			Tool:      preInput.ToolName,
			Phase:     "PRE",
		}

		logMu.Lock()
		toolLog = append(toolLog, entry)
		logMu.Unlock()

		fmt.Printf("  [PRE]  Tool: %-10s | Session: %s\n",
			preInput.ToolName, truncate(preInput.SessionID, 12))

		return claudecode.HookJSONOutput{}, nil
	})

	// PostToolUse：在工具执行完成后再补一条结果日志。
	postToolHook := claudecode.WithPostToolUseHook("", func(
		_ context.Context,
		input any,
		_ *string,
		_ claudecode.HookContext,
	) (claudecode.HookJSONOutput, error) {
		postInput, ok := input.(*claudecode.PostToolUseHookInput)
		if !ok {
			return claudecode.HookJSONOutput{}, nil
		}

		entry := ToolLogEntry{
			Timestamp: time.Now(),
			Tool:      postInput.ToolName,
			Phase:     "POST",
		}

		logMu.Lock()
		toolLog = append(toolLog, entry)
		logMu.Unlock()

		fmt.Printf("  [POST] Tool: %-10s | Response type: %T\n",
			postInput.ToolName, postInput.ToolResponse)

		return claudecode.HookJSONOutput{}, nil
	})

	fmt.Println("Asking Claude to read a file...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.Query(ctx, "Read the file demo/sample.txt and tell me what it contains in one sentence."); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	}, preToolHook, postToolHook, claudecode.WithMaxTurns(3), claudecode.WithCwd(exampleDir()))

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

	// 汇总打印 Hook 调用记录，方便验证前后钩子是否成对触发。
	fmt.Println("\n--- Tool Log Summary ---")
	logMu.Lock()
	for i, entry := range toolLog {
		fmt.Printf("  %d. [%s] %s at %s\n",
			i+1, entry.Phase, entry.Tool, entry.Timestamp.Format("15:04:05.000"))
	}
	fmt.Printf("Total hook invocations: %d\n", len(toolLog))
	logMu.Unlock()
}

// runBlockingExample demonstrates blocking dangerous commands with PreToolUse hooks.
//
// runBlockingExample 演示在 PreToolUse 阶段拦截危险 Bash 命令。
func runBlockingExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 预设一组高风险命令片段，用于演示拦截逻辑。
	dangerousPatterns := []string{"rm -rf", "sudo", "chmod 777", "> /dev/"}

	// 仅挂在 Bash 上，这样只会检查命令型工具输入。
	blockingHook := claudecode.WithPreToolUseHook("Bash", func(
		_ context.Context,
		input any,
		_ *string,
		_ claudecode.HookContext,
	) (claudecode.HookJSONOutput, error) {
		preInput, ok := input.(*claudecode.PreToolUseHookInput)
		if !ok {
			return claudecode.HookJSONOutput{}, nil
		}

		// 从工具输入中取出实际命令文本。
		command, ok := preInput.ToolInput["command"].(string)
		if !ok {
			return claudecode.HookJSONOutput{}, nil
		}

		// 命中危险模式就返回 block 决策，阻止 CLI 真正执行。
		for _, pattern := range dangerousPatterns {
			if strings.Contains(strings.ToLower(command), strings.ToLower(pattern)) {
				fmt.Printf("  [BLOCK] Dangerous command detected: %q\n", truncate(command, 50))

				// 通过 HookJSONOutput 显式告诉 CLI：这条命令要被阻止。
				decision := "block"
				reason := fmt.Sprintf("Command blocked: contains dangerous pattern '%s'", pattern)
				return claudecode.HookJSONOutput{
					Decision: &decision,
					Reason:   &reason,
				}, nil
			}
		}

		fmt.Printf("  [ALLOW] Safe command: %q\n", truncate(command, 50))
		return claudecode.HookJSONOutput{}, nil
	})

	fmt.Println("Asking Claude to run some commands (dangerous ones will be blocked)...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.Query(ctx, "Run 'ls -la demo/' to list files, then run 'echo hello' to test."); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	}, blockingHook, claudecode.WithMaxTurns(5), claudecode.WithCwd(exampleDir()))

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

// runContextInjectionExample demonstrates adding context after tool execution.
//
// runContextInjectionExample 演示如何在工具执行后，把额外上下文回灌给 Claude。
func runContextInjectionExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 用 toolUseID 建立开始时间索引，统计每次工具调用耗时。
	toolStartTimes := make(map[string]time.Time)
	var timingMu sync.Mutex

	// 前置 Hook 记录起始时间。
	preHook := claudecode.WithPreToolUseHook("", func(
		_ context.Context,
		_ any,
		toolUseID *string,
		_ claudecode.HookContext,
	) (claudecode.HookJSONOutput, error) {
		if toolUseID == nil {
			return claudecode.HookJSONOutput{}, nil
		}

		timingMu.Lock()
		toolStartTimes[*toolUseID] = time.Now()
		timingMu.Unlock()

		return claudecode.HookJSONOutput{}, nil
	})

	// 后置 Hook 计算耗时，并把结果注入为附加上下文。
	postHook := claudecode.WithPostToolUseHook("", func(
		_ context.Context,
		input any,
		toolUseID *string,
		_ claudecode.HookContext,
	) (claudecode.HookJSONOutput, error) {
		postInput, ok := input.(*claudecode.PostToolUseHookInput)
		if !ok || toolUseID == nil {
			return claudecode.HookJSONOutput{}, nil
		}

		timingMu.Lock()
		startTime, exists := toolStartTimes[*toolUseID]
		delete(toolStartTimes, *toolUseID)
		timingMu.Unlock()

		if !exists {
			return claudecode.HookJSONOutput{}, nil
		}

		duration := time.Since(startTime)
		context := fmt.Sprintf("Tool %s completed in %v", postInput.ToolName, duration)
		fmt.Printf("  [TIMING] %s\n", context)

		// AdditionalContext 会在 Claude 下一轮继续可见，适合传递诊断信息。
		return claudecode.HookJSONOutput{
			HookSpecificOutput: claudecode.PostToolUseHookSpecificOutput{
				HookEventName:     "PostToolUse",
				AdditionalContext: &context,
			},
		}, nil
	})

	fmt.Println("Asking Claude to perform file operations (timing will be tracked)...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.Query(ctx, "List the files in demo/ directory and read sample.txt if it exists."); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	}, preHook, postHook, claudecode.WithMaxTurns(5), claudecode.WithCwd(exampleDir()))

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

// runFailureRecoveryExample demonstrates injecting recovery context after a tool failure.
//
// The PostToolUseFailure event fires when a tool invocation exits non-zero (Bash) or
// otherwise reports an error. Unlike PostToolUse, it gives the hook a chance to attach
// guidance Claude can use on its next turn - file path hints, retry advice, etc.
//
// User-initiated interrupts (IsInterrupt non-nil and dereferences to true) are
// treated specially: the hook stays silent rather than encouraging a retry, since
// Ctrl+C signals stop intent. A nil IsInterrupt means the CLI omitted the field
// (Python NotRequired[bool] semantics) and must not be treated as an interrupt.
//
// runFailureRecoveryExample 演示工具失败后如何通过 Hook 向下一轮注入恢复提示。
func runFailureRecoveryExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// PostToolUseFailure 目前没有专用便捷函数，因此通过通用 WithHook 注册。
	failureHook := claudecode.WithHook(
		claudecode.HookEventPostToolUseFailure,
		"Bash",
		recoveryCallback,
	)

	fmt.Println("Asking Claude to run a Bash command that will fail...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.Query(ctx, "Run 'cat /nonexistent-file' and tell me what happens."); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	}, failureHook, claudecode.WithMaxTurns(3), claudecode.WithCwd(exampleDir()))

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

// runNotificationExample demonstrates observing CLI-emitted notifications.
//
// The Notification event fires when the CLI surfaces a system-level notification
// (e.g., permission prompts, rate-limit warnings). Unlike Pre/PostToolUse, there
// is no user-controlled way to force-trigger one from a single query - the hook
// observes whatever the CLI decides to notify about during the session.
//
// Registered via generic WithHook because no convenience helper exists for this event.
//
// runNotificationExample 演示如何旁路观察 CLI 产生的通知消息。
func runNotificationExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	notificationHook := claudecode.WithHook(
		claudecode.HookEventNotification,
		"", // 非工具事件不会用到 matcher，这里传空即可。
		notificationCallback,
	)

	fmt.Println("Running a query; any CLI notifications will be observed by the hook...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.Query(ctx, "Briefly summarize the current directory."); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	}, notificationHook, claudecode.WithMaxTurns(2), claudecode.WithCwd(exampleDir()))

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

// notificationCallback is the Notification handler for runNotificationExample.
// Logs the notification fields with a nil-guard on the optional Title.
//
// notificationCallback 是通知事件处理函数，重点演示如何安全读取可选字段 Title。
func notificationCallback(
	_ context.Context,
	input any,
	_ *string,
	_ claudecode.HookContext,
) (claudecode.HookJSONOutput, error) {
	notif, ok := input.(*claudecode.NotificationHookInput)
	if !ok {
		return claudecode.HookJSONOutput{}, nil
	}

	title := "(none)"
	if notif.Title != nil {
		title = *notif.Title
	}
	fmt.Printf("  [NOTIFY] type=%s title=%q message=%q\n",
		notif.NotificationType, title, truncate(notif.Message, 80))

	return claudecode.HookJSONOutput{}, nil
}

// recoveryCallback is the PostToolUseFailure handler for runFailureRecoveryExample.
// Skips context injection when IsInterrupt is true to respect user stop intent.
//
// recoveryCallback 在工具失败时注入恢复建议；若是用户主动中断，则保持静默。
func recoveryCallback(
	_ context.Context,
	input any,
	_ *string,
	_ claudecode.HookContext,
) (claudecode.HookJSONOutput, error) {
	failInput, ok := input.(*claudecode.PostToolUseFailureHookInput)
	if !ok {
		return claudecode.HookJSONOutput{}, nil
	}

	if failInput.IsInterrupt != nil && *failInput.IsInterrupt {
		fmt.Printf("  [INTERRUPT] Tool %s cancelled by user; no recovery hint injected\n",
			failInput.ToolName)
		return claudecode.HookJSONOutput{}, nil
	}

	fmt.Printf("  [RECOVERY] Tool %s failed: %s\n",
		failInput.ToolName, truncate(failInput.Error, 80))

	return claudecode.HookJSONOutput{
		HookSpecificOutput: claudecode.PostToolUseFailureHookSpecificOutput{
			HookEventName:     "PostToolUseFailure",
			AdditionalContext: ptrTo("The previous command failed. Verify the file path exists before retrying, or try a different approach."),
		},
	}, nil
}

// ptrTo returns a pointer to the given value. Useful for *string fields like AdditionalContext.
//
// ptrTo 返回某个值的指针，便于构造 AdditionalContext 这类可选字段。
func ptrTo[T any](v T) *T { return &v }

// ToolLogEntry represents a logged tool usage event.
//
// ToolLogEntry 表示一条工具调用日志记录。
type ToolLogEntry struct {
	Timestamp time.Time
	Tool      string
	Phase     string // 标记发生在执行前还是执行后。
}

// streamResponse reads and displays messages from the client.
//
// streamResponse 读取客户端消息，并把结果压缩为简短可读的示例输出。
func streamResponse(ctx context.Context, client claudecode.Client) error {
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
						text := textBlock.Text
						if len(text) > 150 {
							text = text[:150] + "..."
						}
						fmt.Printf("Response: %s\n", strings.ReplaceAll(text, "\n", " "))
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
// truncate 把过长字符串截断到指定长度，避免示例输出过宽。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
