# AGENTS.md

本文件用于为 Codex（Codex.ai/code）在本仓库中工作时提供指导。

<!-- AUTO-MANAGED: project-description -->

## 概述

**Codex Agent SDK for Go**：用于集成 Codex CLI 的非官方 Go SDK。通过 `Query()`（一次性调用）和 `Client`（流式交互）API 提供编程式访问能力，并与 Python SDK 保持 100% 功能对齐。

- **模块名：** `github.com/tea4go/Codex-agent-sdk-go`
- **包名：** `Codex`
- **Go 版本：** 1.18+

<!-- END AUTO-MANAGED -->

<!-- AUTO-MANAGED: build-commands -->

## 构建与开发命令

```bash
# 构建与测试
go build ./...                    # 构建所有包
go test ./...                     # 运行全部测试
go test -race ./...               # 检测竞态条件
go test -cover ./...              # 覆盖率分析
make test-cover                   # 运行覆盖率测试并生成 HTML 报告

# 特定测试模式
go test -v -run TestClient        # 运行 client 相关测试（详细输出）
go test -count=3 -run TestClient  # 多次运行测试以验证稳定性
make bench                        # 运行基准测试

# 代码质量检查（提交前运行）
gofmt -s -w .                     # 格式化代码（CI 使用 `gofmt -s`；单独执行 `go fmt ./...` 不会带 `-s`，会导致 CI 失败）
go vet ./...                      # 静态分析
golangci-lint run                 # 综合 lint 检查
gocyclo -over 15 .                # 圈复杂度检查

# Makefile 目标（推荐）
make check                        # 运行全部检查（fmt、vet、lint、cyclo）
make cyclo                        # 显示复杂函数（阈值：15）
make cyclo-check                  # 当复杂度超出阈值时失败（CI）
make fmt-check                    # 校验代码格式
make security                     # 运行安全漏洞检查
make sdk-test                     # 以 SDK 使用者的方式测试 SDK
make release-check                # 发布前校验
make ci                           # 在本地运行完整 CI 流程
```

<!-- END AUTO-MANAGED -->

<!-- AUTO-MANAGED: architecture -->

## 架构

```
.
├── client.go              # Client 接口以及 WithClient/WithClientTransport 上下文管理器
├── query.go               # Query API（一次性操作）
├── errors.go              # 结构化错误类型
├── transport.go           # Transport 接口抽象
├── types.go               # 对外重新导出内部类型与常量
├── options.go             # Options 类型与函数式选项
├── options_bench_test.go  # Options 性能基准测试
├── internal/
│   ├── cli/               # CLI 发现与命令构建
│   ├── control/           # 双向控制协议（hooks、permissions、MCP）
│   ├── parser/            # 带推测式解析的 JSON 消息解析
│   ├── shared/            # 共享类型（Message、ContentBlock 接口）
│   └── subprocess/        # 子进程管理与协议适配器
├── examples/              # 使用示例（按复杂度编号）
└── docs/
    ├── architecture/      # 详细架构文档
    └── tracking/          # Python SDK 对齐跟踪（PR 回放追踪）
```

**数据流：**

1. `Query()` / `Client` -> `Transport` 接口 -> `subprocess.Transport` -> Codex CLI
2. CLI stdout -> `parser.Parser` -> `shared.Message` 类型 -> 用户代码
3. 控制协议：`control.Protocol` <-> CLI（hooks、permissions、MCP）

**文档说明：** 详细的设计模式、接口、数据流和贡献指南请参见 `ARCHITECTURE.md` 与 `CONTRIBUTING.md`。

<!-- END AUTO-MANAGED -->

<!-- AUTO-MANAGED: conventions -->

## 代码约定

