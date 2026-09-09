# 功能对齐：Go SDK 与 Python SDK

本文全面对比 Go Agent SDK 与 Python Agent SDK，说明两者已实现 100% 功能对齐。

---

## 执行摘要

**状态：已实现 100% 功能对齐**

Go SDK（`github.com/tea4go/claude-agent-sdk-go`）实现了 Python SDK（`claude-agent-sdk`）的全部功能，并额外提供了更符合 Go 习惯的增强能力。

| 类别 | Python SDK | Go SDK | 对齐情况 |
|:-----|:-----------|:-------|:---------|
| 函数 | 3 | 4（+ 辅助函数） | 100% |
| Client 方法 | 7 | 13（+ 扩展方法） | 100% |
| 消息类型 | 5 | 6 | 100% |
| 内容块类型 | 4 | 4 | 100% |
| 错误类型 | 5 | 6 | 100% |
| Hook 事件 | 6 | 6 | 100% |
| 选项字段 | 30+ | 70+ 个构造函数 | 100% |
| MCP 类型 | 5 | 9（+ 扩展类型） | 100% |
| Sandbox 配置 | 3 种类型 | 3 种类型 | 100% |

---

## 函数

| Python SDK | Go SDK | 说明 |
|:-----------|:-------|:-----|
| `query(prompt, options)` | `Query(ctx, prompt, opts...)` | 采用 Context-first 模式 |
| `tool(name, desc, schema)` | `NewTool(name, desc, schema, handler)` | 工厂函数替代装饰器 |
| `create_sdk_mcp_server(name, version, tools)` | `CreateSDKMcpServer(name, version, tools...)` | 功能完全一致 |

### Go SDK 额外函数

| 函数 | 说明 |
|:-----|:-----|
| `QueryWithTransport()` | 使用自定义 transport 执行查询（适用于测试） |
| `NewClient()` | 创建新的 Client |
| `NewClientWithTransport()` | 使用自定义 transport 创建 Client |
| `WithClient()` | 资源管理辅助函数（类似 `async with`） |
| `WithClientTransport()` | 带自定义 transport 的资源管理辅助函数 |

---

## 类 / Client 接口

### Python：`ClaudeSDKClient`

| 方法 | Go 对应项 | 说明 |
|:-----|:----------|:-----|
| `__init__(options)` | `NewClient(opts...)` | 函数式选项模式 |
| `connect(prompt)` | `Connect(ctx, prompt...)` | 采用 Context-first 模式 |
| `query(prompt, session_id)` | `Query(ctx, prompt)` / `QueryWithSession(ctx, prompt, sessionID)` | 拆分为两个方法 |
| `receive_messages()` | `ReceiveMessages(ctx)` | 返回 channel |
| `receive_response()` | `ReceiveResponse(ctx)` | 返回 `MessageIterator` |
| `interrupt()` | `Interrupt(ctx)` | 控制协议请求；连接仍可继续使用 |
| `rewind_files(uuid)` | `RewindFiles(ctx, messageUUID)` | 采用 Context-first 模式 |
| `disconnect()` | `Disconnect()` | 功能一致 |
| `async with` 上下文管理器 | `WithClient()` 辅助函数 | 更符合 Go 习惯的资源管理方式 |

### Go SDK 额外方法

| 方法 | 说明 |
|:-----|:-----|
| `SetModel(ctx, model)` | 在运行时切换模型 |
| `SetPermissionMode(ctx, mode)` | 在运行时切换权限模式 |
| `GetStreamIssues()` | 获取流校验问题 |
| `GetStreamStats()` | 获取流统计信息 |
| `GetServerInfo(ctx)` | 获取诊断信息 |

---

## 配置选项

### ClaudeAgentOptions 映射

