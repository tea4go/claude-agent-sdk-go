# Claude Agent SDK for Go

<div align="center">
  <img src="gopher.png" alt="Go Gopher" width="200"/>
</div>

<div align="center">

[![CI](https://github.com/tea4go/claude-agent-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/tea4go/claude-agent-sdk-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/tea4go/claude-agent-sdk-go.svg)](https://pkg.go.dev/github.com/tea4go/claude-agent-sdk-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/tea4go/claude-agent-sdk-go)](https://goreportcard.com/report/github.com/tea4go/claude-agent-sdk-go)
[![codecov](https://codecov.io/gh/tea4go/claude-agent-sdk-go/branch/main/graph/badge.svg)](https://codecov.io/gh/tea4go/claude-agent-sdk-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

</div>

Claude Code CLI 的非官方 Go SDK。它通过一套简洁、符合 Go 习惯的 API，把 Claude 的代码理解能力、安全文件操作能力和外部工具集成能力带到生产环境应用中，同时提供完善的错误处理和自动资源管理。

**针对不同场景的 2 套核心 API：**
- **Query API：** 一次性任务、自动化流程、CI/CD 集成
- **Client API：** 交互式对话、多轮工作流、流式响应
- **WithClient：** 符合 Go 习惯的上下文管理方式，自动处理资源生命周期

![Claude Agent SDK in Action](cc-sdk-go-in-action-v2.gif)

## 安装

```bash
go get github.com/tea4go/claude-agent-sdk-go
```

**前置要求：** Go 1.18+、Node.js、Claude Code（`npm install -g @anthropic-ai/claude-code`）

## 核心特性

- **两套 API 覆盖不同需求：** `Query` 适合自动化，`Client` 适合交互式场景
- **100% Python SDK 兼容：** 功能对齐 Python SDK，同时保持 Go 原生设计
- **自动资源管理：** `WithClient` 提供符合 Go 风格的上下文管理模式
- **会话管理：** 通过 `Query()` 和 `QueryWithSession()` 隔离对话上下文，并通过 `ListSessions()`、`GetSessionInfo()`、`GetSessionMessages()` 直接读取磁盘上的会话
- **技能与斜杠命令：** 支持注册进程内技能，并发现 CLI 原生斜杠命令
- **内置工具集成：** 支持文件操作、AWS、GitHub、数据库等工具
- **可用于生产环境：** 提供完善的错误处理、超时控制和资源清理
- **安全优先：** 支持细粒度工具权限和访问控制
- **上下文感知：** 可在多次交互中保持会话状态
- **高级能力：** 支持权限回调、生命周期 Hook、文件检查点等能力

## 用法

### Query API：一次性任务

适合自动化、脚本执行，以及有明确完成条件的任务：

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"

    "github.com/tea4go/claude-agent-sdk-go"
)

func main() {
    fmt.Println("Claude Agent SDK - Query API 示例")
    fmt.Println("提问：What is 2+2?")

    ctx := context.Background()

    iterator, err := claudecode.Query(ctx, "What is 2+2?")
    if err != nil {
        if cliErr := claudecode.AsCLINotFoundError(err); cliErr != nil {
            fmt.Printf("未找到 Claude CLI：%v\n", cliErr)
            fmt.Println("安装命令：npm install -g @anthropic-ai/claude-code")
            return
        }
        if connErr := claudecode.AsConnectionError(err); connErr != nil {
            fmt.Printf("连接失败：%v\n", connErr)
            return
        }
        log.Fatalf("Query 调用失败：%v", err)
    }
    defer iterator.Close()

    fmt.Println("\n响应：")

    for {
        message, err := iterator.Next(ctx)
        if err != nil {
            if errors.Is(err, claudecode.ErrNoMoreMessages) {
                break
            }
            log.Fatalf("读取消息失败：%v", err)
        }

        if message == nil {
            break
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
                    log.Printf("错误：%s", *msg.Result)
                } else {
                    log.Printf("错误：未知错误")
                }
            }
        }
    }

    fmt.Println("\n调用完成！")
}
```

### Client API：交互式与多轮场景

**`WithClient` 提供自动资源管理（等价于 Python 中的 `async with`）：**

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tea4go/claude-agent-sdk-go"
)

func main() {
    fmt.Println("Claude Agent SDK - Client 流式示例")
    fmt.Println("提问：Explain Go goroutines with a simple example")

    ctx := context.Background()
    question := "Explain what Go goroutines are and show a simple example"

    err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
        fmt.Println("\n已连接，开始流式接收：")

        if err := client.Query(ctx, question); err != nil {
            return fmt.Errorf("query 失败：%w", err)
        }

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
                            fmt.Print(textBlock.Text)
                        }
                    }
                case *claudecode.ResultMessage:
                    if msg.IsError {
                        if msg.Result != nil {
                            return fmt.Errorf("错误：%s", *msg.Result)
                        }
                        return fmt.Errorf("错误：未知错误")
                    }
                    return nil
                }
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    })

    if err != nil {
        log.Fatalf("流式调用失败：%v", err)
    }

    fmt.Println("\n\n流式调用完成！")
}
```

### 会话管理

**使用会话管理在多次查询之间保持上下文：**

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tea4go/claude-agent-sdk-go"
)