- **Go 风格优先：** 使用 `gofmt` 格式化，并遵循标准命名约定
- **接口驱动：** 所有消息类型都实现 `Message`，所有内容块都实现 `ContentBlock`
- **错误处理：** 使用带 `%w` 的 `fmt.Errorf` 包装错误，并补充上下文信息
- **Context 优先：** 所有阻塞函数都以 `context.Context` 作为第一个参数
- **JSON 处理：** 对联合类型使用自定义 `UnmarshalJSON`，通过 `"type"` 字段区分分支
- **圈复杂度：** 函数复杂度应控制在 15 以内（由 gocyclo 衡量）；对确实合理超限的场景（大型表驱动测试、复杂编排逻辑）可使用 `//nolint:gocyclo`；优先通过提取辅助函数（例如把 `buildToolsListResult` 从 `routeMcpMethod` 中拆出）来控制分发逻辑复杂度，而不是依赖 nolint；表驱动测试、示例、编排代码和方法分发可以接受更高复杂度
- **命名模式：** 接口描述行为，实现使用具体名称，选项使用 `WithXxx()`，错误使用 `XxxError` 后缀
- **避免不必要导出：** 标识符只有在外部调用方需要时才导出

<!-- END AUTO-MANAGED -->

<!-- AUTO-MANAGED: patterns -->

## 已识别模式