| Python 选项 | Go 选项构造函数 | 状态 |
|:------------|:----------------|:-----|
| `allowed_tools` | `WithAllowedTools(tools...)` | PARITY |
| `disallowed_tools` | `WithDisallowedTools(tools...)` | PARITY |
| `tools` | `WithTools(tools...)` | PARITY |
| - | `WithToolsPreset(preset)` | GO EXTRA |
| - | `WithClaudeCodeTools()` | GO EXTRA |
| `system_prompt` | `WithSystemPrompt(prompt)` | PARITY |
| - | `WithAppendSystemPrompt(prompt)` | GO EXTRA |
| `model` | `WithModel(model)` | PARITY |
| `fallback_model` | `WithFallbackModel(model)` | PARITY |
| `max_turns` | `WithMaxTurns(turns)` | PARITY |
| `max_budget_usd` | `WithMaxBudgetUSD(budget)` | PARITY |
| `max_thinking_tokens` | `WithMaxThinkingTokens(tokens)` | PARITY |
| `permission_mode` | `WithPermissionMode(mode)` | PARITY |
| `permission_prompt_tool_name` | `WithPermissionPromptToolName(toolName)` | PARITY |
| `continue_conversation` | `WithContinueConversation(bool)` | PARITY |
| `resume` | `WithResume(sessionID)` | PARITY |
| `fork_session` | `WithForkSession(fork)` | PARITY |
| `cwd` | `WithCwd(cwd)` | PARITY |
| `add_dirs` | `WithAddDirs(dirs...)` | PARITY |
| `mcp_servers` | `WithMcpServers(servers)` | PARITY |
| - | `WithSdkMcpServer(name, server)` | GO EXTRA |
| `settings` | `WithSettings(settings)` | PARITY |
| `setting_sources` | `WithSettingSources(sources...)` | PARITY |
| `env` | `WithEnv(env)` | PARITY |
| - | `WithEnvVar(key, value)` | GO EXTRA |
| `extra_args` | `WithExtraArgs(args)` | PARITY |
| `cli_path` | `WithCLIPath(path)` | PARITY |
| `max_buffer_size` | `WithMaxBufferSize(size)` | PARITY |
| `stderr` | `WithStderrCallback(callback)` | PARITY |
| `debug_stderr`（已弃用） | `WithDebugWriter(w)` | PARITY |
| - | `WithDebugStderr()` | GO EXTRA |
| - | `WithDebugDisabled()` | GO EXTRA |
| `can_use_tool` | `WithCanUseTool(callback)` | PARITY |
| `hooks` | `WithHooks(hooks)` | PARITY |
| - | `WithHook(event, matcher, callback)` | GO EXTRA |
| - | `WithPreToolUseHook(matcher, callback)` | GO EXTRA |
| - | `WithPostToolUseHook(matcher, callback)` | GO EXTRA |
| `user` | `WithUser(user)` | PARITY |
| `include_partial_messages` | `WithIncludePartialMessages(include)` | PARITY |
| - | `WithPartialStreaming()` | GO EXTRA |
| `enable_file_checkpointing` | `WithEnableFileCheckpointing(enable)` | PARITY |
| - | `WithFileCheckpointing()` | GO EXTRA |
| `agents` | `WithAgents(agents)` | PARITY |
| - | `WithAgent(name, agent)` | GO EXTRA |
| `plugins` | `WithPlugins(plugins)` | PARITY |
| - | `WithPlugin(plugin)` | GO EXTRA |
| - | `WithLocalPlugin(path)` | GO EXTRA |
| `sandbox` | `WithSandbox(sandbox)` | PARITY |
| - | `WithSandboxEnabled(enabled)` | GO EXTRA |
| - | `WithAutoAllowBashIfSandboxed(autoAllow)` | GO EXTRA |
| - | `WithSandboxExcludedCommands(commands...)` | GO EXTRA |
| - | `WithSandboxNetwork(network)` | GO EXTRA |
| `output_format` | `WithOutputFormat(format)` | PARITY |
| - | `WithJSONSchema(schema)` | GO EXTRA |
| `betas` | `WithBetas(betas...)` | PARITY |

---

## 消息类型

