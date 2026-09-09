# Claude Agent SDK for Go 示例

本目录提供一组可运行示例，用于演示 Claude Agent SDK for Go 的使用方式。**Query API** 和 **Client API** 都已达到生产可用水平，并与 Python SDK 完整对齐。

## 前置条件

- Go 1.18+
- Node.js
- Claude Code CLI：`npm install -g @anthropic-ai/claude-code`

## 学习路径

示例按**由易到难**排序。建议按下面的顺序学习：

### 1. 从这里开始：基础用法

```bash
# 01 - 第一次查询（最简单）
cd examples/01_quickstart
go run main.go
```

### 2. 学习流式处理

```bash
# 02 - 实时流式响应
cd examples/02_client_streaming
go run main.go

# 03 - 带上下文的多轮对话
cd examples/03_client_multi_turn
go run main.go
```

### 3. 掌握工具集成

```bash
# 04 - 使用 Query API 调用文件工具
cd examples/04_query_with_tools
go run main.go

# 05 - 使用 Client API 调用文件工具（交互式）
cd examples/05_client_with_tools
go run main.go
```

### 4. MCP 工具集成

```bash
# 06 - 使用 Query API 调用 MCP 工具（时区查询）
cd examples/06_query_with_mcp
go run main.go

# 07 - 使用 Client API 调用 MCP 工具（多轮时间工作流）
cd examples/07_client_with_mcp
go run main.go
```

### 5. 生产实践模式

```bash
# 08 - 高级错误处理与模型切换
cd examples/08_client_advanced
go run main.go

# 09 - 使用 WithClient 自动管理资源
cd examples/09_context_manager
go run main.go

# 10 - 会话管理与隔离
cd examples/10_session_management
go run main.go

# 11 - 使用权限回调控制工具访问
cd examples/11_permission_callback
go run main.go

# 12 - 生命周期 hook 系统
cd examples/12_hooks
go run main.go
```

### 6. 高级特性

```bash
# 13 - 文件检查点与回滚
cd examples/13_file_checkpointing
go run main.go

# 14 - 进程内 SDK MCP server
cd examples/14_sdk_mcp_server
go run main.go

# 15 - 程序化 subagent
cd examples/15_programmatic_subagents
go run main.go

# 16 - 类型安全的结构化输出
cd examples/16_structured_output
go run main.go

# 17 - 自定义插件集成
cd examples/17_plugins
go run main.go

# 18 - Sandbox 安全（Linux/macOS）
cd examples/18_sandbox_security
go run main.go

# 19 - 部分流式更新
cd examples/19_partial_streaming
go run main.go

# 20 - 调试与诊断
cd examples/20_debugging_and_diagnostics
go run main.go

# 23 - 外部 Skill 注册表
cd examples/23_skill_registry
go run main.go

# 24 - Slash command 建议
cd examples/24_slash_commands
go run main.go
```

## 快速测试示例