func main() {
    fmt.Println("Claude Agent SDK - 会话管理示例")

    ctx := context.Background()

    err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
        fmt.Println("\n演示隔离会话：")

        sessionA := "math-session"
        if err := client.QueryWithSession(ctx, "Remember this: x = 5", sessionA); err != nil {
            return err
        }

        sessionB := "programming-session"
        if err := client.QueryWithSession(ctx, "Remember this: language = Go", sessionB); err != nil {
            return err
        }

        fmt.Println("\n查询数学会话：")
        if err := client.QueryWithSession(ctx, "What is x * 2?", sessionA); err != nil {
            return err
        }

        fmt.Println("\n查询编程会话：")
        if err := client.QueryWithSession(ctx, "What language did I mention?", sessionB); err != nil {
            return err
        }

        fmt.Println("\n默认会话（与上述上下文隔离）：")
        return client.Query(ctx, "What did I just ask about?")
    })

    if err != nil {
        log.Fatalf("会话演示失败：%v", err)
    }

    fmt.Println("会话管理演示完成！")
}
```

**传统 Client API（仍然支持）：**

<details>
<summary>点击查看手动资源管理方式</summary>

```go
func traditionalClientExample() {
    ctx := context.Background()

    client := claudecode.NewClient()
    if err := client.Connect(ctx); err != nil {
        log.Fatalf("连接失败：%v", err)
    }
    defer client.Disconnect()

    // 使用 client...
}
```
</details>

## 工具集成与外部服务

你可以集成文件系统、云服务、数据库和开发工具：

**核心工具**（内置文件操作）：

```go
claudecode.Query(ctx, "Read all Go files and create API documentation",
    claudecode.WithAllowedTools("Read", "Write"))
```

**MCP 工具**（外部服务集成）：

```go
claudecode.Query(ctx, "List my S3 buckets and analyze their security settings",
    claudecode.WithAllowedTools("mcp__aws-api-mcp__call_aws", "mcp__aws-api-mcp__suggest_aws_commands", "Write"))
```

## 配置选项

你可以通过函数式选项来自定义 Claude 的行为：

**工具与权限控制：**

```go
claudecode.Query(ctx, prompt,
    claudecode.WithAllowedTools("Read", "Write"),
    claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits))
```

**系统行为：**

```go
claudecode.Query(ctx, prompt,
    claudecode.WithSystemPrompt("You are a senior Go developer"),
    claudecode.WithModel("claude-sonnet-4-5"),
    claudecode.WithMaxTurns(10))
```

**环境变量**（自 v0.2.5 起支持）：

```go
claudecode.NewClient(
    claudecode.WithEnv(map[string]string{
        "HTTP_PROXY":  "http://proxy.example.com:8080",
        "HTTPS_PROXY": "http://proxy.example.com:8080",
    }))

claudecode.NewClient(
    claudecode.WithEnvVar("DEBUG", "1"),
    claudecode.WithEnvVar("CUSTOM_PATH", "/usr/local/bin"))
```

**上下文与工作目录：**

```go
claudecode.Query(ctx, prompt,
    claudecode.WithCwd("/path/to/project"),
    claudecode.WithAddDirs("src", "docs"))
```

**会话管理**（Client API）：

```go
err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
    client.Query(ctx, "Remember: x = 5")
    return client.QueryWithSession(ctx, "What is x?", "math-session")
})
```

**编程式 Agent：**

```go
claudecode.Query(ctx, "Review this codebase for security issues",
    claudecode.WithAgent("security-reviewer", claudecode.AgentDefinition{
        Description: "Reviews code for security vulnerabilities",
        Prompt:      "You are a security expert focused on OWASP top 10...",
        Tools:       []string{"Read", "Grep", "Glob"},
        Model:       claudecode.AgentModelSonnet,
    }))

claudecode.Query(ctx, "Analyze and improve this code",
    claudecode.WithAgents(map[string]claudecode.AgentDefinition{
        "code-reviewer": {
            Description: "Reviews code quality and best practices",
            Prompt:      "You are a senior engineer focused on code quality...",
            Tools:       []string{"Read", "Grep"},
            Model:       claudecode.AgentModelSonnet,
        },
        "test-writer": {
            Description: "Writes comprehensive unit tests",
            Prompt:      "You are a testing expert...",
            Tools:       []string{"Read", "Write", "Bash"},
            Model:       claudecode.AgentModelHaiku,
        },
    }))
```

可用的 Agent 模型常量：`AgentModelSonnet`、`AgentModelOpus`、`AgentModelHaiku`、`AgentModelInherit`

### 技能与斜杠命令

**注册进程内技能，并通过查询接口调用：**

```go
claudecode.RegisterSkill("translate", func(ctx context.Context, args string) (string, error) {
    return "translated: " + args, nil
})