| Python SDK | Go SDK | 状态 |
|:-----------|:-------|:-----|
| `Message`（联合类型） | `Message` 接口 | PARITY |
| `UserMessage` | `UserMessage` 结构体 | PARITY |
| `AssistantMessage` | `AssistantMessage` 结构体 | PARITY |
| `SystemMessage` | `SystemMessage` 结构体 | PARITY |
| `ResultMessage` | `ResultMessage` 结构体 | PARITY |
| `StreamEvent` | `StreamEvent` 结构体 | PARITY |
| - | `RawControlMessage` 结构体 | GO EXTRA |

### 消息类型常量

| Python | Go | 状态 |
|:-------|:---|:-----|
| `"user"` | `MessageTypeUser` | PARITY |
| `"assistant"` | `MessageTypeAssistant` | PARITY |
| `"system"` | `MessageTypeSystem` | PARITY |
| `"result"` | `MessageTypeResult` | PARITY |
| `"stream_event"` | `MessageTypeStreamEvent` | PARITY |
| - | `MessageTypeControlRequest` | GO EXTRA |
| - | `MessageTypeControlResponse` | GO EXTRA |

---

## 内容块类型

| Python SDK | Go SDK | 状态 |
|:-----------|:-------|:-----|
| `ContentBlock`（联合类型） | `ContentBlock` 接口 | PARITY |
| `TextBlock` | `TextBlock` 结构体 | PARITY |
| `ThinkingBlock` | `ThinkingBlock` 结构体 | PARITY |
| `ToolUseBlock` | `ToolUseBlock` 结构体 | PARITY |
| `ToolResultBlock` | `ToolResultBlock` 结构体 | PARITY |

### 内容块类型常量

| Python | Go | 状态 |
|:-------|:---|:-----|
| `"text"` | `ContentBlockTypeText` | PARITY |
| `"thinking"` | `ContentBlockTypeThinking` | PARITY |
| `"tool_use"` | `ContentBlockTypeToolUse` | PARITY |
| `"tool_result"` | `ContentBlockTypeToolResult` | PARITY |

---

## 错误类型

| Python SDK | Go SDK | 状态 |
|:-----------|:-------|:-----|
| `ClaudeSDKError` | `SDKError` 接口 + `BaseError` | PARITY |
| `CLIConnectionError` | `ConnectionError` | PARITY |
| `CLINotFoundError` | `CLINotFoundError` | PARITY |
| `ProcessError` | `ProcessError` | PARITY |
| `CLIJSONDecodeError` | `JSONDecodeError` | PARITY |
| `MessageParseError` | `MessageParseError` | PARITY |

### AssistantMessageError 类型

| Python | Go | 状态 |
|:-------|:---|:-----|
| `"authentication_failed"` | `AssistantMessageErrorAuthFailed` | PARITY |
| `"billing_error"` | `AssistantMessageErrorBilling` | PARITY |
| `"rate_limit"` | `AssistantMessageErrorRateLimit` | PARITY |
| `"invalid_request"` | `AssistantMessageErrorInvalidRequest` | PARITY |
| `"server_error"` | `AssistantMessageErrorServer` | PARITY |
| `"unknown"` | `AssistantMessageErrorUnknown` | PARITY |

### Go 特有的错误类型辅助函数

Go SDK 提供了符合 Go 习惯的辅助函数，设计风格类似 `os.IsNotExist`。这些函数支持处理被包装的错误（内部使用 `errors.As`）。

| 函数 | 说明 | 状态 |
|:-----|:-----|:-----|
| `IsConnectionError(err)` | 判断错误是否为 `ConnectionError` | GO-NATIVE |
| `IsCLINotFoundError(err)` | 判断错误是否为 `CLINotFoundError` | GO-NATIVE |
| `IsProcessError(err)` | 判断错误是否为 `ProcessError` | GO-NATIVE |
| `IsJSONDecodeError(err)` | 判断错误是否为 `JSONDecodeError` | GO-NATIVE |
| `IsMessageParseError(err)` | 判断错误是否为 `MessageParseError` | GO-NATIVE |
| `AsConnectionError(err)` | 提取 `*ConnectionError`，失败时返回 `nil` | GO-NATIVE |
| `AsCLINotFoundError(err)` | 提取 `*CLINotFoundError`，失败时返回 `nil` | GO-NATIVE |
| `AsProcessError(err)` | 提取 `*ProcessError`，失败时返回 `nil` | GO-NATIVE |
| `AsJSONDecodeError(err)` | 提取 `*JSONDecodeError`，失败时返回 `nil` | GO-NATIVE |
| `AsMessageParseError(err)` | 提取 `*MessageParseError`，失败时返回 `nil` | GO-NATIVE |

