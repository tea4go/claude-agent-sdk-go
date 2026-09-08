// Package main demonstrates the Permission Callback System.
//
// This example shows how to use permission callbacks to control
// tool usage at runtime. Permission callbacks enable:
// - Security policy enforcement (allow/deny specific tools)
// - Path-based access control (restrict file writes to certain directories)
// - Audit logging of all tool usage requests
// - Dynamic permission decisions based on context
//
// IMPORTANT: Permission callbacks are only invoked for tools that would
// normally prompt the user for permission AND when using PermissionModeDefault.
//
// - Read-only operations (Read, Glob, Grep) are auto-approved and do NOT trigger callbacks
// - Write operations (Write, Edit, Bash) trigger callbacks ONLY in PermissionModeDefault
// - PermissionModeAcceptEdits auto-approves Write/Edit/Bash without invoking callbacks
// - Permission callbacks require Client API (streaming mode) - Query API does not support them
//
// The permission flow is:
// PreToolUse Hook -> Deny Rules -> Allow Rules -> Ask Rules -> Permission Mode -> canUseTool Callback
//
// NOTE: As of CLI version 1.0.x, there may be issues with permission callback responses
// not being properly processed by the CLI. If you see callbacks being invoked ([CALLBACK]
// messages) but tools still being blocked, this is a known CLI issue. See GitHub issues:
// - https://github.com/anthropics/claude-code/issues/4775
// - https://github.com/anthropics/claude-agent-sdk-python/issues/227
//
// Run: go run main.go
//
// Package main 演示权限回调系统，说明如何在运行时基于工具名、路径和上下文
// 对工具调用做放行、拒绝和审计。
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
	fmt.Println("Claude Agent SDK - Permission Callback Example")
	fmt.Println("==============================================")
	fmt.Println()

	// 示例 1：按工具名做最基础的放行和拒绝。
	fmt.Println("--- Example 1: Tool-Based Permission Control ---")
	fmt.Println("Policy: Allow Write/Bash tools, deny Edit tool")
	fmt.Println("Note: Using PermissionModeDefault to ensure callbacks are invoked")
	fmt.Println("      Read/Glob are auto-approved by CLI and don't trigger callbacks")
	fmt.Println()
	runToolFilterExample()

	// 示例 2：对 Write 工具增加路径级访问控制。
	fmt.Println()
	fmt.Println("--- Example 2: Path-Based Access Control ---")
	fmt.Println("Policy: Only allow writes to /tmp directory")
	fmt.Println("        Block filenames containing 'sensitive'")
	fmt.Println()
	runPathBasedExample()

	// 示例 3：记录所有权限请求，形成审计日志。
	fmt.Println()
	fmt.Println("--- Example 3: Audit Logging ---")
	fmt.Println("Policy: Log all tool requests for security auditing")
	fmt.Println()
	runAuditLoggingExample()

	fmt.Println()
	fmt.Println("Permission callback examples completed!")
}

// runToolFilterExample demonstrates basic tool allow/deny filtering.
//
// runToolFilterExample 演示按工具名称进行最基础的放行与拒绝。
func runToolFilterExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 根据工具名直接决策，是权限回调最直观的用法。
	permissionCallback := claudecode.WithCanUseTool(func(
		_ context.Context,
		toolName string,
		_ map[string]any,
		_ claudecode.ToolPermissionContext,
	) (claudecode.PermissionResult, error) {
		// 先打印所有回调请求，方便观察调用顺序。
		fmt.Printf("  [CALLBACK] Tool: %s\n", toolName)

		// 示例策略：允许 Bash / Write，拒绝 Edit。
		switch toolName {
		case "Bash", "Write":
			fmt.Printf("  [ALLOW] Tool: %s\n", toolName)
			return claudecode.NewPermissionResultAllow(), nil
		case "Edit":
			fmt.Printf("  [DENY]  Tool: %s - Edit not permitted in this example\n", toolName)
			return claudecode.NewPermissionResultDeny("Edit operations are not allowed"), nil
		default:
			// 其他工具这里统一放行；只读工具多数情况下本就不会触发回调。
			fmt.Printf("  [ALLOW] Tool: %s\n", toolName)
			return claudecode.NewPermissionResultAllow(), nil
		}
	})

	// 让 Claude 发起一次写文件操作，从而触发权限决策。
	fmt.Println("Asking Claude to create a test file in /tmp...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.Query(ctx, "Create a file at /tmp/sdk_permission_test.txt with the text 'Hello from SDK permission callback test!'"); err != nil {
			return err
		}

		return streamResponse(ctx, client)
	}, permissionCallback,
		claudecode.WithPermissionMode(claudecode.PermissionModeDefault), // 只有默认模式才会走权限回调。
		claudecode.WithMaxTurns(3),
		claudecode.WithCwd(exampleDir()))

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