- **Transport interface：** CLI 通信的核心抽象；测试时使用 `MockTransport`
- **进程清理：** 采用 SIGTERM -> 等待 5 秒 -> SIGKILL 的处理模式
- **缓冲保护：** 默认 1 MB 上限以防止内存耗尽；`parser.NewWithSize(n)` 可用于覆盖默认值
- **环境变量：** 通过设置 `CLAUDE_CODE_ENTRYPOINT` 标识 SDK 来源
- **表驱动测试：** 适合包含多个测试用例的复杂场景
- **函数式选项：** 使用 `WithXxx()` 形式进行配置
- **基准测试：** 使用 `var sink any` 防止死代码消除，并始终调用 `b.ReportAllocs()` 与 `b.ResetTimer()`
- **tool_use_result 元数据：** `UserMessage.ToolUseResult` 承载丰富的编辑信息（`filePath`、`structuredPatch`、`diffs`）；访问前先用 `HasToolUseResult()` 检查，再通过 `GetToolUseResult()` 读取
- **parent_tool_use_id 位置：** 对 `UserMessage`、`AssistantMessage` 与 `StreamEvent`，该字段都从顶层 JSON 数据解析（而非嵌套在 `message` 对象内）；用于识别由子 agent（Agent / Task 工具）内部生成的消息
- **AssistantMessage error 字段：** `AssistantMessage.Error` 是从顶层 `data["error"]` 解析出的 `*AssistantMessageError`（不是 `data["message"]["error"]`）；CLI 线协议格式为 `{"type":"assistant","error":"rate_limit","message":{...}}`；使用 `HasError()` 判断是否存在，用 `IsRateLimited()` 判断是否为限流错误；`AssistantMessageError` 常量与 Python SDK 完全一致：`rate_limit`、`billing_error`、`server_error`、`authentication_failed`、`invalid_request`、`unknown`
- **初始化错误路由：** `subprocess.routeInitError()` 会检测在 transport 建立连接前到达的错误 `ResultMessage`，并调用 `protocol.HandleControlInitErr()`，通过 `initErrChan` 解阻塞 `SendControlRequest()`
- **控制协议委派：** `SetModel()`、`SetPermissionMode()`、`RewindFiles()`、`GetMcpStatus()` 都会先检查 `t.connected && !t.closeStdin`，随后再委派给 `t.protocol`；若 protocol 为 nil，会返回描述性错误：`"internal error: transport connected but control protocol is nil"`
- **MCP 配置序列化：** `generateMcpConfigFile()` 在构建 CLI 配置时会去掉 SDK server 上的 Go `Instance` 字段，并显式传递 `AlwaysLoad`（而不是依赖 struct json tag）
- **权限建议：** `ToolPermissionContext.Suggestions []PermissionUpdate` 承载 CLI 提供给 `CanUseTool` 回调的权限建议；`PermissionUpdate.Type` 的取值包括 `addRules`、`replaceRules`、`removeRules`、`setMode`、`addDirectories`、`removeDirectories`
- **测试 mock 辅助：** `newClientMockTransport()` / `newQueryMockTransport()` 配合函数式选项（如 `WithQueryAssistantResponse`、`WithQueryMultipleMessages`）；`QueryWithTransport()` 用于注入 transport 的 Query 测试
- **构造函数约定：** `NewGetMcpStatusRequest()` 遵循 `NewPermissionResultAllow/Deny` 的模式，会设置必需的 `Subtype` 字段（`SubtypeGetMcpStatus = "mcp_status"`，不是 `"get_mcp_status"`）；对于 subtype 固定的控制请求类型，应优先使用构造函数
- **McpServerConfigType 常量：** `McpServerConfigTypeStdio` / `SSE` / `HTTP` / `SDK` / `ClaudeAI` 会在根包 `types.go` 中重新导出，并伴随 `McpServerConnectionStatus` 常量一起提供；通过 `McpServerStatusConfig.Type` 字段区分配置类型
- **McpServerStatus 条件字段：** 仅当 `Status == McpServerConnectionStatusConnected` 时，`ServerInfo` 非 nil；仅当 `Status == McpServerConnectionStatusFailed` 时，`Error` 非 nil；仅在连接成功时填充 `Tools`
- **streamErrChan 汇聚：** `ClientImpl.streamErrChan chan error`（缓冲区大小为 1）用于接收 `QueryStream` goroutine 产生的错误；它会与 transport 的 `errChan` 一起传给 `clientIterator`；`Next()` 会同时 select 二者，因此调用方无需额外起 goroutine 也能收到全部流式错误
- **prepareOptions()：** 先应用默认值，再做校验；当设置了 `CanUseTool` 回调时，会自动将 `PermissionPromptToolName` 配置为 `"stdio"`；该函数由 `validateOptions()` 更名而来，以体现其“默认值填充 + 校验”的双重职责
- **ReceiveResponse() 断连行为：** 在未连接状态下调用时，会返回一个非 nil 的 `clientIterator`，其中 `msgChan` 已关闭，`errChan` 为空但非 nil；因此调用方始终可以安全 range 读取
- **PostToolUseFailureHookInput：** 与 `PostToolUseHookInput` 不同；字段包括：`ToolUseID string`、`Error string`、`IsInterrupt *bool json:"is_interrupt,omitempty"`；`IsInterrupt` 为 nil 表示 JSON 中没有该键（对应 Python `NotRequired[bool]` 语义）；`PostToolUseFailureHookSpecificOutput` 在结构上与 `PostToolUseHookSpecificOutput` 一致，仅 `HookEventName` 字面量不同，二者都包含 `AdditionalContext *string`（omitempty）；当前 hook 事件数为 10 个（包含 Python SDK PR #545 引入的 Notification、SubagentStart、PermissionRequest）；`_SubagentContextMixin` 字段（`agent_id` / `agent_type`）被延后到 Phase 2 item #13（Python PR #628）中引入并应用到 PostToolUseFailureHookInput；HookEvent 常量块顺序与 Python SDK 保持一致：PreToolUse、PostToolUse、PostToolUseFailure、UserPromptSubmit、Stop、SubagentStop、PreCompact、Notification、SubagentStart、PermissionRequest
- **WithHook() 通用 API：** 对没有便捷辅助函数的 Hook 事件（如 `PostToolUseFailure`、`Notification`、`SubagentStart`、`PermissionRequest`）使用 `WithHook(eventName, toolFilter, callback)`；便捷函数 `WithPreToolUseHook` / `WithPostToolUseHook` 只适用于 PreToolUse 和 PostToolUse；规范示例见 `examples/12_hooks` 的 Example 4（PostToolUseFailure）与 Example 5（Notification）
- **HookSpecificOutput 类型：** `PostToolUseHookSpecificOutput` 与 `PostToolUseFailureHookSpecificOutput` 都要求设置 `HookEventName` 字段（json tag 为 `hookEventName`）；`AdditionalContext *string`（omitempty）用于向 Codex 下一轮注入上下文；在示例中推荐使用本地 `ptrTo[T any]` 辅助函数（`func ptrTo[T any](v T) *T { return &v }`）构造 `*string`；`PermissionRequestHookSpecificOutput.Decision map[string]any` 是唯一一个没有 `omitempty` 的输出字段，它是必填的（与 Python 的 required dict 保持一致）
- **PostToolUseFailure IsInterrupt 判空：** 必须使用 `failInput.IsInterrupt != nil && *failInput.IsInterrupt` 进行判断；nil 表示 JSON 键缺失，而不是 false；若为 true，应跳过恢复上下文注入，以尊重用户的停止意图
- **SubagentStopHookInput agent 字段：** `AgentID`、`AgentTranscriptPath`、`AgentType` 在 Python SDK PR #545 中以扁平必填字符串字段加入（并非来自 `_SubagentContextMixin`）；mixin 是在 Python PR #628 / Phase 2 item #13 才单独引入并应用到四个工具生命周期输入类型上的
- **getAnySlice 辅助函数：** 位于 `internal/control/hooks.go`，语义与 `getMap` 对齐；当键缺失时返回 nil，以保留 Python `NotRequired` 语义；适用于 Python 字段类型为 `list[Any]` 且采用 `NotRequired` 语义的场景（例如 `PermissionRequestHookInput.PermissionSuggestions`）
- **UpdatedMCPToolOutput 命名：** Go 字段名使用 Go 风格的大写缩略词写法（`UpdatedMCPToolOutput`），同时通过 wire tag 保持 Python camelCase（`json:"updatedMCPToolOutput,omitempty"`）；今后新增带缩略词的字段（MCP、HTTP、SSE）也应遵循这一模式
- **NotificationHookInput 字段：** 包含 `NotificationType string`、`Message string`、`Title *string`（omitempty）；`Title` 为 nil 对应 Python `NotRequired[str]` 的缺失状态，因此解引用前必须判空；Notification hook 返回空的 `HookJSONOutput{}`（仅观察，不产生 HookSpecificOutput）；`PermissionRequestHookInput.PermissionSuggestions` 的类型是 `[]any`（对应 Python `list[Any]`），同样保留 nil 表示字段缺失的语义；`PermissionRequestHookInput` 故意不包含 `agent_id` / `agent_type`，因为 Python PR #545 并未对其应用 mixin；这些字段会在 Phase 2 item #13 再引入
- **MCP 工具注解（Phase 1 #5，Python PR #551）：** 存在两套不同的注解类型：`shared.ToolAnnotations`（作者侧，遵循 MCP 规范，Hint 后缀为 `ReadOnlyHint` / `DestructiveHint` / `IdempotentHint` / `OpenWorldHint`，另有 `Title`，均为 `*T` 且带 omitempty）与 `control.McpToolAnnotations`（CLI 状态响应侧，Hint 后缀被去掉，字段为 `ReadOnly` / `Destructive` / `OpenWorld`）；这与 Python 中 `mcp.types.ToolAnnotations` 和 `McpToolAnnotations` TypedDict 的拆分保持一致；`ToolOption` 函数式选项模式把 `NewTool()` 扩展为可接收可变参数 `...ToolOption`（保持向后兼容）；调用方式为 `NewTool(..., WithToolAnnotations(&ToolAnnotations{...}))`；`annotationsToMap` 通过 `json.Marshal + Unmarshal` 往返处理以遵守 omitempty；`buildToolsListResult` 被提取为独立辅助函数（而非内联进 `routeMcpMethod` 的 switch），用于控制 gocyclo；`tools/list` 的 wire 行为为：指针为 nil 时完全省略 `"annotations"` 键；传入 `&ToolAnnotations{}`（非 nil 但字段全为空）时则输出 `"annotations": {}`；wire 层字段使用 camelCase：`title`、`readOnlyHint`、`destructiveHint`、`idempotentHint`、`openWorldHint`

