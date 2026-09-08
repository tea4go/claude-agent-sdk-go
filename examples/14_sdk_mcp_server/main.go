// Package main demonstrates the In-Process SDK MCP Server feature.
//
// This example shows how to create custom tools that run within your
// Go application (no external subprocess needed). SDK MCP servers enable:
// - Custom domain-specific tools (calculators, data transformers, etc.)
// - In-memory computations without subprocess overhead
// - Full control over tool implementation with Go-native code
// - Integration with existing Go libraries and data structures
//
// Key components:
//   - NewTool: Creates tool definitions (Go alternative to Python's @tool decorator)
//   - WithToolAnnotations: Attaches MCP-spec behavioral hints to a tool
//     (title, readOnlyHint, destructiveHint, idempotentHint, openWorldHint)
//   - CreateSDKMcpServer: Creates an MCP server instance with tools
//   - WithSdkMcpServer: Adds the server to the client configuration
//   - Tool naming: mcp__<server_name>__<tool_name> format for AllowedTools
//
// Run: go run main.go
//
// Package main 演示进程内 SDK MCP Server，说明如何直接在 Go 进程里定义工具、
// 组装服务器，并通过标准 MCP 接口暴露给 Claude 使用。
package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	claudecode "github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - In-Process SDK MCP Server Example")
	fmt.Println("====================================================")
	fmt.Println()

	// 示例 1：构造一个最简单的计算器工具集。
	fmt.Println("--- Example 1: Calculator Server ---")
	fmt.Println("Tools: add (adds two numbers), sqrt (square root)")
	fmt.Println()
	runCalculatorExample()

	// 示例 2：演示字符串处理类工具。
	fmt.Println()
	fmt.Println("--- Example 2: Text Processing Server ---")
	fmt.Println("Tools: uppercase, reverse, word_count")
	fmt.Println()
	runTextProcessorExample()

	// 示例 3：演示如何给工具补充 MCP 规范里的行为提示。
	fmt.Println()
	fmt.Println("--- Example 3: Annotated Tool ---")
	fmt.Println("Tools: circle_area (read-only, idempotent, closed-world hints)")
	fmt.Println()
	runAnnotatedToolExample()

	fmt.Println()
	fmt.Println("SDK MCP Server examples completed!")
}

// ptrTo returns a pointer to v. Used to populate the pointer fields on
// ToolAnnotations (Title *string, ReadOnlyHint *bool, etc.). A nil pointer
// means "field not set" and is omitted from the wire format; *false vs nil
// is a meaningful distinction for the CLI.
//
// ptrTo 返回值对应的指针，便于构造 ToolAnnotations 中的大量可选字段。
func ptrTo[T any](v T) *T { return &v }

// runCalculatorExample demonstrates a calculator with math tools.
//
// runCalculatorExample 演示如何把一组数学函数打包成进程内 MCP 工具。
func runCalculatorExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// add：最基础的双数求和工具。
	addTool := claudecode.NewTool(
		"add",
		"Add two numbers together and return the sum",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{
					"type":        "number",
					"description": "First number to add",
				},
				"b": map[string]any{
					"type":        "number",
					"description": "Second number to add",
				},
			},
			"required": []string{"a", "b"},
		},
		func(_ context.Context, args map[string]any) (*claudecode.McpToolResult, error) {
			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			result := a + b
			fmt.Printf("  [TOOL] add(%v, %v) = %v\n", a, b, result)
			return &claudecode.McpToolResult{
				Content: []claudecode.McpContent{
					{Type: "text", Text: fmt.Sprintf("%.2f + %.2f = %.2f", a, b, result)},
				},
			}, nil
		},
	)

	// sqrt：附带一个简单的负数校验，演示工具错误返回。
	sqrtTool := claudecode.NewTool(
		"sqrt",
		"Calculate the square root of a number",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"n": map[string]any{
					"type":        "number",
					"description": "Number to find square root of",
				},
			},
			"required": []string{"n"},
		},
		func(_ context.Context, args map[string]any) (*claudecode.McpToolResult, error) {
			n, _ := args["n"].(float64)
			if n < 0 {
				fmt.Printf("  [TOOL] sqrt(%v) = ERROR (negative number)\n", n)
				return &claudecode.McpToolResult{
					Content: []claudecode.McpContent{
						{Type: "text", Text: "Error: Cannot calculate square root of negative number"},
					},
					IsError: true,
				}, nil
			}
			result := math.Sqrt(n)
			fmt.Printf("  [TOOL] sqrt(%v) = %v\n", n, result)
			return &claudecode.McpToolResult{
				Content: []claudecode.McpContent{
					{Type: "text", Text: fmt.Sprintf("sqrt(%.2f) = %.4f", n, result)},
				},
			}, nil
		},
	)

	// 把多个工具组装成一个进程内 MCP 服务器实例。
	calculator := claudecode.CreateSDKMcpServer("calculator", "1.0.0", addTool, sqrtTool)

	fmt.Println("Asking Claude to perform calculations...")

	// 通过 WithSdkMcpServer 把该服务器挂到客户端配置里。
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.Query(ctx, "Using the calculator tools, calculate 15 + 27, then find the square root of 144. Show the results."); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	},
		claudecode.WithSdkMcpServer("calc", calculator),
		claudecode.WithAllowedTools("mcp__calc__add", "mcp__calc__sqrt"),
		claudecode.WithMaxTurns(5),
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