// runPathBasedExample demonstrates path-based access control for Write operations.
//
// runPathBasedExample 演示如何基于写入路径和文件名做更细粒度的权限限制。
func runPathBasedExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 对 Write 工具额外检查 file_path，实现路径级准入控制。
	permissionCallback := claudecode.WithCanUseTool(func(
		_ context.Context,
		toolName string,
		input map[string]any,
		_ claudecode.ToolPermissionContext,
	) (claudecode.PermissionResult, error) {
		fmt.Printf("  [CALLBACK] Tool: %s\n", toolName)

		// 仅对 Write 做路径校验，其他工具保持简单策略。
		if toolName == "Write" {
			filePath, ok := input["file_path"].(string)
			if !ok {
				return claudecode.NewPermissionResultDeny("Missing file_path parameter"), nil
			}

			// 只允许写入 /tmp，避免修改任意目录。
			if !strings.HasPrefix(filePath, "/tmp/") {
				fmt.Printf("  [DENY]  Write outside /tmp: %s\n", filePath)
				return claudecode.NewPermissionResultDeny(
					fmt.Sprintf("Writes only allowed to /tmp, not: %s", filePath),
				), nil
			}

			// 进一步拦截敏感文件名，展示多条件策略组合。
			if strings.Contains(strings.ToLower(filepath.Base(filePath)), "sensitive") {
				fmt.Printf("  [DENY]  Sensitive filename blocked: %s\n", filePath)
				return claudecode.NewPermissionResultDeny("Cannot create files with 'sensitive' in name"), nil
			}

			fmt.Printf("  [ALLOW] Write to allowed path: %s\n", filePath)
			return claudecode.NewPermissionResultAllow(), nil
		}

		// 允许 Bash 便于 Claude 做结果验证。
		if toolName == "Bash" {
			fmt.Printf("  [ALLOW] Bash command allowed\n")
			return claudecode.NewPermissionResultAllow(), nil
		}

		// 其余工具默认放行。
		return claudecode.NewPermissionResultAllow(), nil
	})

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		// 先做一次合法写入，验证允许路径可以通过。
		fmt.Println("Attempt 1: Writing to /tmp/allowed_test.txt (should succeed)...")
		if err := client.Query(ctx, "Create a file at /tmp/allowed_test.txt with 'Test content'"); err != nil {
			return err
		}
		if err := streamResponse(ctx, client); err != nil {
			return err
		}

		// 再做一次命中规则的写入，验证拒绝分支是否生效。
		fmt.Println("\nAttempt 2: Writing to /tmp/sensitive_data.txt (should be blocked)...")
		if err := client.Query(ctx, "Create a file at /tmp/sensitive_data.txt with 'Secret content'"); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	}, permissionCallback,
		claudecode.WithPermissionMode(claudecode.PermissionModeDefault), // 默认模式下才能触发回调。
		claudecode.WithMaxTurns(5),
		claudecode.WithCwd(exampleDir()))

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

// runAuditLoggingExample demonstrates audit logging of all tool requests.
//
// runAuditLoggingExample 演示如何把所有工具请求记录为线程安全的审计日志。
func runAuditLoggingExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 审计日志会被回调并发写入，因此需要互斥锁保护。
	var auditLog []AuditEntry
	var auditMu sync.Mutex

	// 这个回调不做拦截，只记录所有权限请求。
	permissionCallback := claudecode.WithCanUseTool(func(
		_ context.Context,
		toolName string,
		input map[string]any,
		_ claudecode.ToolPermissionContext,
	) (claudecode.PermissionResult, error) {
		// 先构造一条审计记录，再落入共享切片。
		entry := AuditEntry{
			Timestamp: time.Now(),
			Tool:      toolName,
			Input:     input,
			Allowed:   true, // 这里只做审计，不做拦截。
		}

		// 并发追加日志时必须加锁。
		auditMu.Lock()
		auditLog = append(auditLog, entry)
		auditMu.Unlock()

		fmt.Printf("  [AUDIT] Tool: %-10s | Input keys: %v\n", toolName, mapKeys(input))

		// 审计模式下统一放行，让重点落在记录而非限制。
		return claudecode.NewPermissionResultAllow(), nil
	})

	fmt.Println("Asking Claude to create a test file (all operations will be logged)...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.Query(ctx, "Create a file at /tmp/sdk_audit_test.txt with the current timestamp, then run 'cat /tmp/sdk_audit_test.txt' to verify it."); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	}, permissionCallback,
		claudecode.WithPermissionMode(claudecode.PermissionModeDefault), // 默认模式下才能触发回调。
		claudecode.WithMaxTurns(5),
		claudecode.WithCwd(exampleDir()))

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

	// 最后汇总打印本次工具请求的审计结果。
	fmt.Println("\n--- Audit Log Summary ---")
	auditMu.Lock()
	for i, entry := range auditLog {
		status := "ALLOWED"
		if !entry.Allowed {
			status = "DENIED"
		}
		fmt.Printf("  %d. [%s] %s at %s\n",
			i+1, status, entry.Tool, entry.Timestamp.Format("15:04:05"))
	}
	fmt.Printf("Total tool requests: %d\n", len(auditLog))
	auditMu.Unlock()
}

// AuditEntry represents a logged tool usage request.
//
// AuditEntry 表示一条被记录下来的工具使用请求。
type AuditEntry struct {
	Timestamp time.Time
	Tool      string
	Input     map[string]any
	Allowed   bool
}

// streamResponse reads and displays messages from the client.
//
// streamResponse 读取客户端消息，并将回答压缩成较短摘要输出。
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
						// 示例输出尽量简短，只显示前 150 个字符。
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

// mapKeys returns the keys of a map as a slice.
//
// mapKeys 返回 map 的全部键，便于打印输入摘要。
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