你可以先运行下面这个简单示例，确认环境配置是否正确：

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "time"

    "github.com/tea4go/claude-agent-sdk-go"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    iterator, err := claudecode.Query(ctx, "What is Go?")
    if err != nil {
        log.Fatal(err)
    }
    defer iterator.Close()

    for {
        message, err := iterator.Next(ctx)
        if err != nil {
            if errors.Is(err, claudecode.ErrNoMoreMessages) {
                break
            }
            log.Fatal(err)
        }

        if message == nil {
            break
        }

        if assistantMsg, ok := message.(*claudecode.AssistantMessage); ok {
            for _, block := range assistantMsg.Content {
                if textBlock, ok := block.(*claudecode.TextBlock); ok {
                    fmt.Print(textBlock.Text)
                }
            }
        }
    }
}
```

## 示例说明

### 🟢 入门级

#### `01_quickstart/` - 第一次查询
- **核心概念**：基础 Query API、消息处理
- **包含特性**：简单查询、system prompt、消息处理
- **预计耗时**：2 分钟

#### `02_client_streaming/` - 实时流式处理
- **核心概念**：Client API、流式响应
- **包含特性**：连接管理、实时处理
- **预计耗时**：5 分钟

#### `03_client_multi_turn/` - 多轮对话
- **核心概念**：上下文保持、多轮对话
- **包含特性**：追问、会话管理
- **预计耗时**：5 分钟

### 🟡 中级

#### `04_query_with_tools/` - 文件操作
- **核心概念**：工具集成、文件处理
- **包含特性**：Read/Write/Edit 工具、安全限制
- **预计耗时**：10 分钟

#### `05_client_with_tools/` - 交互式文件工作流
- **核心概念**：多轮工具使用、渐进式开发
- **包含特性**：交互式文件处理、跨工具保留上下文
- **预计耗时**：10 分钟

#### `06_query_with_mcp/` - MCP 工具集成
- **核心概念**：MCP 工具、外部服务集成
- **包含特性**：通过 MCP time server 执行时区查询
- **前置依赖**：`uvx`（用于 `mcp-server-time`）
- **预计耗时**：10 分钟

### 🔴 高级

#### `07_client_with_mcp/` - 多轮 MCP 工作流
- **核心概念**：多步骤 MCP 操作、上下文保持
- **包含特性**：跨时区时间转换、多轮工作流
- **前置依赖**：`uvx`（用于 `mcp-server-time`）
- **预计耗时**：10 分钟

#### `08_client_advanced/` - 高级 Client 特性
- **核心概念**：动态模型切换、结构化错误处理
- **包含特性**：`SetModel()`、类型化错误检查、带模型切换的多轮对话
- **预计耗时**：15 分钟

#### `09_context_manager/` - 资源管理模式
- **核心概念**：`WithClient` 模式与手动连接管理的对比
- **包含特性**：自动资源清理、错误处理对比
- **预计耗时**：10 分钟

#### `10_session_management/` - 会话隔离
- **核心概念**：会话管理、对话隔离
- **包含特性**：默认会话与自定义会话、`QueryWithSession()`、上下文隔离
- **预计耗时**：10 分钟

#### `11_permission_callback/` - 权限回调
- **核心概念**：工具权限控制、安全策略
- **包含特性**：允许/拒绝工具执行、基于路径的访问控制、审计日志
- **预计耗时**：15 分钟

#### `12_hooks/` - 生命周期 Hook 系统
- **核心概念**：生命周期 hook、事件拦截
- **包含特性**：PreToolUse/PostToolUse hook、命令拦截、上下文注入
- **预计耗时**：15 分钟

#### `13_file_checkpointing/` - 文件检查点与回滚
- **核心概念**：文件状态管理、检查点/回滚操作
- **包含特性**：`WithFileCheckpointing()`、`RewindFiles()`、消息 UUID
- **预计耗时**：15 分钟

#### `14_sdk_mcp_server/` - 进程内 SDK MCP Server
- **核心概念**：自定义 MCP 工具、进程内工具执行
- **包含特性**：`NewTool()`、`CreateSDKMcpServer()`、`WithSdkMcpServer()`
- **预计耗时**：20 分钟

### 专家级

#### `15_programmatic_subagents/` - 程序化 subagent
- **核心概念**：agent 定义、专用 agent
- **包含特性**：`WithAgent()`、`WithAgents()`、`AgentDefinition`、`AgentModel` 常量
- **预计耗时**：15 分钟

#### `16_structured_output/` - 类型安全的结构化输出
- **核心概念**：JSON Schema 约束、结构化响应
- **包含特性**：`WithJSONSchema()`、`WithOutputFormat()`、`ResultMessage.StructuredOutput`
- **预计耗时**：15 分钟

#### `17_plugins/` - 插件配置
- **核心概念**：插件集成、可扩展性
- **包含特性**：`WithLocalPlugin()`、`WithPlugins()`、`SdkPluginConfig`
- **预计耗时**：10 分钟

#### `18_sandbox_security/` - Sandbox 安全
- **核心概念**：命令隔离、安全边界
- **包含特性**：`WithSandboxEnabled()`、`WithSandboxNetwork()`、排除命令
- **平台**：仅限 Linux/macOS
- **预计耗时**：15 分钟

#### `19_partial_streaming/` - 部分流式输出
- **核心概念**：实时更新、渐进式渲染
- **包含特性**：`WithPartialStreaming()`、`StreamEvent` 类型、增量处理
- **预计耗时**：15 分钟

#### `20_debugging_and_diagnostics/` - 调试与诊断
- **核心概念**：调试输出、环境配置、健康监控
- **包含特性**：`WithDebugWriter()`、`WithStderrCallback()`、`GetServerInfo()`
- **预计耗时**：15 分钟

#### `23_skill_registry/` - 外部 Skill 注册表
- **核心概念**：从共享注册目录加载指定 Skill
- **包含特性**：`WithSkillRegistry()`、临时插件包装器、限定范围的 Skill 工具
- **预计耗时**：10 分钟

#### `24_slash_commands/` - Slash command 建议
- **核心概念**：原生 slash command 发现、前端输入建议
- **包含特性**：`DiscoverSlashCommands()`、`SlashCommand`
- **预计耗时**：5 分钟

## 常见模式

### Query API - 一次性操作
```go
// 简单查询
iterator, err := claudecode.Query(ctx, "Explain Go interfaces")

