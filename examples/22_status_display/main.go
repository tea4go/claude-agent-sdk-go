// Example 22: Status Display
//
// Demonstrates how to build a real-time status display similar to the
// Claude Code CLI bottom status bar. Shows how to:
//   - Track token usage (per-turn and cumulative)
//   - Display cost and duration from the ResultMessage
//   - Detect rate-limit and error states via RateLimitEventMessage
//   - Show current activity based on StreamEvent and message types
//   - Handle unknown message types via RawMessage forward compatibility
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

// statusTracker accumulates state for the status display.
type statusTracker struct {
	turns           int
	inputTokens     int
	outputTokens    int
	cacheReadTokens int
	cacheWriteTokens int
	costUSD         float64
	durationMs      int
	apiDurationMs   int
	currentActivity string
	isRateLimited   bool
	rateLimitResets time.Time
	errors          []string
	model           string
	stopReason      string
}

func (s *statusTracker) render() {
	var b strings.Builder

	// Activity indicator
	activity := s.currentActivity
	if activity == "" {
		activity = "idle"
	}

	// Rate limit warning
	if s.isRateLimited {
		until := ""
		if !s.rateLimitResets.IsZero() {
			until = fmt.Sprintf(" (resets %s)", s.rateLimitResets.Format("15:04:05"))
		}
		fmt.Fprintf(&b, "RATE LIMITED%s | ", until)
	}

	// Core status line
	fmt.Fprintf(&b, "%s", activity)

	if s.model != "" {
		fmt.Fprintf(&b, " | %s", s.model)
	}

	if s.turns > 0 {
		fmt.Fprintf(&b, " | turn %d", s.turns)
	}

	if s.inputTokens > 0 || s.outputTokens > 0 {
		fmt.Fprintf(&b, " | tokens: %dk in / %dk out",
			s.inputTokens/1000, s.outputTokens/1000)
	}

	if s.cacheReadTokens > 0 || s.cacheWriteTokens > 0 {
		fmt.Fprintf(&b, " cache:%dk/%dk",
			s.cacheReadTokens/1000, s.cacheWriteTokens/1000)
	}

	if s.costUSD > 0 {
		fmt.Fprintf(&b, " | $%.4f", s.costUSD)
	}

	if s.durationMs > 0 {
		fmt.Fprintf(&b, " | %s", formatDuration(s.durationMs))
	}

	if s.stopReason != "" {
		fmt.Fprintf(&b, " | stop: %s", s.stopReason)
	}

	if len(s.errors) > 0 {
		fmt.Fprintf(&b, " | ERRORS: %s", strings.Join(s.errors, "; "))
	}

	fmt.Fprintf(&b, "\n")
	fmt.Print(b.String())
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
	return fmt.Sprintf("%dm%ds", m, s)
}

func main() {
	prompt := "Analyze the codebase structure and list the main packages"
	if len(os.Args) > 1 {
		prompt = strings.Join(os.Args[1:], " ")
	}

	tracker := &statusTracker{}

	opts := []claudecode.Option{
		claudecode.WithAllowedTools("Read", "Glob", "Grep"),
	}

	// Enable partial streaming so we get StreamEvent messages for real-time
	// activity tracking.
	ctx := context.Background()

	iter, err := claudecode.Query(ctx, prompt, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating query: %v\n", err)
		os.Exit(1)
	}
	defer iter.Close()

	for {
		msg, err := iter.Next(ctx)
		if err != nil {
			if err == claudecode.ErrNoMoreMessages {
				break
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			break
		}

		// Type-switch on all message types the SDK can produce.
		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			tracker.turns++
			tracker.model = m.Model
			tracker.currentActivity = "generating response"
			tracker.stopReason = m.GetStopReason()

			if m.IsToolUse() {
				// Find the tool being used for more specific status
				for _, block := range m.Content {
					if tu, ok := block.(*claudecode.ToolUseBlock); ok {
						tracker.currentActivity = fmt.Sprintf("using tool: %s", tu.Name)
						break
					}
				}
			}

			// Accumulate per-turn token usage if available
			if m.HasUsage() {
				tracker.inputTokens += m.Usage.InputTokens
				tracker.outputTokens += m.Usage.OutputTokens
				tracker.cacheReadTokens += m.Usage.CacheReadInputTokens
				tracker.cacheWriteTokens += m.Usage.CacheCreationInputTokens
			}

			if m.IsRateLimited() {
				tracker.isRateLimited = true
				tracker.currentActivity = "rate limited - retrying"
			} else if m.HasError() {
				tracker.errors = append(tracker.errors, string(m.GetError()))
			}

		case *claudecode.UserMessage:
			tracker.currentActivity = "processing tool result"

		case *claudecode.SystemMessage:
			// System messages carry subtypes like "init", "result"
			// that indicate the current phase.
			tracker.currentActivity = fmt.Sprintf("system: %s", m.Subtype)

		case *claudecode.ResultMessage:
			// Final summary - this is the authoritative cost/duration.
			tracker.currentActivity = "done"
			tracker.durationMs = m.DurationMs
			tracker.apiDurationMs = m.DurationAPIMs

			if m.TotalCostUSD != nil {
				tracker.costUSD = *m.TotalCostUSD
			}

			// ResultMessage.Usage is the conversation-level total;
			// prefer it over our accumulated per-turn counts.
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
			// Partial streaming events provide fine-grained activity tracking.
			eventType, _ := m.Event["type"].(string)
			switch eventType {
			case claudecode.StreamEventTypeContentBlockStart:
				if cb, ok := m.Event["content_block"].(map[string]any); ok {
					if t, ok := cb["type"].(string); ok {
						switch t {
						case claudecode.ContentBlockTypeToolUse:
							if name, ok := cb["name"].(string); ok {
								tracker.currentActivity = fmt.Sprintf("calling tool: %s", name)
							}
						case claudecode.ContentBlockTypeThinking:
							tracker.currentActivity = "thinking"
						case claudecode.ContentBlockTypeText:
							tracker.currentActivity = "writing response"
						}
					}
				}
			case claudecode.StreamEventTypeContentBlockDelta:
				// Delta events are very frequent; no need to update status on each.
			case claudecode.StreamEventTypeMessageStart:
				tracker.currentActivity = "starting message"
			case claudecode.StreamEventTypeMessageDelta:
				// message_delta can carry updated Usage and StopReason.
				if delta, ok := m.Event["delta"].(map[string]any); ok {
					if sr, ok := delta["stop_reason"].(string); ok {
						tracker.stopReason = sr
					}
				}
				if usageRaw, ok := m.Event["usage"].(map[string]any); ok {
					if v, ok := usageRaw["output_tokens"].(float64); ok {
						tracker.outputTokens = int(v)
					}
				}
			case claudecode.StreamEventTypeMessageStop:
				tracker.currentActivity = "message complete"
			}

		case *claudecode.RateLimitEventMessage:
			if m.IsAllowed() {
				tracker.isRateLimited = false
			} else {
				tracker.isRateLimited = true
				tracker.currentActivity = "rate limited - retrying"
				if m.RateLimitInfo.ResetsAt > 0 {
					tracker.rateLimitResets = time.Unix(m.RateLimitInfo.ResetsAt, 0)
				}
			}

		case *claudecode.RawMessage:
			// Forward-compatible: handle unknown message types from future
			// CLI versions without SDK upgrades.
			fmt.Printf("[unknown type: %s]\n", m.MessageType)
		}

		tracker.render()
	}

	// Final summary
	fmt.Println("\n--- Final Summary ---")
	tracker.render()
}
