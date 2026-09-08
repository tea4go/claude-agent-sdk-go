// Package main demonstrates Programmatic Subagents configuration.
//
// This example shows how to define specialized agents programmatically
// using the SDK. Programmatic subagents enable:
// - Custom agent definitions without external configuration files
// - Specialized agents with specific tools and prompts
// - Model selection per agent (Sonnet, Haiku, Opus, or Inherit)
// - Dynamic agent configuration at runtime
//
// Key components:
// - WithAgent: Add a single agent definition
// - WithAgents: Add multiple agent definitions at once
// - AgentDefinition: Struct containing agent configuration
// - AgentModel: Constants for model selection (AgentModelSonnet, etc.)
//
// Run: go run main.go
//
// Package main 演示如何通过 SDK 在代码里直接定义 subagent，
// 以便按职责、工具集和模型拆分不同的智能体角色。
package main

import (
	"context"
	"fmt"
	"time"

	claudecode "github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Programmatic Subagents Example")
	fmt.Println("==================================================")
	fmt.Println()

	// 示例 1：先定义一个单独的代码审查 agent。
	fmt.Println("--- Example 1: Single Agent Definition ---")
	fmt.Println("Defining a code reviewer agent with specific tools and model...")
	runSingleAgentExample()

	// 示例 2：再扩展为多个职责清晰的 agent。
	fmt.Println()
	fmt.Println("--- Example 2: Multiple Agents ---")
	fmt.Println("Defining multiple specialized agents for a development workflow...")
	runMultipleAgentsExample()

	// 示例 3：展示可选的 agent 模型常量。
	fmt.Println()
	fmt.Println("--- Example 3: Agent Model Options ---")
	fmt.Println("Demonstrating available agent model constants...")
	showAgentModelOptions()

	fmt.Println()
	fmt.Println("Programmatic subagents example completed!")
}

// runSingleAgentExample demonstrates adding a single agent with WithAgent.
//
// runSingleAgentExample 演示如何注册一个单独的 agent 定义。
func runSingleAgentExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 这个 agent 专门负责代码审查，工具集也偏只读。
	codeReviewerAgent := claudecode.AgentDefinition{
		Description: "Reviews code for best practices, security issues, and style",
		Prompt:      "You are an expert code reviewer. Analyze code for bugs, security vulnerabilities, and adherence to best practices. Provide constructive feedback.",
		Tools:       []string{"Read", "Grep", "Glob"},
		Model:       claudecode.AgentModelSonnet,
	}

	fmt.Printf("Agent: code-reviewer\n")
	fmt.Printf("  Description: %s\n", codeReviewerAgent.Description)
	fmt.Printf("  Tools: %v\n", codeReviewerAgent.Tools)
	fmt.Printf("  Model: %s\n", codeReviewerAgent.Model)
	fmt.Println()

	// 把 agent 定义挂到客户端后，再让 Claude 在回答里使用它。
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		// 提示词里显式点名 agent，便于看到运行时命名效果。
		if err := client.Query(ctx, "Using the code-reviewer agent, briefly describe what a code review should check for. Keep your response under 50 words."); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	},
		claudecode.WithAgent("code-reviewer", codeReviewerAgent),
		claudecode.WithMaxTurns(2),
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
		fmt.Printf("Error: %v\n", err)
	}
}

// runMultipleAgentsExample demonstrates adding multiple agents with WithAgents.
//
// runMultipleAgentsExample 演示如何一次性注册多个不同职责的 agent。
func runMultipleAgentsExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 模拟一个小型研发流程：测试、文档、重构分别由不同 agent 承担。
	agents := map[string]claudecode.AgentDefinition{
		"test-writer": {
			Description: "Writes comprehensive unit tests for code",
			Prompt:      "You are a test engineer. Write thorough unit tests with edge cases and clear assertions.",
			Tools:       []string{"Read", "Write", "Bash"},
			Model:       claudecode.AgentModelHaiku, // 测试生成更偏速度优先。
		},
		"documentation": {
			Description: "Creates and updates code documentation",
			Prompt:      "You are a technical writer. Create clear, concise documentation with examples.",
			Tools:       []string{"Read", "Write", "Edit"},
			Model:       claudecode.AgentModelSonnet,
		},
		"refactorer": {
			Description: "Refactors code for better maintainability",
			Prompt:      "You are a refactoring expert. Improve code structure while maintaining functionality.",
			Tools:       []string{"Read", "Write", "Edit", "Grep"},
			Model:       claudecode.AgentModelInherit, // 继承父级模型，减少重复配置。
		},
	}

	fmt.Printf("Defined %d agents:\n", len(agents))
	for name, agent := range agents {
		fmt.Printf("  - %s (%s): %s\n", name, agent.Model, agent.Description)
	}
	fmt.Println()

	// 把 agent 集合整体挂进去，供 Claude 在需要时挑选使用。
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		// 让 Claude 先复述一遍当前可用 agent，确认配置生效。
		if err := client.Query(ctx, "List the specialized agents available to you and their purposes. Keep response under 75 words."); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	},
		claudecode.WithAgents(agents),
		claudecode.WithMaxTurns(2),
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
		fmt.Printf("Error: %v\n", err)
	}
}

// showAgentModelOptions displays all available agent model constants.
//
// showAgentModelOptions 输出所有可选的 agent 模型常量及其语义。
func showAgentModelOptions() {
	fmt.Println("Available AgentModel constants:")
	fmt.Printf("  - AgentModelSonnet:  %q - Claude Sonnet (balanced)\n", claudecode.AgentModelSonnet)
	fmt.Printf("  - AgentModelHaiku:   %q - Claude Haiku (fast)\n", claudecode.AgentModelHaiku)
	fmt.Printf("  - AgentModelOpus:    %q - Claude Opus (powerful)\n", claudecode.AgentModelOpus)
	fmt.Printf("  - AgentModelInherit: %q - Inherit parent model\n", claudecode.AgentModelInherit)
}

// streamResponse reads and displays messages from the client.
//
// streamResponse 读取客户端消息并直接输出助手文本。
func streamResponse(ctx context.Context, client claudecode.Client) error {
	msgChan := client.ReceiveMessages(ctx)

	for {
		select {
		case message := <-msgChan:
			if message == nil {
				fmt.Println()
				return nil
			}

			switch msg := message.(type) {
			case *claudecode.AssistantMessage:
				for _, block := range msg.Content {
					if textBlock, ok := block.(*claudecode.TextBlock); ok {
						fmt.Print(textBlock.Text)
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