// 随后在提示词中调用
iterator, err := claudecode.Query(ctx, "/translate 你好")
```

**发现原生斜杠命令，用于前端输入建议：**

```go
commands, err := claudecode.DiscoverSlashCommands(ctx)
for _, cmd := range commands {
    fmt.Printf("/%s - %s\n", cmd.Name, cmd.Description)
}
```

### 磁盘会话读取

**无需运行中的 CLI 连接即可读取持久化会话：**

```go
// 列出所有持久化会话
sessions, err := claudecode.ListSessions()

// 查看特定会话及其消息
info, err := claudecode.GetSessionInfo(sessionID)
messages, err := claudecode.GetSessionMessages(sessionID)
```

## 文档

- [架构说明](ARCHITECTURE.md)：系统设计与组件概览
- [贡献指南](CONTRIBUTING.md)：开发环境与贡献规范
- [Python SDK 对齐情况](docs/parity.md)：与 Python SDK 的功能对比
- [pkg.go.dev](https://pkg.go.dev/github.com/tea4go/claude-agent-sdk-go)：GoDoc 在线文档

## 高级特性

SDK 提供了一系列面向生产环境的高级能力：

- **权限回调**：通过代码控制工具访问权限（[示例 11](examples/11_permission_callback/)）
- **生命周期 Hook**：拦截工具执行相关事件（[示例 12](examples/12_hooks/)）
- **文件检查点**：跟踪并回滚文件改动（[示例 13](examples/13_file_checkpointing/)）
- **SDK MCP Server**：创建进程内自定义工具（[示例 14](examples/14_sdk_mcp_server/)）
- **流式诊断**：通过 `GetStreamIssues()` 和 `GetStreamStats()` 监控流状态

完整示例请查看 [examples 目录](examples/README.md)。

## 何时使用哪套 API

**以下场景更适合 Query API：**

- 需要一次性自动化或脚本执行
- 任务完成标准明确
- 希望自动清理资源
- 正在构建 CI/CD 集成
- 更偏好简单、无状态的调用方式

**以下场景更适合 Client API（`WithClient`）：**

- 需要交互式对话
- 需要在多次请求之间保留上下文
- 正在构建复杂的多步骤工作流
- 需要实时流式响应
- 需要基于前一次结果继续迭代
- **希望自动管理资源（推荐）**

## 示例

详细说明请参见 [`examples/README.md`](examples/README.md)。

### 快速开始

| 示例 | 说明 |
|------|------|
| [`01_quickstart`](examples/01_quickstart/) | Query API 基础 |
| [`02_client_streaming`](examples/02_client_streaming/) | WithClient 流式基础 |
| [`03_client_multi_turn`](examples/03_client_multi_turn/) | 多轮对话 |

### 工具集成

| 示例 | 说明 |
|------|------|
| [`04_query_with_tools`](examples/04_query_with_tools/) | 使用 Query API 执行文件操作 |
| [`05_client_with_tools`](examples/05_client_with_tools/) | 交互式文件工作流 |
| [`06_query_with_mcp`](examples/06_query_with_mcp/) | 外部 MCP Server 集成 |
| [`07_client_with_mcp`](examples/07_client_with_mcp/) | 多轮 MCP 工作流 |

### 生产实践

| 示例 | 说明 |
|------|------|
| [`08_client_advanced`](examples/08_client_advanced/) | 错误处理与模型切换 |
| [`09_context_manager`](examples/09_context_manager/) | WithClient 与手动模式对比 |
| [`10_session_management`](examples/10_session_management/) | 会话隔离 |

### 安全与生命周期

| 示例 | 说明 |
|------|------|
| [`11_permission_callback`](examples/11_permission_callback/) | 权限回调 |
| [`12_hooks`](examples/12_hooks/) | 生命周期 Hook |
| [`13_file_checkpointing`](examples/13_file_checkpointing/) | 文件回滚能力 |
| [`14_sdk_mcp_server`](examples/14_sdk_mcp_server/) | 进程内自定义工具 |

### 高级模式

| 示例 | 说明 |
|------|------|
| [`15_programmatic_subagents`](examples/15_programmatic_subagents/) | 编程式子 Agent 定义 |
| [`16_structured_output`](examples/16_structured_output/) | 基于 JSON Schema 的结构化输出 |
| [`17_plugins`](examples/17_plugins/) | 插件配置 |
| [`18_sandbox_security`](examples/18_sandbox_security/) | 沙箱化 Bash 执行 |
| [`19_partial_streaming`](examples/19_partial_streaming/) | 实时部分流式输出 |
| [`20_debugging_and_diagnostics`](examples/20_debugging_and_diagnostics/) | 调试输出、环境变量、stderr 监控 |

### 会话与技能

| 示例 | 说明 |
|------|------|
| [`21_list_sessions`](examples/21_list_sessions/) | 从磁盘列出会话（含 git worktree） |
| [`21_skills`](examples/21_skills/) | 进程内注册的技能 |
| [`22_session_messages`](examples/22_session_messages/) | 从磁盘读取会话消息 |
| [`22_status_display`](examples/22_status_display/) | 实时状态显示 |
| [`23_skill_registry`](examples/23_skill_registry/) | 从外部注册表加载技能 |
| [`24_slash_commands`](examples/24_slash_commands/) | 发现原生斜杠命令 |

## 许可证

MIT