**说明**：Python 使用 `isinstance()` 检查错误类型。Go SDK 提供这些辅助函数，作为比手动类型断言更符合 Go 习惯的替代方案。

---

## Hook 类型

### Hook 事件

| Python | Go | 状态 |
|:-------|:---|:-----|
| `"PreToolUse"` | `HookEventPreToolUse` | PARITY |
| `"PostToolUse"` | `HookEventPostToolUse` | PARITY |
| `"UserPromptSubmit"` | `HookEventUserPromptSubmit` | PARITY |
| `"Stop"` | `HookEventStop` | PARITY |
| `"SubagentStop"` | `HookEventSubagentStop` | PARITY |
| `"PreCompact"` | `HookEventPreCompact` | PARITY |

### Hook 类型定义

| Python SDK | Go SDK | 状态 |
|:-----------|:-------|:-----|
| `HookEvent` | `HookEvent` 类型 | PARITY |
| `HookCallback` | `HookCallback` 类型 | PARITY |
| `HookContext` | `HookContext` 结构体 | PARITY |
| `HookMatcher` | `HookMatcher` 结构体 | PARITY |
| `HookJSONOutput` | `HookJSONOutput` 结构体 | PARITY |
| `AsyncHookJSONOutput` | `AsyncHookJSONOutput` 结构体 | PARITY |

### Hook 输入类型

| Python | Go | 状态 |
|:-------|:---|:-----|
| `BaseHookInput` | `BaseHookInput` | PARITY |
| `PreToolUseHookInput` | `PreToolUseHookInput` | PARITY |
| `PostToolUseHookInput` | `PostToolUseHookInput` | PARITY |
| `UserPromptSubmitHookInput` | `UserPromptSubmitHookInput` | PARITY |
| `StopHookInput` | `StopHookInput` | PARITY |
| `SubagentStopHookInput` | `SubagentStopHookInput` | PARITY |
| `PreCompactHookInput` | `PreCompactHookInput` | PARITY |

### Hook 输出类型

| Python | Go | 状态 |
|:-------|:---|:-----|
| `PreToolUseHookSpecificOutput` | `PreToolUseHookSpecificOutput` | PARITY |
| `PostToolUseHookSpecificOutput` | `PostToolUseHookSpecificOutput` | PARITY |
| `UserPromptSubmitHookSpecificOutput` | `UserPromptSubmitHookSpecificOutput` | PARITY |

---

## MCP 类型

| Python SDK | Go SDK | 状态 |
|:-----------|:-------|:-----|
| `SdkMcpTool` | `McpTool` 结构体 | PARITY |
| `McpServerConfig`（联合类型） | `McpServerConfig` 接口 | PARITY |
| `McpStdioServerConfig` | `McpStdioServerConfig` | PARITY |
| `McpSSEServerConfig` | `McpSSEServerConfig` | PARITY |
| `McpHttpServerConfig` | `McpHTTPServerConfig` | PARITY |
| `McpSdkServerConfig` | `McpSdkServerConfig` | PARITY |

### Go SDK 的 MCP 扩展

| 类型 | 说明 |
|:-----|:-----|
| `McpServer` 接口 | MCP server 的抽象接口 |
| `SdkMcpServer` 结构体 | 进程内 server 实现 |
| `McpToolHandler` | 工具处理函数类型 |
| `McpToolResult` | 工具执行结果 |
| `McpContent` | 工具结果中的内容 |
| `McpToolDefinition` | 用于工具列表的定义类型 |

---

## 权限类型

