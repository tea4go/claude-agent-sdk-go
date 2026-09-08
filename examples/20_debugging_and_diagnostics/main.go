// Package main demonstrates Debugging and Diagnostics features.
//
// This example shows how to configure debugging output, environment
// variables, and diagnostics for development and production monitoring.
// Debugging and diagnostics enable:
// - Capturing CLI debug output for troubleshooting
// - Setting environment variables for the subprocess
// - Monitoring stderr output in real-time
// - Checking connection health and statistics
//
// Key components:
// - WithDebugWriter: Redirect debug output to an io.Writer
// - WithDebugStderr: Convenience to send debug to os.Stderr
// - WithDebugDisabled: Disable all debug output
// - WithStderrCallback: Line-by-line stderr monitoring
// - WithEnv: Set multiple environment variables
// - WithEnvVar: Set a single environment variable
// - GetServerInfo: Get connection status information
// - GetStreamStats: Get streaming statistics
// - GetStreamIssues: Get validation issues from stream
//
// Run: go run main.go
//
// Package main 演示调试与诊断能力，包括环境变量注入、调试输出、stderr 回调
// 以及连接与流状态的诊断方法。
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	claudecode "github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Debugging and Diagnostics Example")
	fmt.Println("=====================================================")
	fmt.Println()

	// 示例 1：给 CLI 子进程注入环境变量。
	fmt.Println("--- Example 1: Environment Variables ---")
	fmt.Println("Setting custom environment variables for subprocess...")
	demonstrateEnvironmentVariables()

	// 示例 2：配置不同的调试输出去向。
	fmt.Println()
	fmt.Println("--- Example 2: Debug Output Configuration ---")
	fmt.Println("Configuring debug output destinations...")
	demonstrateDebugOutput()

	// 示例 3：注册 stderr 回调做实时监控。
	fmt.Println()
	fmt.Println("--- Example 3: Stderr Callback ---")
	fmt.Println("Setting up stderr monitoring callback...")
	demonstrateStderrCallback()

	// 示例 4：调用诊断接口查看连接和流状态。
	fmt.Println()
	fmt.Println("--- Example 4: Server Diagnostics ---")
	fmt.Println("Using diagnostics methods for health monitoring...")
	demonstrateServerDiagnostics()

	fmt.Println()
	fmt.Println("Debugging and diagnostics example completed!")
}

// demonstrateEnvironmentVariables shows WithEnv and WithEnvVar.
//
// demonstrateEnvironmentVariables 演示批量和单个环境变量的配置方式。
func demonstrateEnvironmentVariables() {
	// 先演示批量注入环境变量。
	envMap := map[string]string{
		"MY_API_KEY":     "secret-key-12345",
		"DEBUG":          "true",
		"LOG_LEVEL":      "debug",
		"CUSTOM_SETTING": "enabled",
	}

	fmt.Println("Using WithEnv for multiple variables:")
	for k, v := range envMap {
		// 输出示例时把敏感值打码，避免误导真实使用场景。
		displayValue := v
		if k == "MY_API_KEY" {
			displayValue = "***masked***"
		}
		fmt.Printf("  %s=%s\n", k, displayValue)
	}

	client := claudecode.NewClient(
		claudecode.WithEnv(envMap),
	)
	fmt.Println("Client created with environment variables")
	fmt.Println()

	// 再演示单独设置一个环境变量的便捷写法。
	fmt.Println("Using WithEnvVar for single variable:")
	fmt.Println("  SINGLE_VAR=single_value")

	client2 := claudecode.NewClient(
		claudecode.WithEnvVar("SINGLE_VAR", "single_value"),
	)
	fmt.Println("Client created with single environment variable")

	_ = client
	_ = client2
}

// demonstrateDebugOutput shows debug output configuration options.
//
// demonstrateDebugOutput 演示调试输出的几种常见落点。
func demonstrateDebugOutput() {
	// 方案 1：写入自定义 buffer，适合测试和程序内捕获。
	fmt.Println("Option 1: WithDebugWriter - capture to buffer")
	var debugBuffer bytes.Buffer
	client1 := claudecode.NewClient(
		claudecode.WithDebugWriter(&debugBuffer),
	)
	fmt.Printf("  Debug output will be written to buffer\n")
	fmt.Printf("  Buffer type: %T\n", &debugBuffer)
	_ = client1
	fmt.Println()

	// 方案 2：直接写到 stderr，适合本地调试。
	fmt.Println("Option 2: WithDebugStderr - output to stderr")
	client2 := claudecode.NewClient(
		claudecode.WithDebugStderr(),
	)
	fmt.Println("  Debug output will appear on stderr")
	_ = client2
	fmt.Println()

	// 方案 3：显式关闭调试输出。
	fmt.Println("Option 3: WithDebugDisabled - no debug output")
	client3 := claudecode.NewClient(
		claudecode.WithDebugDisabled(),
	)
	fmt.Println("  Debug output is completely suppressed")
	_ = client3
	fmt.Println()

	// 方案 4：输出到文件，适合生产问题排查。
	fmt.Println("Option 4: WithDebugWriter - output to file")
	fmt.Println("  Example: claudecode.WithDebugWriter(logFile)")
	fmt.Println("  Useful for production logging and post-mortem analysis")
}

