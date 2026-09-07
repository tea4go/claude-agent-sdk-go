// Package claudecode provides the Claude Agent SDK for Go.
//
// This SDK enables programmatic interaction with Claude Code CLI through two main APIs:
// - Query() for one-shot requests with automatic cleanup
// - Client for bidirectional streaming conversations
//
// The SDK follows Go-native patterns with goroutines and channels, providing
// context-first design for cancellation and timeouts.
//
// claudecode 包提供 Go 语言版的 Claude Agent SDK。
// 该 SDK 通过两个主要 API 实现对 Claude Code CLI 的编程化交互：
//   - Query()：一次性请求，自动清理资源
//   - Client：双向流式对话
//
// SDK 遵循 Go 原生的 goroutine 与 channel 模式，采用 context 优先的设计以支持取消与超时。
//
// Example usage:
//
//	import "github.com/tea4go/claude-agent-sdk-go"
//
//	// One-shot query
//	messages, err := claudecode.Query(ctx, "Hello, Claude!")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Streaming client
//	client := claudecode.NewClient(
//		claudecode.WithSystemPrompt("You are a helpful assistant"),
//	)
//	defer client.Close()
package claudecode

// Version represents the current version of the Claude Agent SDK for Go.
//
// Version 表示当前 Go 版 Claude Agent SDK 的版本号。
const Version = "0.1.0"