| Python | Go | 状态 |
|:-------|:---|:-----|
| `CanUseTool` | `CanUseToolCallback` | PARITY |
| `ToolPermissionContext` | `ToolPermissionContext` | PARITY |
| `PermissionResult` | `PermissionResult` 接口 | PARITY |
| `PermissionResultAllow` | `PermissionResultAllow` 结构体 | PARITY |
| `PermissionResultDeny` | `PermissionResultDeny` 结构体 | PARITY |
| `PermissionUpdate` | `PermissionUpdate` 结构体 | PARITY |
| `PermissionRuleValue` | `PermissionRuleValue` 结构体 | PARITY |

### 权限模式

| Python | Go | 状态 |
|:-------|:---|:-----|
| `"default"` | `PermissionModeDefault` | PARITY |
| `"acceptEdits"` | `PermissionModeAcceptEdits` | PARITY |
| `"plan"` | `PermissionModePlan` | PARITY |
| `"bypassPermissions"` | `PermissionModeBypassPermissions` | PARITY |

---

## Sandbox 配置

| Python SDK | Go SDK | 状态 |
|:-----------|:-------|:-----|
| `SandboxSettings` | `SandboxSettings` 结构体 | PARITY |
| `SandboxNetworkConfig` | `SandboxNetworkConfig` 结构体 | PARITY |
| `SandboxIgnoreViolations` | `SandboxIgnoreViolations` 结构体 | PARITY |

### SandboxSettings 字段

| Python | Go | 状态 |
|:-------|:---|:-----|
| `enabled` | `Enabled` | PARITY |
| `autoAllowBashIfSandboxed` | `AutoAllowBashIfSandboxed` | PARITY |
| `excludedCommands` | `ExcludedCommands` | PARITY |
| `allowUnsandboxedCommands` | `AllowUnsandboxedCommands` | PARITY |
| `network` | `Network` | PARITY |
| `ignoreViolations` | `IgnoreViolations` | PARITY |
| `enableWeakerNestedSandbox` | `EnableWeakerNestedSandbox` | PARITY |

### SandboxNetworkConfig 字段

| Python | Go | 状态 |
|:-------|:---|:-----|
| `allowLocalBinding` | `AllowLocalBinding` | PARITY |
| `allowUnixSockets` | `AllowUnixSockets` | PARITY |
| `allowAllUnixSockets` | `AllowAllUnixSockets` | PARITY |
| `httpProxyPort` | `HTTPProxyPort` | PARITY |
| `socksProxyPort` | `SOCKSProxyPort` | PARITY |

---

## 高级特性

| 特性 | Python SDK | Go SDK | 状态 |
|:-----|:-----------|:-------|:-----|
| 流式响应 | `async for message in query()` | `MessageIterator.Next(ctx)` | PARITY |
| 部分消息流式输出 | `include_partial_messages=True` | `WithPartialStreaming()` | PARITY |
| 会话管理 | `resume`、`fork_session` | `WithResume()`、`WithForkSession()` | PARITY |
| 文件检查点 | `enable_file_checkpointing` | `WithFileCheckpointing()` | PARITY |
| 文件回滚 | `rewind_files(uuid)` | `RewindFiles(ctx, uuid)` | PARITY |
| 中断支持 | `interrupt()` | `Interrupt(ctx)` | PARITY（所有平台都支持控制协议） |
| 结构化输出 | `output_format` | `WithOutputFormat()`、`WithJSONSchema()` | PARITY |
| 自定义 agent | `agents` | `WithAgents()`、`WithAgent()` | PARITY |
| 插件 | `plugins` | `WithPlugins()`、`WithLocalPlugin()` | PARITY |
| Beta 特性 | `betas` | `WithBetas()` | PARITY |

### Go SDK 的高级扩展

| 特性 | 说明 |
|:-----|:-----|
| `Transport` 接口 | 便于测试的自定义 transport 抽象 |
| `StreamValidator` | 用于流校验和诊断 |
| `GetStreamIssues()` | 获取流问题列表 |
| `GetStreamStats()` | 获取流统计信息 |
| `SetModel()` | 在运行时切换模型 |
| `SetPermissionMode()` | 在运行时切换权限模式 |
| `GetServerInfo()` | 获取诊断信息 |