// demonstrateStderrCallback shows real-time stderr monitoring.
//
// demonstrateStderrCallback 演示如何实时接收 CLI 的 stderr 输出。
func demonstrateStderrCallback() {
	// 回调可能并发触发，因此共享日志切片需要锁保护。
	var stderrLines []string
	var mu sync.Mutex

	// 这个回调既保存日志，也即时打印摘要。
	stderrCallback := func(line string) {
		mu.Lock()
		stderrLines = append(stderrLines, line)
		mu.Unlock()
		fmt.Printf("  [STDERR] %s\n", truncate(line, 60))
	}

	client := claudecode.NewClient(
		claudecode.WithStderrCallback(stderrCallback),
	)

	fmt.Println("Stderr callback configured")
	fmt.Println("Every line from CLI stderr will invoke the callback")
	fmt.Println()
	fmt.Println("Use cases:")
	fmt.Println("  - Real-time monitoring of CLI messages")
	fmt.Println("  - Capturing warnings and errors")
	fmt.Println("  - Building diagnostic dashboards")
	fmt.Println("  - Forwarding to logging services")

	_ = client
}

// demonstrateServerDiagnostics shows GetServerInfo and GetStreamStats.
//
// demonstrateServerDiagnostics 演示连接诊断、流统计和流问题检查。
func demonstrateServerDiagnostics() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("Connecting to demonstrate diagnostics methods...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		// 先看连接层状态，确认当前 transport 是否正常建立。
		fmt.Println()
		fmt.Println("GetServerInfo() - Connection status:")
		info, err := client.GetServerInfo(ctx)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
		} else {
			for k, v := range info {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}

		// 发一轮最小查询，给流统计和问题检测制造样本数据。
		if err := client.Query(ctx, "What is 2+2? Answer in one word."); err != nil {
			return err
		}

		// 读完这一轮响应，随后再查看统计结果。
		msgChan := client.ReceiveMessages(ctx)
		for {
			select {
			case message := <-msgChan:
				if message == nil {
					goto statsSection
				}
				switch msg := message.(type) {
				case *claudecode.AssistantMessage:
					// AssistantMessage 上的 error 是消息级错误信息，
					// 与 ResultMessage.IsError 代表的整体结果错误不是一回事。
					if msg.HasError() {
						errType := msg.GetError()
						if msg.IsRateLimited() {
							fmt.Printf("\n[rate limited] %s\n", errType)
						} else {
							fmt.Printf("\n[assistant error] %s\n", errType)
						}
					}
					for _, block := range msg.Content {
						if textBlock, ok := block.(*claudecode.TextBlock); ok {
							fmt.Printf("\nClaude: %s\n", textBlock.Text)
						}
					}
				case *claudecode.ResultMessage:
					if msg.IsError {
						if msg.Result != nil {
							return fmt.Errorf("error: %s", *msg.Result)
						}
						return fmt.Errorf("error: unknown error")
					}
					goto statsSection
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}

	statsSection:
		// 再看流统计，了解工具请求、响应等整体情况。
		fmt.Println()
		fmt.Println("GetStreamStats() - Streaming statistics:")
		stats := client.GetStreamStats()
		fmt.Printf("  Stats: %+v\n", stats)

		// 最后看流校验问题，确认是否存在缺失或异常消息。
		fmt.Println()
		fmt.Println("GetStreamIssues() - Validation issues:")
		issues := client.GetStreamIssues()
		if len(issues) == 0 {
			fmt.Println("  No issues detected (healthy stream)")
		} else {
			for i, issue := range issues {
				fmt.Printf("  %d. %+v\n", i+1, issue)
			}
		}

		return nil
	},
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
		// 这是诊断示例，没有 CLI 时给出提示即可，不强行报错退出。
		fmt.Printf("Note: %v\n", err)
		fmt.Println("(Diagnostics methods require an active connection)")
	}
}

// truncate shortens a string to maxLen characters.
//
// truncate 截断过长日志，避免示例输出过宽。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Demonstrate debug to file pattern (not executed, just shown).
//
// init 用一个不会执行的闭包演示“调试输出写文件”的典型配置方式。
func init() {
	// 这种模式适合生产环境留存调试日志。
	_ = func() {
		logFile, err := os.OpenFile("claude-debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return
		}
		defer logFile.Close()

		_ = claudecode.NewClient(
			claudecode.WithDebugWriter(logFile),
			claudecode.WithStderrCallback(func(line string) {
				// 额外把 stderr 也写入日志文件，便于统一排查。
				fmt.Fprintln(logFile, "[STDERR]", line)
			}),
		)
	}
}