// runTextProcessorExample demonstrates text processing tools.
//
// runTextProcessorExample 演示字符串处理工具的定义方式。
func runTextProcessorExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// uppercase：把输入文本转成大写。
	uppercaseTool := claudecode.NewTool(
		"uppercase",
		"Convert text to uppercase",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{
					"type":        "string",
					"description": "Text to convert to uppercase",
				},
			},
			"required": []string{"text"},
		},
		func(_ context.Context, args map[string]any) (*claudecode.McpToolResult, error) {
			text, _ := args["text"].(string)
			result := strings.ToUpper(text)
			fmt.Printf("  [TOOL] uppercase(%q) = %q\n", text, result)
			return &claudecode.McpToolResult{
				Content: []claudecode.McpContent{
					{Type: "text", Text: result},
				},
			}, nil
		},
	)

	// reverse：对 rune 切片原地翻转，兼容 Unicode 字符。
	reverseTool := claudecode.NewTool(
		"reverse",
		"Reverse a string",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{
					"type":        "string",
					"description": "Text to reverse",
				},
			},
			"required": []string{"text"},
		},
		func(_ context.Context, args map[string]any) (*claudecode.McpToolResult, error) {
			text, _ := args["text"].(string)
			runes := []rune(text)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			result := string(runes)
			fmt.Printf("  [TOOL] reverse(%q) = %q\n", text, result)
			return &claudecode.McpToolResult{
				Content: []claudecode.McpContent{
					{Type: "text", Text: result},
				},
			}, nil
		},
	)

	// word_count：演示一个简单统计型工具。
	wordCountTool := claudecode.NewTool(
		"word_count",
		"Count the number of words in text",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{
					"type":        "string",
					"description": "Text to count words in",
				},
			},
			"required": []string{"text"},
		},
		func(_ context.Context, args map[string]any) (*claudecode.McpToolResult, error) {
			text, _ := args["text"].(string)
			words := strings.Fields(text)
			count := len(words)
			fmt.Printf("  [TOOL] word_count(%q) = %d\n", text, count)
			return &claudecode.McpToolResult{
				Content: []claudecode.McpContent{
					{Type: "text", Text: fmt.Sprintf("Word count: %d", count)},
				},
			}, nil
		},
	)

	// 把三个文本工具组合成同一个 MCP 服务。
	textProcessor := claudecode.CreateSDKMcpServer(
		"textproc", "1.0.0",
		uppercaseTool, reverseTool, wordCountTool,
	)

	fmt.Println("Asking Claude to process text...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.Query(ctx, "Using the text processing tools: 1) Convert 'hello world' to uppercase, 2) Reverse 'Claude Code', 3) Count words in 'The quick brown fox jumps over the lazy dog'."); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	},
		claudecode.WithSdkMcpServer("text", textProcessor),
		claudecode.WithAllowedTools("mcp__text__uppercase", "mcp__text__reverse", "mcp__text__word_count"),
		claudecode.WithMaxTurns(5),
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

// runAnnotatedToolExample demonstrates attaching MCP-spec ToolAnnotations to
// a tool. The annotations are advisory hints sent to the CLI in the JSONRPC
// tools/list response under the "annotations" key. The key is omitted entirely
// when a tool has no annotations (matching Python's exclude_none semantics).
//
// runAnnotatedToolExample 演示给工具附加只读、幂等等行为提示。
func runAnnotatedToolExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// circle_area 是纯计算：不读外部状态、没有副作用、相同输入必得相同输出，
	// 因此适合标记为只读、幂等、闭世界（非 open world）。
	circleAreaTool := claudecode.NewTool(
		"circle_area",
		"Compute the area of a circle given its radius",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"radius": map[string]any{
					"type":        "number",
					"description": "Radius of the circle (non-negative)",
				},
			},
			"required": []string{"radius"},
		},
		func(_ context.Context, args map[string]any) (*claudecode.McpToolResult, error) {
			radius, _ := args["radius"].(float64)
			if radius < 0 {
				return &claudecode.McpToolResult{
					Content: []claudecode.McpContent{
						{Type: "text", Text: "Error: radius must be non-negative"},
					},
					IsError: true,
				}, nil
			}
			area := math.Pi * radius * radius
			fmt.Printf("  [TOOL] circle_area(%v) = %.4f\n", radius, area)
			return &claudecode.McpToolResult{
				Content: []claudecode.McpContent{
					{Type: "text", Text: fmt.Sprintf("Area of circle with radius %.2f = %.4f", radius, area)},
				},
			}, nil
		},
		claudecode.WithToolAnnotations(&claudecode.ToolAnnotations{
			Title:          ptrTo("Circle Area Calculator"),
			ReadOnlyHint:   ptrTo(true),
			IdempotentHint: ptrTo(true),
			OpenWorldHint:  ptrTo(false),
		}),
	)

	geometry := claudecode.CreateSDKMcpServer("geometry", "1.0.0", circleAreaTool)

	fmt.Println("Asking Claude to compute a circle area...")

	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		if err := client.Query(ctx, "Use the circle_area tool to compute the area of a circle with radius 7."); err != nil {
			return err
		}
		return streamResponse(ctx, client)
	},
		claudecode.WithSdkMcpServer("geo", geometry),
		claudecode.WithAllowedTools("mcp__geo__circle_area"),
		claudecode.WithMaxTurns(5),
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

// streamResponse reads and displays messages from the client.
//
// streamResponse 读取客户端消息，并把回答压缩成适合示例展示的摘要。
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
						// 只展示前 200 个字符，避免示例输出过长。
						text := textBlock.Text
						if len(text) > 200 {
							text = text[:200] + "..."
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
