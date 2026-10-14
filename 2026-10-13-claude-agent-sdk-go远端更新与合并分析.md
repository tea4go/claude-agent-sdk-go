# claude-agent-sdk-go 远端更新与合并计划

> **范围**：tea4go/main `e1b1846` ← severity1/main `d167fbe`（上游 v0.10.0）  
> **状态**：分析和补丁检查已完成；下文是待执行的代码合并计划，尚未修改 SDK 源码。  
> **日期**：2026-10-13

## 目录

1. [远端更新了哪些功能](#summary)
2. [冲突与删除项：明确采用哪一边](#decisions)
3. [具体合并步骤与验收要求](#steps)
4. [远端更新的代码文件](#files)
5. [按类别逐个分析 41 个 commit](#commits)

<a id="summary"></a>

## 一、远端更新了哪些功能

远端这 41 个提交主要新增了**后台任务控制、MCP 服务管理、会话改名和标签**，补齐了**单次提问的工具调用能力**，并集中修复了**进程退出、消息丢失和关闭卡住**的问题。具体变化如下：

| 功能             | 远端改了什么                                                           |
| -------------- | ---------------------------------------------------------------- |
| 单次提问           | 单次提问也走完整初始化流程，补齐工具权限回调和相关配置；回答结束后自动释放资源。                         |
| 后台任务           | 可以收到任务开始、进度和完成通知，也可以主动停止任务。                                      |
| MCP 服务管理       | 会话中可以重新连接 MCP 服务，或启用、停用某个服务。                                     |
| 会话管理           | 新增会话改名、标签；修复长项目路径下找不到会话、历史记录单行过大读不出来，以及恢复会话时的参数问题。               |
| 运行稳定性          | Claude 进程异常退出时及时报错；先交付已经收到的回答，再报退出错误；多轮对话只读取当前轮；修复关闭卡住和输出被截断的问题。 |
| 工具和权限回调        | 工具、权限和 hook 回调可以并发处理，减少互相阻塞；允许工具执行时，正确回传工具输入。                    |
| 模型与子代理配置       | 补充思考模式配置和子代理配置字段；明确子代理继承工具、禁用全部工具的区别。                            |
| 返回消息           | 补充用量、限流详情、回答停止原因和子代理信息；兼容新的系统事件，遇到未知内容块时继续读取回答。                  |
| Windows、CI 和文档 | Windows 要求使用原生 CLI，拒绝批处理启动路径；收紧自动工作流触发权限，覆盖率上报失败不再打断测试，并更新版本和说明。 |

**合并结论：**会话管理、权限输入修复、消息字段补充可以先移植；任务控制、MCP 管理、单次提问和退出处理需要一起解决通信与生命周期冲突。本地已有用量统计、中断和输出缓冲，保留现有实现、只补差异。能直接应用的完整提交只有 `077e959c`，`84efca4f` 只取 CI 改动；其余按后面的步骤整合。

<a id="decisions"></a>

## 二、冲突与删除项：明确采用哪一边

### 2.1 冲突解决方案

下表是**本计划拟采用的目标行为**；与远端不同的地方会在移植记录中保留说明。

| 冲突点                   | 本地 / 远端差异                                       | 本计划采用的方案                                                   |
| --------------------- | ----------------------------------------------- | ---------------------------------------------------------- |
| Usage 类型              | 本地两个消息类型均为 `*Usage`；远端为原始 map                   | 保留本地字段和辅助方法；需要原始信息时加扩展字段，不改变现有访问方式                         |
| 未知顶层消息                | 本地返回 RawMessage；远端丢弃                            | 保留 RawMessage，同时采用未知内容块跳过、新事件类型解析                          |
| Next(ctx)             | 本地响应单次读取 ctx；远端 Query 忽略它                       | 保留本地取消语义，同时响应 Query 的整体 ctx                                |
| Connect ctx 与 Abort   | 本地取消会结束进程树；远端 ctx 仅用于连接                         | 保留本地取消和 Abort；新增即时退出监测                                     |
| Windows/Unix 子进程      | 本地有进程组、Job Object/taskkill、隐藏窗口                 | 保留平台代码和 x/sys 依赖，吸收远端管道/回收修复                               |
| 消息与错误顺序               | 本地消息经过 Skill 合流，错误直连 transport                  | 在合流前排序；让消息、终止错误、EOF 经过同一有序路径，禁止错误绕过合流                      |
| CLI 退出后 Skill 通道      | 本地 injectChan 仍开着，上层流不结束                        | 会话结束时拒绝新注入并取消运行中 handler；保留已接收缓冲结果；不无限等待不响应取消的回调           |
| GetServerInfo         | 本地只有 connected/transport_type；远端返回真实 initialize | 返回真实响应副本并兼容保留旧键；SlashCommands 继续独立保留                       |
| Thinking 默认值          | 本地默认 8000 但不发 flag；远端取消默认且恢复 flag               | 默认不发预算；仅显式配置时生效，旧选项保留并标记弃用                                 |
| AgentDefinition.Tools | 旧实现丢弃空列表；远端空列表代表无工具                             | 采用远端 nil=继承、空列表=禁用全部的语义，并增加测试                              |
| 公开接口扩展                | 远端新增 EndInput/StopTask/MCP/Done/Err             | 接入新方法，保留 Abort/GetSlashCommands；同步更新自定义实现和 mock，明确记录源码兼容变化 |
| 模块与发行版本               | tea4go 模块、本地版本与远端不同                             | 导入统一为 tea4go；上游 v0.10.0 只是对齐目标，不直接覆盖本地发行号                  |

### 2.2 要删除或替换的旧实现

远端本次**没有整文件删除**。下面主要是内部路径替换，不代表删除公开的 Query 功能。

| 旧实现/行为                                                                 | 升级后的替代方式                                     | 执行时机                         |
| ---------------------------------------------------------------------- | -------------------------------------------- | ---------------------------- |
| `BuildCommandWithPrompt()`、`NewWithPrompt()`、promptArg、closeStdin 模式分支 | 单一 stream-json 构造入口，prompt 经 stdin user 消息发送 | 第 4 步，所有调用点迁移并测试后删除          |
| `addAgentFlags()` / `--agents` 序列化                                     | agents 放入 initialize 请求                      | 第 4 步                        |
| `needsProtocolHandshake()` 条件握手分支                                      | 连接统一 initialize                              | 第 4 步；保留实际所需的 MCP/Skill 判断逻辑 |
| 隐式默认 MaxThinkingTokens=8000                                            | 默认不指定，由 CLI 决定                               | 第 6 步；**不删除**旧选项和兼容常量        |
| Windows .bat/.cmd CLI 路径选择                                             | 原生 claude.exe 查找和明确拒绝错误                      | 第 2 步，说明安装方式变化               |
| 重复的 MCP 异步分发/旧 stdin writer                                            | 统一分发器、单一 stdin writer                        | 第 3 步，替代实现验证完成后清理            |

### 2.3 不删除、不恢复的内容

| 内容                                                                          | 明确处理                             |
| --------------------------------------------------------------------------- | -------------------------------- |
| 本地 Abort、RawMessage、Usage/ID/StopReason、SessionID、Skill 注册与注入、SlashCommands | **全部保留**；远端没有这些实现，不等于应从本地删除      |
| 本地 Windows 无窗口启动、Job Object/taskkill、Unix 进程组                               | **保留并整合**                        |
| 本地已删除 `docs/reference.md`                                                   | **不自动恢复**；有效说明整合进现有中文文档          |
| 本地已删除 `integration_test.go`                                                 | **不自动恢复旧整文件**；把需要的新回归用例迁入针对性测试文件 |
| 本地中文版 README、示例、开发说明                                                        | **保留中文结构**；只更新与最终代码相关的事实         |

<a id="steps"></a>

## 三、具体合并步骤与验收要求

### 第 0 步：建立整合基线

**代码步骤尚未执行。** 已完成固定版本比较、38 文件合并冲突检查、41 commit 补丁检查和 9 个行为探针。

- [ ] 从当前 tea4go/main 建立独立整合分支，保留本地扩展。
- [ ] 整理自定义 Client/Transport 实现、mock 和跨轮迭代调用点，列出需要同步迁移的位置。
- [ ] 将现有 9 个探针转为正式回归用例，保留本地 Next(ctx) 与 RawMessage 的契约。
- [ ] 记录现有测试基线，区分原有失败和迁移引入的失败。

**验收**：固定 HEAD，原有测试结果有记录，本地必须保留的行为都有对应检查项。

### 第 1 步：先接入 CI 更新

**提交**：`077e959c`；`84efca4f` 仅 CI 文件。

**文件**：`.github/workflows/claude.yml`、`.github/workflows/ci.yml`。

- [ ] 合并触发者限制和固定 action 引用。
- [ ] 关闭测试矩阵 fail-fast；覆盖率上报失败不阻断测试。
- [ ] 不把英文自动记忆整段覆盖到本地说明。

**验收**：workflow 配置正确，Go/OS 测试矩阵和本地权限配置保留。

### 第 2 步：移植独立修复和字段补充

**提交**：B 类 7 个 commit，详见第五章。

**文件**：`internal/session/session.go`、`session.go`、`internal/control/permissions.go`、`internal/cli/discovery.go`、`internal/parser/json.go`、`internal/shared/message.go`、hook 类型文件等。

- [ ] 修复会话大行，接入改名/标签/长目录定位。
- [ ] 回填权限 updatedInput。
- [ ] 接入 resume 参数和 Windows CLI 路径检查，保留本地平台进程管理。
- [ ] 补 Result.StopReason、hook agent 字段、RateLimitInfo.Raw。

**验收**：2MiB 会话行可读取；改名/标签可读回；原始/改写/显式空工具输入正确；新字段兼容缺失值；参数边界检查通过。

### 第 3 步：统一进程生命周期和控制请求

**提交**：`1840dc96`、`4615a71d`、`862ed8e1`、`60491c7d`、`c14ed11b`、`9241f178`，以及 `c3bc12b1`/`564e2e55`/`fef3fa3a` 的相关部分。

**文件**：`internal/subprocess/{transport,process,io,stdin,protocol_adapter}.go`、`internal/control/protocol.go`、`client.go`、`errors.go`。

- [ ] 启动成功后立即运行唯一 waiter；采用独立输出管道，避免 Wait 提前关闭 stdout。
- [ ] 接入 Done/Err、死进程检查和 pending request 快速失败。
- [ ] 统一 MCP/hook/permission 异步分发与取消。
- [ ] 保留进程树、Abort、Connect ctx 取消；将阻塞写移出 transport 状态锁。
- [ ] 普通 Disconnect 先有界排空，再执行树级关闭；取消/Abort 保持强制终止语义。

**验收**：自然退出及时可见；慢回调不阻塞控制响应；管道写满时仍可关闭；后代持有 stdout 时不无限等待；进程树无遗留。

### 第 4 步：迁移 Query 和初始化握手

**提交**：`7773c9e4`、`171132e9`、`a5535fbc`、`21091257`，以及 `055c97d1` 的协议部分。

**文件**：`query.go`、`client.go`、`types.go`、`internal/cli/discovery.go`、`internal/control/{protocol,types}.go`、`internal/subprocess/{transport,config}.go`、`internal/shared/stream.go`。

- [ ] Query 和 Client 共用配置校验及 initialize；agents/skills 通过握手传入。
- [ ] prompt 经 stdin 发送；接入 EndInput，并在双向控制完成前保留 stdin。
- [ ] Query 正常结束/失败/取消都自动清理，Close 保持幂等。
- [ ] 接回本地 Skill 注册表、直接执行入口、SessionID 和 SlashCommands。
- [ ] 返回真实 GetServerInfo，同时兼容旧字段。
- [ ] 验证完成后删除旧 --print/--agents 内部路径。

**验收**：普通 Query、带权限/hooks/SDK MCP 的 Query、长 agent 配置正常；Skill 与 SessionID 不回退；未取消 Query 时，单次 Next(ctx) 仍可独立取消。

### 第 5 步：解决消息顺序与多轮读取

**提交**：`392d6459`、`fc434a4c`，以及相关错误状态后续修复。

**文件**：`stream_reader.go`、`client.go` 的合流器/迭代器、`query.go`。

- [ ] 在 transport 源头完成消息/错误排序，再与 Skill 结果合流。
- [ ] 统一消息、错误、EOF 的传递顺序，消除错误绕过合流的路径。
- [ ] 定义会话结束后的 handler 取消与合流关闭，保留已接收的缓冲消息。
- [ ] ReceiveResponse 返回本轮 ResultMessage 后结束；下一轮新建迭代器。
- [ ] 错误跨轮保留，不重复报告，也不被第一轮结束吞掉。

**验收**：缓冲结果先于退出错误；连续两轮不串读；CLI EOF 后上层流可结束；同时有 Skill 注入时也满足这些条件。

### 第 6 步：接入新 API 与配置，补齐兼容行为

**提交**：`de83b7d9`、`91b34f56`、`028dd04c`、`6c60eb27`、`8f679ee2`，以及 `055c97d1`/`564e2e55`/`fef3fa3a` 的剩余功能。

**文件**：`client.go`、`options.go`、`types.go`、`internal/shared/{agent,task,message,options,version}.go`、解析器与控制协议文件。

- [ ] 新增任务生命周期视图、StopTask、MCP 重连/启停。
- [ ] 接入扩展 AgentDefinition，正确区分 nil/空列表。
- [ ] 接入 Thinking，取消隐式 8000 默认，保留旧 API。
- [ ] 接入 reset/服务端工具块/完整限流信息，保留 RawMessage。
- [ ] 补环境变量和恢复位置选项；SDKVersion 使用本地版本来源。
- [ ] 同步自定义接口实现和 mock；修正状态示例的 Usage 重复累加。

**验收**：新增 API 的控制请求/响应、失败路径和配置边界通过；现有 Usage 字段访问和本地扩展仍可用。普通使用方调用方式保留，自定义 Client/Transport 实现的新增方法列入迁移说明。

### 第 7 步：回归、清理与文档收尾

- [ ] 统一新增代码/测试的 import 为 `github.com/tea4go/claude-agent-sdk-go`，保留 Windows 的 x/sys。
- [ ] 按第二章清理被替换的旧内部路径；不恢复已删除旧文件。
- [ ] 执行 Go 单元测试、race、vet、格式和已有 lint/复杂度检查；工具缺失单独记录，不自动安装。
- [ ] 单独检查有独立 go.mod 的示例；根目录 `go test ./...` 不等于覆盖全部示例模块。
- [ ] 在 macOS/Linux/Windows 验证进程退出、取消、阻塞写、无窗口启动和进程树清理。
- [ ] 在真实 CLI 环境验证多轮会话、权限回调、MCP、agent、resume 和 Skill 注册/注入。
- [ ] 按最终实现更新中文说明和 tracking；记录源码兼容变化、Windows CLI 路径变化及本地版本号。

**完成条件**：独立修复、新 API、本地保留能力全部验收；剩余偏差逐项列明。此计划本身不执行发布、提交或推送。

<a id="files"></a>

## 四、远端更新的代码文件

补丁检查记录：[逐提交检查](./claude-agent-sdk-go更新分析附件/commit-apply-check.json)、[完整分支冲突记录](./claude-agent-sdk-go更新分析附件/merge-tree.txt)。逐提交检查是将每个补丁单独对当前本地 HEAD 检查，不是依次应用全部提交；完整分支合并检查发现 38 个冲突文件。

远端相对共同祖先共修改 **86 个文件**：31 个非测试 Go 文件、37 个测试文件、15 个 Markdown 文档、3 个配置文件。新增 20 个文件，其中 5 个是运行时代码、15 个是测试；没有整文件删除。

| 模块        | 更新代码                                                                                                        |
| --------- | ----------------------------------------------------------------------------------------------------------- |
| 根包 API    | `client.go`、`query.go`、**新增** `stream_reader.go`、`options.go`、`types.go`、`errors.go`、`session.go`、`doc.go`  |
| CLI 参数    | `internal/cli/discovery.go`                                                                                 |
| 控制协议      | `internal/control/protocol.go`、`types.go`、`permissions.go`、`hooks.go`、`types_hook.go`、`mcp.go`              |
| 消息解析/会话文件 | `internal/parser/json.go`、`internal/session/session.go`                                                     |
| 共享类型      | `internal/shared/message.go`、`options.go`、`stream.go`；**新增** `agent.go`、`task.go`、`version.go`              |
| 子进程       | `internal/subprocess/transport.go`、`config.go`、`io.go`、`process.go`、`protocol_adapter.go`；**新增** `stdin.go` |
| 示例        | `examples/15_programmatic_subagents/main.go`、`examples/20_debugging_and_diagnostics/main.go`                |

[完整文件清单：逐文件变更、行数、测试、配置及每个 commit 的代码文件](./claude-agent-sdk-go更新分析附件/远端代码变更清单.md)

已有验证依据：同一组 9 个探针在双方固定源码副本执行。远端改善了本地的结果保序、EOF 清理、按轮结束、权限参数、通道结束、会话大行、未知内容块 7 项；本地在 Next(ctx) 独立取消和 RawMessage 保留 2 项上满足现有契约。本次尚未执行完整集成后的全量、race、真实 CLI 或 Windows 运行验证。

- [根包行为探针](./claude-agent-sdk-go更新分析附件/review_probe_test.go)
- [会话大行探针](./claude-agent-sdk-go更新分析附件/session_probe_test.go)
- [解析器探针](./claude-agent-sdk-go更新分析附件/parser_probe_test.go)

<a id="commits"></a>

## 五、按类别逐个分析 41 个 commit

**分类口径**：A 类可直接处理 CI（2 个）；B 类独立移植（7 个）；C 类联动/冲突整合（21 个）；D 类本地已有（3 个）；E 类文档/版本（8 个）。每个 commit 只归入一个主要类别；混合提交在“处理办法”中拆分。

**补丁检查口径**：每个 commit 单独对当前本地 HEAD 检查。除 A1 外，其余整提交均未通过直接应用检查；A2 的 CI 文件单独通过。是否能直接应用补丁与是否功能重复、是否有前置依赖是不同判断。

## A 类：可直接处理的 CI 提交（2 个）

### A1. 077e959c — 限制 @claude 触发者（#172）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/077e959c6303dd3963452187b1f2ce3cafb4d685) · 共变更 1 个文件

- **更新代码/文件**：`.github/workflows/claude.yml`。
- **与本地关系**：CI：限制 @claude 触发者并固定 action commit。
- **具体处理**：直接应用整个补丁。仅允许 OWNER/MEMBER/COLLABORATOR 触发，并固定 action 的 commit；保留本地现有权限配置。
- **计划与验收**：第 1 步；检查 workflow 条件和 action 引用。

### A2. 84efca4f — Codecov 失败不终止测试矩阵（#190）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/84efca4fc91187e6808ce522247e3dadf45b6917) · 共变更 2 个文件

- **更新代码/文件**：`.github/workflows/ci.yml`、`CLAUDE.md`。
- **与本地关系**：CI：Codecov 失败不阻断测试，测试矩阵关闭 fail-fast。
- **具体处理**：只应用 .github/workflows/ci.yml；CLAUDE.md 的英文自动记忆不整段覆盖，最后补一条中文说明。
- **计划与验收**：第 1 步；确认 fail-fast:false 和覆盖率步骤 continue-on-error:true。

## B 类：本地缺失、可独立移植的功能（7 个）

### B1. 8271a7f5 — 会话 JSONL 支持超过 1MB 的单行（#139）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/8271a7f5a44f60fe82aff99c1ff7cb7113a81f3c) · 共变更 2 个文件

- **更新代码/文件**：`internal/session/session.go`。
- **与本地关系**：确实缺失：会话文件 JSONL 的 1MB Scanner 上限，本次 2MiB 行测试已复现。
- **具体处理**：移植 Reader.ReadBytes 逐行解析并让两个读取入口共用；保留本地目录和工作树处理。此处是历史会话文件，与 stdout 缓冲修复不同。
- **计划与验收**：第 2 步；2MiB 单行和后续消息都能读出，坏 JSON 行仍跳过。

### B2. 4b646936 — 权限允许响应补齐 updatedInput（#158）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/4b64693675b6801fa81b45af1b8b08df02066a1c) · 共变更 2 个文件

- **更新代码/文件**：`internal/control/permissions.go`。
- **与本地关系**：确实缺失：allow 未设置 UpdatedInput 时回填原工具输入。
- **具体处理**：保留现有 PermissionResultAllow API；回调未设置 UpdatedInput 时，用原工具 input 回填。不要把用户有意返回的空 map 当作未设置。
- **计划与验收**：第 2 步；原始输入、改写输入和显式空输入三种响应各有测试。

### B3. c5df0c08 — 恢复会话参数避免被当成 CLI flag（#160）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/c5df0c08a2d2ffbb3e54b2f1c1e8bc1715e79f9d) · 共变更 7 个文件

- **更新代码/文件**：`client.go`、`internal/cli/discovery.go`。
- **与本地关系**：确实缺失：--resume=value 单 argv token，避免以短横线开头的值成为 flag。
- **具体处理**：移植 --resume=value 的单参数形式及该提交的 Windows 参数校验；保留 WithSessionID 及其校验。
- **计划与验收**：第 2 步；普通 ID、短横线开头值、Windows 特殊字符都按预期处理。

### B4. 58cf570f — Windows 拒绝批处理 CLI 路径（#161）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/58cf570f29d04e8f1bc8ae40fd6dc749f3c2356e) · 共变更 7 个文件

- **更新代码/文件**：`internal/cli/discovery.go`、`internal/subprocess/transport.go`。
- **与本地关系**：新增兼容性约束：Windows 拒绝 .bat/.cmd CLI 路径，要求原生可执行文件。
- **具体处理**：采用远端路径检查与原生 claude.exe 查找；保留本地 NewExecCommandContext 的无窗口启动和进程树。记录 .bat/.cmd 不再接受的兼容性变化。
- **计划与验收**：第 2 步；用 Windows 原生 CLI 验证，拒绝批处理路径且不弹控制台窗口。

### B5. 6c9d76fe — 会话改名、标签和长目录定位（#171）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/6c9d76fe39022c9844d7e6e07be12b5e1c2781ee) · 共变更 7 个文件

- **更新代码/文件**：`internal/session/session.go`、`session.go`。
- **与本地关系**：新增：RenameSession、TagSession，以及长项目目录定位修复。
- **具体处理**：移植 RenameSession/TagSession、长目录定位和对应测试；追加写入现有 JSONL，保留本地会话读取返回类型。
- **计划与验收**：第 2 步；标题/标签写入后读回，nil 清除标签，长项目路径可定位。

### B6. 6ace5227 — 保留原始限流信息（#187）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/6ace52276a1a06e17f873e5afd7e8b98a6c694d2) · 共变更 4 个文件

- **更新代码/文件**：`internal/parser/json.go`、`internal/shared/message.go`。
- **与本地关系**：新增：RateLimitInfo.Raw，保留未建模字段。
- **具体处理**：增加 RateLimitInfo.Raw 并在解析时保存完整 rate_limit_info；保留已有强类型字段。
- **计划与验收**：第 2 步；未知限流字段可从 Raw 读取。

### B7. f24197dc — 结果停止原因与工具 hook 的 agent 字段（#191）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/f24197dc00d48330902ab0507536ade20553d4c9) · 共变更 12 个文件

- **更新代码/文件**：`internal/control/hooks.go`、`internal/control/types_hook.go`、`internal/parser/json.go`、`internal/shared/message.go`。
- **与本地关系**：新增：ResultMessage.StopReason 以及四种工具生命周期 hook 的 AgentID/AgentType；勿与本地 AssistantMessage.StopReason 混淆。
- **具体处理**：给 ResultMessage 加 StopReason；给 PreToolUse/PostToolUse/PostToolUseFailure/PermissionRequest 输入补 AgentID/AgentType。保留本地 AssistantMessage.StopReason。
- **计划与验收**：第 2 步；字段缺失/null 不误填，四种 hook 都能解析子代理上下文。

## C 类：需要联动整合或解决行为冲突的提交（21 个）

### C1. 7773c9e4 — Query 与 Client 统一流式初始化（#134）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/7773c9e44aa149ca2140c6c5de77e33fd0eb6227) · 共变更 37 个文件

- **更新代码/文件**：`client.go`、`examples/15_programmatic_subagents/main.go`、`internal/cli/discovery.go`、`internal/control/hooks.go`等 13 个非测试 Go文件。
- **与本地关系**：架构迁移：Query/Client 统一流式输入及 initialize，agents 从 argv 移至握手，新增 Transport.EndInput。
- **具体处理**：这是整合主干。改用 stdin user JSON；agents 经 initialize 发送；新增 EndInput；接回本地 Skill 注册、直接执行、SessionID 和 SlashCommands。调用点迁移完成后再删旧 --print 路径。
- **计划与验收**：第 4 步，依赖第 3 步；普通 Query、带 hook/权限/MCP 的 Query、长 agent prompt 都通过。

### C2. 1840dc96 — 启动后立即回收进程并修复输出管道竞态（#150）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/1840dc96071879459c04bf053ab66fc72690bb9c) · 共变更 9 个文件

- **更新代码/文件**：`internal/control/protocol.go`、`internal/subprocess/io.go`、`internal/subprocess/process.go`、`internal/subprocess/transport.go`。
- **与本地关系**：部分重叠：本地有 sync.Once Wait owner；远端在 Start 后立即 Wait，并使用 os.Pipe，需结合进程树实现整合。
- **具体处理**：保留本地 sync.Once/进程树；把 waiter 移到启动成功后立即运行。采用独立 os.Pipe 思路，先构造协议再启动 stdout reader，避免 Wait 提前关读管道。
- **计划与验收**：第 3 步；CLI 自然退出及时被发现，退出前最后一行输出不丢失。

### C3. 4615a71d — 所有控制请求并发处理（#151）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/4615a71debef7641db3b123a948a97df5911ec7e) · 共变更 13 个文件

- **更新代码/文件**：`internal/control/hooks.go`、`internal/control/mcp.go`、`internal/control/permissions.go`、`internal/control/protocol.go`等 6 个非测试 Go文件。
- **与本地关系**：部分重叠：本地仅 SDK MCP 异步；远端扩展到权限/hooks，并管理取消和错误响应。
- **具体处理**：在本地 MCP 异步分发基础上扩展 hook 和权限请求；统一管理 handler context、取消、panic/error 响应，避免叠加两套异步分发。
- **计划与验收**：第 3 步；慢权限/hook 不阻塞后续控制响应，关闭后不写回过期结果。

### C4. 171132e9 — Query 正确启用权限回调（#152）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/171132e9ae6d5f7694298f78cd461790c5aa74b9) · 共变更 5 个文件

- **更新代码/文件**：`client.go`、`options.go`、`query.go`。
- **与本地关系**：确实缺失：Query 不会自动配置 permission-prompt-tool=stdio；测试已复现。
- **具体处理**：抽出供 Query 和 Client 共用的 prepareOptions；设置 stdio 并检查互斥配置。真实 Query 支持依赖统一流式，不能只移植参数赋值。
- **计划与验收**：第 4 步，与 7773c9e4 配套；mock 参数测试和真实权限往返都通过。

### C5. a5535fbc — Query 结束后自动关闭（#153）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/a5535fbc0570a5d0bf931aaafcb54645a8b6193d) · 共变更 3 个文件

- **更新代码/文件**：`query.go`。
- **与本地关系**：确实缺失：Query 读至 EOF 不自动 Close；测试已复现。
- **具体处理**：统一 EOF、读错误、初始化后发送失败和 Query ctx 取消的收尾；显式 Close 保持幂等。保留本地 Next(ctx) 的独立取消能力。
- **计划与验收**：第 4 步；每个终止路径只 Close 一次，不留临时文件和进程。

### C6. 862ed8e1 — CLI 异常退出向调用方报错（#156）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/862ed8e1743eabd5ca93f35861e955ce8c92114c) · 共变更 10 个文件

- **更新代码/文件**：`client.go`、`internal/subprocess/io.go`、`internal/subprocess/transport.go`、`query.go`。
- **与本地关系**：确实缺失：把 CLI 非零退出上报为 ProcessError；本地主要检测无 stdout 等情况。
- **具体处理**：接入即时 waiter 的退出状态；完整交付已接收消息后报告 ProcessError。保留本地无输出/stderr 诊断信息。
- **计划与验收**：第 3、5 步；非零退出可见，并且最终 ResultMessage 不被错误抢先吞掉。

### C7. fc434a4c — ReceiveResponse 只读取当前轮（#159）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/fc434a4c6ea392a54457fbf58799187031a3abd3) · 共变更 3 个文件

- **更新代码/文件**：`client.go`。
- **与本地关系**：确实缺失：ReceiveResponse 在 ResultMessage 后结束；测试已复现。
- **具体处理**：采用返回 ResultMessage 后结束迭代器的语义；下一轮重新调用 ReceiveResponse。检查本地调用方是否把一个迭代器跨多轮复用。
- **计划与验收**：第 5 步；连续两轮不串读，下一轮第一条消息仍可读取。

### C8. c3bc12b1 — 关闭流程、stdin 锁与 argv 加固（#163）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/c3bc12b14a18fc78dd6514302f39adf3436a0674) · 共变更 22 个文件

- **更新代码/文件**：`client.go`、`internal/cli/discovery.go`、`internal/subprocess/config.go`、`internal/subprocess/io.go`等 10 个非测试 Go文件。
- **与本地关系**：高风险语义分歧：Connect context 只约束连接；关闭流程改为 EOF/TERM/KILL；stdin 锁与本地部分重叠。
- **具体处理**：拆开移植：采用锁外 I/O、共享 stdinWriter、EndInput/Close 区分和 argv 加固；保留本地 Connect ctx 取消、Abort、进程树。普通 Disconnect 增加有界 EOF 排空，再执行本地树级终止。
- **计划与验收**：第 3、4 步；管道写满时仍可取消/关闭；明确不采用远端 Connect ctx 只管连接的语义。

### C9. 055c97d1 — 环境变量、恢复参数和协议选项对齐（#170）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/055c97d1bf6aefb35d9b4434be64318afb64e2a7) · 共变更 32 个文件

- **更新代码/文件**：`client.go`、`doc.go`、`internal/cli/discovery.go`、`internal/control/protocol.go`等 11 个非测试 Go文件。
- **与本地关系**：混合：SettingSources 默认值及 Skills initialize 本地已有；环境变量清理、SDK 版本、resume 截断参数等仍需吸收。
- **具体处理**：只补缺口：清理继承的 CLAUDECODE、发送本地 SDK 版本、两种恢复截断选项、argv 边界。SettingSources 默认值和 Client skills 握手已有，保留；补到 Query 路径。
- **计划与验收**：第 4、6 步；保留本地配置隔离、Skill 过滤和 SessionID，不复制远端版本号。

### C10. 60491c7d — Client.Done/Err 和死进程快速失败（#165）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/60491c7dafee0854f8ad5b78a4c54da68361f335) · 共变更 13 个文件

- **更新代码/文件**：`client.go`、`internal/subprocess/io.go`、`internal/subprocess/transport.go`、`types.go`。
- **与本地关系**：新增：Client.Done/Err、进程退出后 fail-fast，需要接入本地进程树和合流通道。
- **具体处理**：把退出通知接到本地即时 waiter；为客户端增加 Done/Err，并让控制操作先检查进程状态。不要依赖 stdout EOF 判定进程退出。
- **计划与验收**：第 3 步；子进程持有 stdout 时也能收到退出通知；更新自定义 Client/mock。

### C11. 21091257 — GetServerInfo 返回真实初始化结果（#166）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/21091257a271fecf543be6a209fdd2ffe056b8c6) · 共变更 15 个文件

- **更新代码/文件**：`client.go`、`examples/20_debugging_and_diagnostics/main.go`、`internal/control/protocol.go`、`internal/subprocess/config.go`等 5 个非测试 Go文件。
- **与本地关系**：行为变化：GetServerInfo 返回真实 initialize 响应，本地仍返回 connected/transport_type。
- **具体处理**：保存 CLI 完整 initialize 响应并返回副本；兼容保留 connected/transport_type 旧键。保留独立的 SlashCommands 缓存和发现方法。
- **计划与验收**：第 4 步；既能读真实模型/命令信息，旧诊断调用也能工作。

### C12. 392d6459 — 结果消息先于退出错误交付（#164）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/392d64597959f79a614a82ceb6018d9f36f4728f) · 共变更 8 个文件

- **更新代码/文件**：`client.go`、`query.go`、`stream_reader.go`。
- **与本地关系**：确实缺失：缓冲消息优先于退出错误；本次 100 次构造测试本地丢失结果 51 次，远端 0 次。
- **具体处理**：Query 可使用有序 reader；Client 必须在合流前处理 transport 消息与错误顺序，再接入本地 Skill 通道。禁止直接把远端 reader 包在 mergedChan 外而让错误绕过合流。
- **计划与验收**：第 5 步；将现有 100 次保序探针扩展到有本地 Skill 注入的场景。

### C13. c14ed11b — 关闭时立即结束等待中的控制请求（#167）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/c14ed11bdcef9228f770dbb7c375a9441b4307b7) · 共变更 10 个文件

- **更新代码/文件**：`internal/control/protocol.go`、`internal/control/types.go`、`internal/subprocess/io.go`。
- **与本地关系**：确实缺失：Close/CLI 退出立即结束 pending control requests，新增 ErrProtocolClosed。
- **具体处理**：引入 ErrProtocolClosed、FailPendingRequests 及关闭检查；退出错误同时通知正在等待的请求，不再逐个等满超时。
- **计划与验收**：第 3 步；多个 pending 请求在关闭/CLI 退出后及时失败，关闭后不再发送请求。

### C14. 9241f178 — 关闭不被后代持有 stdout 拖住（#169）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/9241f1789159dfc1f7a018bbda0c6e8f16021b5d) · 共变更 5 个文件

- **更新代码/文件**：`internal/subprocess/transport.go`。
- **与本地关系**：部分重叠：本地已有关闭读管道和限制读协程等待；远端采用 CLI 根进程退出后的管道收尾策略。
- **具体处理**：吸收有界排空和主动关闭读端的处理；保留本地后代进程清理能力，避免只关闭读管道却放任后台进程继续运行。
- **计划与验收**：第 3 步；后代持有 stdout 时 Disconnect/Abort 能结束且进程树清理正常。

### C15. de83b7d9 — 任务生命周期消息与 StopTask（#168）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/de83b7d97f6a7e8a6b27492ad2e5f71de11c762c) · 共变更 20 个文件

- **更新代码/文件**：`client.go`、`internal/control/protocol.go`、`internal/control/types.go`、`internal/parser/json.go`等 7 个非测试 Go文件。
- **与本地关系**：新增：任务生命周期类型化视图和 StopTask；实际接收值仍是 *SystemMessage。
- **具体处理**：新增 shared/task.go、AsTask* 解码和 StopTask 控制链；事件仍以 SystemMessage 到达，保留原 type-switch 分支。
- **计划与验收**：第 6 步，依赖第 3–5 步；正常/失败/停止状态可识别，并可停止指定 task_id。

### C16. 6c60eb27 — ThinkingConfig 和有效思考预算（#181）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/6c60eb27b1c2a4bfbdad3cd9f38080a9aff839ce) · 共变更 12 个文件

- **更新代码/文件**：`internal/cli/discovery.go`、`internal/shared/options.go`、`options.go`。
- **与本地关系**：新增：ThinkingConfig/WithThinking；MaxThinkingTokens 从不发 flag 变为生效，取消默认 8000。
- **具体处理**：引入 WithThinking/三种模式/显示选项；取消默认 8000；保留旧 WithMaxThinkingTokens 但标记弃用，只有显式设置才发送预算 flag。
- **计划与验收**：第 6 步；默认不发预算参数，显式预算生效，typed nil 不 panic。

### C17. 8f679ee2 — 未知内容块兼容及会话重置消息（#182）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/8f679ee20aca714739e23782bdf38f493b90f3b8) · 共变更 11 个文件

- **更新代码/文件**：`internal/parser/json.go`、`internal/shared/message.go`、`types.go`。
- **与本地关系**：取舍：未知顶层消息本地保留 RawMessage、远端丢弃；未知内容块远端可跳过，本地报错；新增 ConversationResetMessage。
- **具体处理**：采用未知内容块跳过和 ConversationResetMessage；保留本地未知顶层类型转 RawMessage 的行为，不采用远端静默丢弃。
- **计划与验收**：第 6 步；未知顶层事件保留，未知块不影响已知文本，reset 字段正常解析。

### C18. 564e2e55 — v0.9.0 前的混合修复（#185）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/564e2e55617afea11306cab83a916fc5e30ab348) · 共变更 23 个文件

- **更新代码/文件**：`client.go`、`errors.go`、`internal/cli/discovery.go`、`internal/parser/json.go`等 12 个非测试 Go文件。
- **与本地关系**：混合后续修复：Done/Err 锁、共享 pending error、错误导出、thinking typed nil、服务端工具块等。
- **具体处理**：按功能拆入各阶段：退出锁/错误导出放第 3 步，共享 pending error 放第 5 步，服务端工具块和 thinking typed nil 放第 6 步；不整包覆盖。
- **计划与验收**：第 3、5、6 步；重点复测 Next(ctx) 取消、并发迭代器状态和服务端工具块。

### C19. fef3fa3a — v0.9.1 混合修复及限流补全（#189）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/fef3fa3a1d1a98c688f412938ee75774725627ed) · 共变更 14 个文件

- **更新代码/文件**：`client.go`、`internal/parser/json.go`、`internal/shared/message.go`、`internal/shared/options.go`等 7 个非测试 Go文件。
- **与本地关系**：混合：v0.9.1 退出状态窗口修复、thinking typed nil、限流利用率与额外常量。
- **具体处理**：接入 Disconnect 中 Err 的一致性修复；增加 Utilization/OverageDisabledReason/状态常量；带入 thinking nil 修复；跳过远端版本号赋值。
- **计划与验收**：第 3、6 步；断连期间错误类型稳定，限流缺失值保留 nil。

### C20. 91b34f56 — MCP 重连与启停（#192）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/91b34f567f2f7709bbb090ff84e683d2c016f513) · 共变更 15 个文件

- **更新代码/文件**：`client.go`、`internal/control/protocol.go`、`internal/control/types.go`、`internal/subprocess/config.go`等 5 个非测试 Go文件。
- **与本地关系**：新增：ReconnectMcpServer/ToggleMcpServer，扩展公开 Client/Transport 接口。
- **具体处理**：增加 ReconnectMcpServer/ToggleMcpServer 完整控制链，采用远端超时和输入校验；false 必须在 JSON 中发出。
- **计划与验收**：第 6 步；重连、启用、禁用和空名称错误均覆盖；更新 Client/Transport/mock。

### C21. 028dd04c — 子代理配置字段与校验（#193）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/028dd04c21d8060f2673a0ee63e77f5af0c7306e) · 共变更 17 个文件

- **更新代码/文件**：`client.go`、`examples/15_programmatic_subagents/main.go`、`internal/control/types.go`、`internal/shared/agent.go`等 7 个非测试 Go文件。
- **与本地关系**：新增：AgentDefinition 的九类字段及校验；Tools 等 nil 与空切片的 wire 语义不同。
- **具体处理**：引入 agent.go 和新字段；在 initialize 中编码，保留 nil/空列表差异。保留本地 Skill 注册表入口；验证不支持的内联 SDK server 配置。
- **计划与验收**：第 6 步，依赖第 4 步；Tools:nil 继承，Tools:[] 禁用工具，负预算等非法输入失败。

## D 类：本地已有，保留本地实现或只补差异（3 个）

### D1. 4702b65f — AssistantMessage 的单次 API Usage（#148）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/4702b65f41ad683dbfaf1bb73291bdf4b37a790f) · 共变更 8 个文件

- **更新代码/文件**：`internal/parser/json.go`、`internal/shared/message.go`。
- **与本地关系**：已部分覆盖，存在类型分歧：本地 AssistantMessage.Usage 为 *Usage，还保留 ID/StopReason；远端为 *map[string]any。
- **具体处理**：不套用 map 类型替换。本地已有 *Usage、ID、StopReason；保留这些字段与方法，按需要增加原始字段扩展。状态示例改为按 ID 更新/去重，避免重复事件逐条累加。
- **计划与验收**：第 6 步补差异；现有 m.Usage.InputTokens 代码继续编译。

### D2. 9a6d28ed — Interrupt 从信号切换为控制请求（#154）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/9a6d28ed1a184c307f89de4f74bdaf0130ed7e46) · 共变更 6 个文件

- **更新代码/文件**：`client.go`、`internal/subprocess/transport.go`。
- **与本地关系**：本地已有：Interrupt 已通过 control protocol 中断当前轮次；仍可吸收远端 dead-process 检查。
- **具体处理**：跳过重复的核心改动，本地已实现 protocol.Interrupt；仅随退出监测补 live-process 检查，保留 Abort。
- **计划与验收**：第 3 步；Interrupt 后可继续下一轮，Abort 才终止进程树。

### D3. 76ca378e — stdout 缓冲按需增长（#188）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/76ca378e0d47a1532911a720bdfa5473da4ba741) · 共变更 3 个文件

- **更新代码/文件**：`internal/subprocess/io.go`。
- **与本地关系**：部分重叠：本地 stdout 已是 Reader+Builder 按需增长；远端改进 Scanner 初始缓冲区分配。
- **具体处理**：保留本地 Reader+Builder；不为了此提交切回 Scanner。对照移植大小边界的测试要求，保证最大值边界一致。
- **计划与验收**：第 3、7 步；等于上限通过，超限返回 SDK 错误，大上限不立即分配全部内存。

## E 类：文档和版本提交，最后统一整理（8 个）

### E1. 2bcffbc0 — 自动记忆文档更新

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/2bcffbc0dec755dd299a6ff02fb9845190ccecd8) · 共变更 1 个文件

- **更新代码/文件**：`CLAUDE.md`。
- **与本地关系**：开发说明更新，无独立运行时功能。
- **具体处理**：不直接套用英文自动记忆；最终实现落地后，只把仍适用的事实写入现有中文开发说明。
- **计划与验收**：第 7 步；说明与最终代码一致。

### E2. fde63ade — Python SDK 对齐跟踪扩展（#155）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/fde63adec803cf17b4bb0fdf7d050e1cbdb0c903) · 共变更 4 个文件

- **更新代码/文件**：`docs/parity.md`、`docs/reference.md`、`docs/tracking/README.md`、`docs/tracking/post-snapshot.md`。
- **与本地关系**：Python SDK 对齐跟踪及参考文档更新。
- **具体处理**：吸收待办范围和 PR 对应关系；保留中文文档，只有真正移植完成的项才标为完成。
- **计划与验收**：第 7 步；不恢复已删除参考文档，也不照抄远端完成状态。

### E3. fb22dfcb — 修正 100% parity 声明（#157）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/fb22dfcb568d7aa3bc135c1a364f581254ed36d0) · 共变更 3 个文件

- **更新代码/文件**：`.goreleaser.yml`、`CLAUDE.md`、`docs/parity.md`。
- **与本地关系**：文档措辞改为以 100% parity 为目标，而非声称已经全部实现。
- **具体处理**：采用“以完整对齐为目标”的准确表述；发布链接、模块路径和中文风格保留 tea4go。
- **计划与验收**：第 7 步；发布页不出现 severity1 的模块安装地址。

### E4. f697fc34 — v0.7.1 对齐跟踪文档（#162）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/f697fc34d93d128b3e7d95fee806b9f8848a370c) · 共变更 4 个文件

- **更新代码/文件**：`CLAUDE.md`、`docs/parity.md`、`docs/tracking/README.md`、`docs/tracking/post-snapshot.md`。
- **与本地关系**：v0.7.1 对齐跟踪文档更新。
- **具体处理**：作为移植记录参考，不单独覆盖当前文档；与其它 tracking 更新合并整理。
- **计划与验收**：第 7 步；按本地最终功能更新状态。

### E5. 220072db — v0.8.0 对齐与贡献 PR 映射（#173）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/220072db96cf5dd1096cc1e97f0230a98adb553e) · 共变更 3 个文件

- **更新代码/文件**：`docs/parity.md`、`docs/tracking/README.md`、`docs/tracking/post-snapshot.md`。
- **与本地关系**：v0.8.0 对齐跟踪及贡献 PR 映射文档。
- **具体处理**：保留可追溯的 PR 映射，避免覆盖本地已补功能的说明；按最终实现整理。
- **计划与验收**：第 7 步；记录上游来源与本地差异。

### E6. 774c6695 — 标记 #164–#169 已合并（#174）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/774c66951ce4b5d5b42d6db539ec15a9709ad884) · 共变更 4 个文件

- **更新代码/文件**：`CLAUDE.md`、`docs/parity.md`、`docs/tracking/README.md`、`docs/tracking/post-snapshot.md`。
- **与本地关系**：记录 #164–#169 已合并的对齐文档。
- **具体处理**：远端 merged 不代表本地已移植；等对应步骤验收后更新本地 tracking。
- **计划与验收**：第 7 步；状态准确反映本地。

### E7. cf58dbcd — 远端 v0.9.0 发布版本（#186）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/cf58dbcd347a781fe17cdb5238d1b8fc0550b414) · 共变更 2 个文件

- **更新代码/文件**：`internal/shared/version.go`。
- **与本地关系**：v0.9.0 发布版本更新。
- **具体处理**：跳过将 SDKVersion 直接设为 0.9.0；新增版本环境变量时使用 tea4go 自己的版本来源。
- **计划与验收**：第 7 步；区分“对齐上游版本”与“本地发行版本”。

### E8. d167fbef — 远端 v0.10.0 发布版本（#194）

[查看原始 commit](https://github.com/severity1/claude-agent-sdk-go/commit/d167fbef06759f127f5623bcf685c05b422de5df) · 共变更 2 个文件

- **更新代码/文件**：`internal/shared/version.go`。
- **与本地关系**：v0.10.0 发布版本更新。
- **具体处理**：作为本次比较目标，不直接把本地发行版本改成 0.10.0；本次文档工作不发布版本。
- **计划与验收**：第 7 步；发布说明注明对齐至 d167fbe，版本由本地发布流程管理。

