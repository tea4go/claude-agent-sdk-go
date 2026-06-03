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
type statusTracker struct {
	startTime       time.Time
	inputTokens     int
	outputTokens    int
	cacheReadTokens int
	cacheWriteTokens int
	costUSD         float64
	currentActivity string
	isRateLimited   bool
	rateLimitResets time.Time
	errors          []string
	model           string
	isThinking      bool
	lastLineLen     int // for clearing previous output
	done            bool
	finalDurationMs int
	finalAPIDurMs   int
}

// formatTokenK formats a token count as X.Xk (matches CLI style).
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
func (s *statusTracker) render() {
	if s.done {
		// Final line: just print normally, no overwriting needed
		fmt.Println(s.formatStatusLine())
		return
	}

	line := s.formatStatusLine()
	// Clear previous line and write new one in-place
	if s.lastLineLen > 0 {
		// Move cursor to start of line, clear it, then write new content
		fmt.Printf("\r\033[K%s", line)
	} else {
		fmt.Print(line)
	}
	s.lastLineLen = len(line)
}

// formatStatusLine builds a CLI-style status string like:
//   Processing. (1m 23s • 31.2k in • 2.1k out)
func (s *statusTracker) formatStatusLine() string {
	var parts []string

	// 1. Activity description (with trailing period, matching CLI style)
	activity := activityLabel(s.currentActivity, s.isRateLimited, s.isThinking)
	parts = append(parts, activity)

	// 2. Parenthesized stats
	var stats []string

	// Duration: always show elapsed time from local timer
	stats = append(stats, s.formatElapsed())

	// Token counts
	if s.inputTokens > 0 || s.outputTokens > 0 {
		tokenStr := fmt.Sprintf("%s in", formatTokenK(s.inputTokens))
		if s.outputTokens > 0 {
			tokenStr += fmt.Sprintf(" • %s out", formatTokenK(s.outputTokens))
		}
		stats = append(stats, tokenStr)
	}

	// Cache tokens (when present)
	if s.cacheReadTokens > 0 {
		stats = append(stats, fmt.Sprintf("%s cache read", formatTokenK(s.cacheReadTokens)))
	}

	// Cost
	if s.costUSD > 0 {
		stats = append(stats, fmt.Sprintf("$%.2f", s.costUSD))
	}

	// Rate limit warning
	if s.isRateLimited {
		warning := "rate limited"
		if !s.rateLimitResets.IsZero() {
			warning += fmt.Sprintf(" (resets %s)", s.rateLimitResets.Format("15:04:05"))
		}
		stats = append(stats, warning)
	}

	// Thinking indicator
	if s.isThinking {
		stats = append(stats, "thinking")
	}

	// Error indicator
	if len(s.errors) > 0 {
		stats = append(stats, fmt.Sprintf("error: %s", strings.Join(s.errors, "; ")))
	}

	if len(stats) > 0 {
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(stats, " • ")))
	}

	return strings.Join(parts, " ")
}

// activityLabel converts internal activity state to a CLI-style label.
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
		// Tool-specific activity (e.g. "Glob", "Read")
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

			// Accumulate per-turn token usage
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
			// Map system subtypes to human-readable activities
			switch m.Subtype {
			case "init":
				tracker.currentActivity = "init"
			case "api_retry":
				tracker.isRateLimited = true
			default:
				// Other system subtypes (hook_started, hook_response, etc.)
				// are internal protocol messages - don't change the visible activity
			}

		case *claudecode.ResultMessage:
			tracker.done = true
			tracker.currentActivity = "done"
			tracker.finalDurationMs = m.DurationMs
			tracker.finalAPIDurMs = m.DurationAPIMs

			if m.TotalCostUSD != nil {
				tracker.costUSD = *m.TotalCostUSD
			}

			// ResultMessage.Usage is the authoritative conversation-level total
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
			// Forward-compatible: silently handle unknown types
		}

		tracker.render()
	}

	// Clear the in-place status line before printing the final summary
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