---

## 其他类型

### Agent 类型

| Python | Go | 状态 |
|:-------|:---|:-----|
| `AgentDefinition` dataclass | `AgentDefinition` 结构体 | PARITY |
| `"sonnet"` | `AgentModelSonnet` | PARITY |
| `"opus"` | `AgentModelOpus` | PARITY |
| `"haiku"` | `AgentModelHaiku` | PARITY |
| `"inherit"` | `AgentModelInherit` | PARITY |

### Plugin 类型

| Python | Go | 状态 |
|:-------|:---|:-----|
| `SdkPluginConfig` TypedDict | `SdkPluginConfig` 结构体 | PARITY |
| `"local"` | `SdkPluginTypeLocal` | PARITY |

### Setting Source 类型

| Python | Go | 状态 |
|:-------|:---|:-----|
| `"user"` | `SettingSourceUser` | PARITY |
| `"project"` | `SettingSourceProject` | PARITY |
| `"local"` | `SettingSourceLocal` | PARITY |

### Beta 特性

| Python | Go | 状态 |
|:-------|:---|:-----|
| `"context-1m-2025-08-07"` | `SdkBetaContext1M` | PARITY |

---

## 面向 Python SDK 用户的迁移指南

### 关键差异

1. **Context-first 模式**：Go 函数将 `context.Context` 作为第一个参数，用于取消和超时控制。

2. **函数式选项**：Go 不使用单一的 options 对象，而是通过 `With*()` 函数进行配置。

3. **接口替代类**：Go 使用接口（`Client`、`Message`、`ContentBlock`），而不是类。

4. **错误处理方式不同**：Go 使用显式错误返回值，而不是异常机制。

5. **资源管理方式不同**：使用 `WithClient()` 代替 `async with`，以实现自动资源清理。

### 代码对比

**Python：**
```python
from claude_agent_sdk import query, ClaudeAgentOptions

options = ClaudeAgentOptions(
    system_prompt="You are an expert",
    allowed_tools=["Read", "Write"],
    permission_mode="acceptEdits"
)

async for message in query(prompt="Hello", options=options):
    if isinstance(message, AssistantMessage):
        print(message.content)
```

**Go：**
```go
import "github.com/tea4go/claude-agent-sdk-go"

iterator, err := claudecode.Query(ctx, "Hello",
    claudecode.WithSystemPrompt("You are an expert"),
    claudecode.WithAllowedTools("Read", "Write"),
    claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits),
)
if err != nil {
    return err
}
defer iterator.Close()

for {
    message, err := iterator.Next(ctx)
    if errors.Is(err, claudecode.ErrNoMoreMessages) {
        break
    }
    if assistant, ok := message.(*claudecode.AssistantMessage); ok {
        // Process content
    }
}
```

### Client 用法对比

**Python：**
```python
async with ClaudeSDKClient(options) as client:
    await client.query("Hello")
    async for msg in client.receive_response():
        print(msg)
```

**Go：**
```go
err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
    if err := client.Query(ctx, "Hello"); err != nil {
        return err
    }
    for msg := range client.ReceiveMessages(ctx) {
        // Process message
    }
    return nil
}, opts...)
```

---

## 结论

Go SDK 在与 Python SDK 完整对齐的同时，还提供了更符合 Go 生态的扩展能力：

- **已实现 100% Python SDK 功能**
- **使用函数式选项模式**，配置方式更灵活
- **采用 Context-first 设计**，便于取消和超时控制
- **基于接口设计**，更利于测试
- **提供额外诊断能力**（`StreamValidator`、`GetStreamIssues`、`GetStreamStats`）
- **支持运行时配置切换**（`SetModel`、`SetPermissionMode`）
- **支持自定义 transport**，便于测试

Go SDK 已达到生产可用水平，适合在 Go 环境中构建需要集成 Claude Code 的应用。