// 带 system prompt
iterator, err := claudecode.Query(ctx, "Review this code",
    claudecode.WithSystemPrompt("You are a senior Go developer"))

// 带工具
iterator, err := claudecode.Query(ctx, "Analyze all files",
    claudecode.WithAllowedTools("Read", "Write"))
```

### Client API - 多轮对话

**WithClient 模式（推荐）：**
```go
// 自动资源管理
err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
    // 第一个问题
    client.Query(ctx, "What is dependency injection?")
    // Process response...

    // 追问（保留上下文）
    return client.Query(ctx, "Show me a Go example")
})
```

**手动模式（仍然支持）：**
```go
client := claudecode.NewClient()
defer client.Disconnect()

// 第一个问题
client.Query(ctx, "What is dependency injection?")
// Process response...

// 追问（保留上下文）
client.Query(ctx, "Show me a Go example")
// Process response...
```

### 会话管理 - 更清晰的 API

**新的清晰版 Query API（推荐）：**
```go
// 默认会话
client.Query(ctx, "What is dependency injection?")

// 自定义会话
client.QueryWithSession(ctx, "Remember this context", "my-session")

// 不同会话之间彼此隔离
client.Query(ctx, "What did I just say?") // 不会记住 "my-session" 中的上下文
```

**新 API 的优势：**
- ✅ **意图清晰**：`Query()` 与 `QueryWithSession()` 职责明确
- ✅ **类型安全**：避免可变参数带来的歧义
- ✅ **与 Python 对齐**：对应 Python SDK 的 `client.query(session_id="...")`
- ✅ **符合 Go 习惯**：沿用 `WithContext()`、`WithTimeout()` 这类标准库模式

### MCP 工具 - 云端集成
```go
// AWS 操作（必须显式列出工具名，不支持通配符）
iterator, err := claudecode.Query(ctx, "List my S3 buckets",
    claudecode.WithAllowedTools(
        "mcp__aws-api-mcp__call_aws",
        "mcp__aws-api-mcp__suggest_aws_commands"))
```

### 工具预设
```go
// 显式列表 - 控制力最强
claudecode.WithAllowedTools("Read", "Write", "Edit")

// 预设 - 使用方便（完整 Claude Code 工具集）
claudecode.WithClaudeCodeTools()

// 自定义预设
claudecode.WithToolsPreset("my_custom_preset")
```

## 错误处理

```go
iterator, err := claudecode.Query(ctx, "test")
if err != nil {
    // 使用 As* 辅助函数提取带字段的类型化错误
    if cliErr := claudecode.AsCLINotFoundError(err); cliErr != nil {
        fmt.Printf("CLI not found at: %s\n", cliErr.Path)
        fmt.Println("Please install: npm install -g @anthropic-ai/claude-code")
        return
    }
    if connErr := claudecode.AsConnectionError(err); connErr != nil {
        fmt.Printf("Connection failed: %v\n", connErr)
        return
    }
    log.Fatal(err)
}
```

## 什么时候该用哪种 API

### 🎯 适合选择 Query API 的场景
- 一次性提问或命令
- 批处理任务
- CI/CD 脚本
- 简单自动化
- 资源开销更低

### 🔄 适合选择 Client API 的场景
- 多轮对话
- 交互式应用
- 依赖上下文的工作流
- 需要实时流式处理
- 更复杂的状态管理

## 需要帮助？

- 安装说明请参考 [主 README](../README.md)
- 建议先从 `01_quickstart` 开始掌握基础模式
- 最佳学习方式是按编号顺序逐步查看示例
- SDK 的设计模式遵循 [Python SDK](https://docs.anthropic.com/en/docs/claude-code/sdk)
