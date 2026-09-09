# Module: examples

<!-- AUTO-MANAGED: module-description -->
## 用途

本目录包含一组可运行示例，用于演示 SDK 的常见使用模式。示例按复杂度编号（01 - 20），从入门到高级，覆盖 Query API、Client API、工具调用、MCP 集成以及生产实践。

<!-- END AUTO-MANAGED -->

<!-- AUTO-MANAGED: architecture -->
## 模块架构

```
examples/
├── 01_quickstart/           # Query API 基础用法
├── 02_client_streaming/     # 实时流式响应
├── 03_client_multi_turn/    # 多轮对话
├── 04_query_with_tools/     # 使用 Query API 执行文件操作
├── 05_client_with_tools/    # 交互式文件工作流；tool_use_result 元数据、SetPermissionMode
├── 06_query_with_mcp/       # MCP server 集成（Query）
├── 07_client_with_mcp/      # MCP server 集成（Client）
├── 08_client_advanced/      # 错误处理与模型切换
├── 09_context_manager/      # WithClient 模式
├── 10_session_management/   # 会话隔离
├── 11_permission_callback/  # 工具权限控制
├── 12_hooks/                # 生命周期 hooks；PreToolUse 日志、命令拦截、PostToolUse 上下文注入、通过 WithHook() 处理 PostToolUseFailure 恢复、通过 WithHook() 观察 Notification
├── 13_file_checkpointing/   # 文件回滚能力
├── 14_sdk_mcp_server/       # 进程内自定义工具；包含 3 个子示例：Calculator、Text Processor、Annotated Tool（通过 WithToolAnnotations 为 circle_area 设置 ReadOnlyHint/IdempotentHint/OpenWorldHint）
├── 15_programmatic_subagents/ # 程序化 subagent 定义
├── 16_structured_output/    # JSON Schema 约束
├── 17_plugins/              # 插件配置
├── 18_sandbox_security/     # 命令隔离
├── 19_partial_streaming/    # 实时增量更新
├── 20_debugging_and_diagnostics/ # 调试输出与健康监控
└── README.md                # 示例文档
```

<!-- END AUTO-MANAGED -->

<!-- AUTO-MANAGED: conventions -->
## 模块约定

- 每个示例都位于独立目录中，自包含且可单独运行
- 所有示例都包含可执行的 `main.go`
- 在示例目录下通过 `go run main.go` 运行
- 前置依赖会在 README.md 中注明（例如 MCP server 示例需要 `uvx`）
- 以 `WithClient` + `client.Query` + `client.ReceiveMessages(ctx)` 作为标准流式处理模式
- 对于没有便捷辅助函数的 hook 事件（如 PostToolUseFailure、Notification、SubagentStart、PermissionRequest），使用 `WithHook(eventName, toolFilter, callback)`；`WithPreToolUseHook` / `WithPostToolUseHook` 是仅有的两个便捷辅助函数
- 使用本地 `ptrTo[T any]` 辅助函数（`func ptrTo[T any](v T) *T { return &v }`）构造指针字段；示例 12 用于 `AdditionalContext *string`，示例 14 用于 `ToolAnnotations` 中的指针字段（如 `ReadOnlyHint *bool`）
- 使用 `toolUseID *string` 参数关联前后 hook 调用（例如用 `toolUseID` 作为 map key 统计耗时）；解引用前始终先判断 `if toolUseID == nil`

<!-- END AUTO-MANAGED -->

<!-- AUTO-MANAGED: dependencies -->
## 关键依赖

- 根包 `claudecode`
- 已安装 Claude CLI（`npm install -g @anthropic-ai/claude-code`）
- Go 1.18+
- 可选：MCP server 示例需要 `uvx`

<!-- END AUTO-MANAGED -->

<!-- MANUAL -->
## 备注

<!-- END MANUAL -->