<!-- END AUTO-MANAGED -->

<!-- AUTO-MANAGED: git-insights -->

## Git 观察

- 约定式提交前缀：`feat:`、`fix:`、`docs:`、`test:`、`refactor:`、`chore:`
- 提交中的 Issue 引用格式：`(Issue #N)` 或 `(#N)`；PR 正文中使用 `Closes #N`
- 采用基于 PR 的工作流，并配套 CI 检查
- 最近关注点：Phase 1 item #5 - MCP 工具注解（Python PR #551）：新增作者侧 `shared.ToolAnnotations` 结构体，以及 `NewTool()` 上的 `ToolOption` / `WithToolAnnotations`；`annotationsToMap` 使用 JSON 往返处理；`buildToolsListResult` 从 `routeMcpMethod` 中提取（gocyclo 重构，移除 nolint）；`tools/list` 在 nil 时省略键，在 `&ToolAnnotations{}` 时输出空 map；示例 14 展示了如何在 `circle_area` 工具上使用 `WithToolAnnotations`；Phase 1 item #4 - 3 个新 hook 事件与缺失字段（Python PR #545）：新增 Notification、SubagentStart、PermissionRequest 事件类型；hook 数量从 7 增至 10；PostToolUseFailure hook 事件（Python PR #535，Phase 1 item #2）；AssistantMessage error 对象形式修复（Python PR #506，Go PR #124 squash）；GetMcpStatus() 控制协议方法（Python PR #516，Go PR #124）
- 基准测试组织方式：在 options、parser、shared、control、cli 等核心模块中统一使用表驱动基准测试
- Makefile 集成：所有代码质量检查（fmt、vet、lint、cyclo）统一收敛到 `make check`
- Python SDK 对齐跟踪：`docs/tracking/README.md` 负责跟踪所有待移植的 Python SDK PR（快照截至 2026-04-12）；按时间分为 4 个阶段（Phase 1：1 月 26 日至 2 月 20 日，Phase 2：3 月 3 日至 3 月 16 日，Phase 3：3 月 20 日至 3 月 30 日，Phase 4：3 月 31 日至 4 月 8 日）；Phase 1 已完成项：#1 GetMcpStatus（Go PR #124，Python PR #516）、#2 PostToolUseFailure hook（Go PR #125，Python PR #535）、#3 AssistantMessage error 字段修复（Go PR #124 squash，Python PR #506）、#4 3 个新 hook 事件与缺失字段（Go PR #128，Python PR #545）、#5 MCP 工具注解（Python PR #551，Go PR #133）；下一项待处理：Phase 1 item #6；Phase 3 item #35 已完成（Go PR #114，Python PR #749）；快照之后合并的 PR（2026-04-12 之后）记录在 `docs/tracking/post-snapshot.md` 中，并使用 "P" 行前缀（P1、P2、...）以及字母分支（a / b / c / ...）表示延后范围

<!-- END AUTO-MANAGED -->

<!-- AUTO-MANAGED: best-practices -->

## 最佳实践

- **TDD 方法：** 先编写失败测试，再实现使其通过
- **测试文件组织：** 先写测试函数，再写 mocks，最后写辅助函数
- **辅助函数：** 测试工具函数中始终调用 `t.Helper()`
- **线程安全：** 所有 mock 都必须正确使用互斥锁，保证线程安全
- **测试自包含：** 每个测试文件维护自己的辅助函数，避免跨文件依赖
- **基准测试组织：** 使用贴近真实场景的表驱动基准测试，并通过 `b.ReportAllocs()` 统计分配情况
- **`t.Fatal() + return`：** 在子测试中，`t.Fatal()` 后始终紧跟 `return`，以避免 staticcheck 的 SA5011 空指针误报（staticcheck 不会推断出 `t.Fatal()` 会中止执行）

<!-- END AUTO-MANAGED -->

<!-- MANUAL -->

## 自定义说明

在此补充项目特有说明。本节不会被自动修改。

<!-- END MANUAL -->
