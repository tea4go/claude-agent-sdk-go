// Example 22: Status Display
//
// Demonstrates how to build a real-time status display that mirrors the
// Claude Code CLI bottom status bar. Shows how to:
//   - Track token usage (per-turn and cumulative)
//   - Display cost and duration (using local timer for real-time duration)
//   - Detect rate-limit and error states via RateLimitEventMessage
//   - Show current activity based on StreamEvent and message types
//   - Handle unknown message types via RawMessage forward compatibility
//
// Output format matches the CLI style:
//   Processing. (1m 23s • 31.2k in • 2.1k out)
//   Using tool Glob... (1m 30s • 31.2k in • 2.1k out • thinking)
//
// Run with: go run examples/22_status_display/main.go
//
// Package main 演示如何基于消息流自行实现一个接近 Claude CLI 底部状态栏的实时状态展示。

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tea4go/claude-agent-sdk-go"
)

// statusTracker accumulates state for the CLI-style status display.
//
// statusTracker 聚合构建状态栏所需的运行时信息。
type statusTracker struct {
	startTime        time.Time
	inputTokens      int
	outputTokens     int
	cacheReadTokens  int
	cacheWriteTokens int
	costUSD          float64
	currentActivity  string
	isRateLimited    bool
	rateLimitResets  time.Time
	errors           []string
	model            string
	isThinking       bool
	lastLineLen      int // 记录上一行长度，便于原地清屏重绘。
	done             bool
	finalDurationMs  int
	finalAPIDurMs    int
}

// formatTokenK formats a token count as X.Xk (matches CLI style).
//
// formatTokenK 把 token 数格式化成更接近 CLI 风格的 k 单位表示。
func formatTokenK(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	k := float64(n) / 1000.0
	if k >= 100 {
		return fmt.Sprintf("%dk", int(k))
	}
	return fmt.Sprintf("%.1fk", k)
}

// formatElapsed formats elapsed time from startTime (matches CLI style).
//
// formatElapsed 根据启动时间计算已耗时长。
func (s *statusTracker) formatElapsed() string {
	elapsed := time.Since(s.startTime)
	sec := int(elapsed.Seconds())
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	m := sec / 60
	rem := sec % 60
	return fmt.Sprintf("%dm %ds", m, rem)
}

// render outputs a single status line, overwriting the previous one.
// Uses ANSI escape codes to clear the previous line for in-place updates.
//
// render 在终端中原地刷新单行状态，模拟 CLI 状态栏效果。
func (s *statusTracker) render() {
	if s.done {
		// 收尾阶段直接正常打印，不再原地覆盖。
		fmt.Println(s.formatStatusLine())
		return
	}

	line := s.formatStatusLine()
	// 清掉上一帧，再在同一行渲染最新状态。
	if s.lastLineLen > 0 {
		// 光标回到行首并清空整行，实现原位更新。
		fmt.Printf("\r\033[K%s", line)
	} else {
		fmt.Print(line)
	}
	s.lastLineLen = len(line)
}

// formatStatusLine builds a CLI-style status string like:
//
//	Processing. (1m 23s • 31.2k in • 2.1k out)
//
// formatStatusLine 组装类似 CLI 的状态文本。
func (s *statusTracker) formatStatusLine() string {
	var parts []string

	// 1. 先生成活动描述，带句号，尽量贴近 CLI 文案风格。
	activity := activityLabel(s.currentActivity, s.isRateLimited, s.isThinking)
	parts = append(parts, activity)

	// 2. 再拼接括号中的统计数据。
	var stats []string

	// 时长始终以本地计时器为准，更新更实时。
	stats = append(stats, s.formatElapsed())

	// token 统计：输入、输出以及缓存命中信息。
	if s.inputTokens > 0 || s.outputTokens > 0 {
		tokenStr := fmt.Sprintf("%s in", formatTokenK(s.inputTokens))
		if s.outputTokens > 0 {
			tokenStr += fmt.Sprintf(" • %s out", formatTokenK(s.outputTokens))
		}
		stats = append(stats, tokenStr)
	}

	// 有缓存读取 token 时也展示出来。
	if s.cacheReadTokens > 0 {
		stats = append(stats, fmt.Sprintf("%s cache read", formatTokenK(s.cacheReadTokens)))
	}

	// 成本信息仅在可用时显示。
	if s.costUSD > 0 {
		stats = append(stats, fmt.Sprintf("$%.2f", s.costUSD))
	}

	// 被限流时插入醒目的警告状态。
	if s.isRateLimited {
		warning := "rate limited"
		if !s.rateLimitResets.IsZero() {
			warning += fmt.Sprintf(" (resets %s)", s.rateLimitResets.Format("15:04:05"))
		}
		stats = append(stats, warning)
	}

	// 思考态单独打标，便于区分“正在输出”和“正在思考”。
	if s.isThinking {
		stats = append(stats, "thinking")
	}

	// 累积到的错误统一拼到尾部。
	if len(s.errors) > 0 {
		stats = append(stats, fmt.Sprintf("error: %s", strings.Join(s.errors, "; ")))
	}

	if len(stats) > 0 {
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(stats, " • ")))
	}

	return strings.Join(parts, " ")
}

// activityLabel converts internal activity state to a CLI-style label.
//
// activityLabel 把内部状态映射成更接近 CLI 的可读标签。
func activityLabel(activity string, rateLimited bool, thinking bool) string {
	if rateLimited {
		return "Retrying..."
	}
	if thinking {
		return "Thinking..."
	}

	switch activity {
	case "calling_tool":
		return "Processing."
	case "tool_result":
		return "Processing."
	case "generating":
		return "Processing."
	case "writing":
		return "Writing."
	case "init":
		return "Starting."
	case "done":
		return "Complete."
	case "":
		return "Waiting."
	default:
		// 工具态显示为 “Using xxx...”，更贴近官方 CLI 表现。
		if strings.HasPrefix(activity, "tool:") {
			toolName := strings.TrimPrefix(activity, "tool:")
			return fmt.Sprintf("Using %s...", toolName)
		}
		return fmt.Sprintf("%s.", activity)
	}
}

func main() {
	prompt := "Analyze the codebase structure and list the main packages"
	if len(os.Args) > 1 {
		prompt = strings.Join(os.Args[1:], " ")
	}

	tracker := &statusTracker{
		startTime: time.Now(),
	}

	opts := []claudecode.Option{
		claudecode.WithAllowedTools("Read", "Glob", "Grep"),
	}

	ctx := context.Background()

	iter, err := claudecode.Query(ctx, prompt, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating query: %v\n", err)
		os.Exit(1)
	}
	defer iter.Close()

	fmt.Println() // blank line before status starts

	for {
		msg, err := iter.Next(ctx)
		if err != nil {
			if err == claudecode.ErrNoMoreMessages {
				break
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			break
		}

		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			tracker.isThinking = false
			tracker.model = m.Model

			if m.IsToolUse() {
				for _, block := range m.Content {
					if tu, ok := block.(*claudecode.ToolUseBlock); ok {
						tracker.currentActivity = "tool:" + tu.Name
						break
					}
				}
			} else {
				tracker.currentActivity = "generating"
			}

			// AssistantMessage 上的 usage 是按轮的，先做增量累计。
			if m.HasUsage() {
				tracker.inputTokens += m.Usage.InputTokens
				tracker.outputTokens += m.Usage.OutputTokens
				tracker.cacheReadTokens += m.Usage.CacheReadInputTokens
				tracker.cacheWriteTokens += m.Usage.CacheCreationInputTokens
			}

			if m.IsRateLimited() {
				tracker.isRateLimited = true
			} else {
				tracker.isRateLimited = false
				if m.HasError() {
					tracker.errors = append(tracker.errors, string(m.GetError()))
				}
			}

		case *claudecode.UserMessage:
			tracker.currentActivity = "tool_result"

		case *claudecode.SystemMessage:
			// system 消息主要用于更新可视状态，不直接展示正文。
			switch m.Subtype {
			case "init":
				tracker.currentActivity = "init"
			case "api_retry":
				tracker.isRateLimited = true
			default:
				// 其他 subtype 多为内部协议细节，不影响前台状态文案。
			}

		case *claudecode.ResultMessage:
			tracker.done = true
			tracker.currentActivity = "done"
			tracker.finalDurationMs = m.DurationMs
			tracker.finalAPIDurMs = m.DurationAPIMs

			if m.TotalCostUSD != nil {
				tracker.costUSD = *m.TotalCostUSD
			}

			// ResultMessage.Usage 才是整段对话级别的最终权威统计。
			if m.HasUsage() {
				tracker.inputTokens = m.Usage.InputTokens
				tracker.outputTokens = m.Usage.OutputTokens
				tracker.cacheReadTokens = m.Usage.CacheReadInputTokens
				tracker.cacheWriteTokens = m.Usage.CacheCreationInputTokens
			}

			if m.IsError {
				tracker.errors = append(tracker.errors, "conversation failed")
			}
			for _, e := range m.Errors {
				tracker.errors = append(tracker.errors, e)
			}

		case *claudecode.StreamEvent:
			eventType, _ := m.Event["type"].(string)
			switch eventType {
			case claudecode.StreamEventTypeContentBlockStart:
				if cb, ok := m.Event["content_block"].(map[string]any); ok {
					if t, ok := cb["type"].(string); ok {
						switch t {
						case claudecode.ContentBlockTypeToolUse:
							if name, ok := cb["name"].(string); ok {
								tracker.currentActivity = "tool:" + name
								tracker.isThinking = false
							}
						case claudecode.ContentBlockTypeThinking:
							tracker.isThinking = true
							tracker.currentActivity = "generating"
						case claudecode.ContentBlockTypeText:
							tracker.currentActivity = "writing"
							tracker.isThinking = false
						}
					}
				}
			case claudecode.StreamEventTypeMessageDelta:
				if usageRaw, ok := m.Event["usage"].(map[string]any); ok {
					if v, ok := usageRaw["output_tokens"].(float64); ok {
						tracker.outputTokens = int(v)
					}
				}
			}

		case *claudecode.RateLimitEventMessage:
			if m.IsAllowed() {
				tracker.isRateLimited = false
			} else {
				tracker.isRateLimited = true
				if m.RateLimitInfo.ResetsAt > 0 {
					tracker.rateLimitResets = time.Unix(m.RateLimitInfo.ResetsAt, 0)
				}
			}

		case *claudecode.RawMessage:
			// 保持前向兼容：未知消息类型先静默跳过。
		}

		tracker.render()
	}

	// 打印最终摘要前先清掉原地刷新的那一行。
	if tracker.lastLineLen > 0 {
		fmt.Printf("\r\033[K")
	}

	fmt.Println("\n--- Final Summary ---")
	fmt.Println(tracker.formatStatusLine())
	if tracker.finalDurationMs > 0 {
		fmt.Printf("  Total duration: %s (API: %s)\n",
			formatDuration(tracker.finalDurationMs),
			formatDuration(tracker.finalAPIDurMs))
	}
}

// formatDuration formats a millisecond duration into a readable short string.
//
// formatDuration 把毫秒时长格式化成便于终端展示的短字符串。
func formatDuration(ms int) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	s := ms / 1000
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	s = s % 60
	return fmt.Sprintf("%dm %ds", m, s)
}
