# 可视化 Claude Code 工具开发计划

## 项目概述

基于 claude-agent-sdk-go 开发一个可视化的 Claude Code 使用工具，提供会话管理、Skills/Agents 可视化、权限交互等功能。

---

## 功能需求分析

### 1. 会话功能与会话继续

**SDK 支持情况：**
- **Client API**：支持双向流式通信，适合实时交互场景
- **会话继续**：
  - `WithContinueConversation(true)` - 继续最近会话
  - `WithResume(sessionID)` - 恢复指定会话
  - `WithForkSession(true)` - 分叉会话
- **上下文会话ID**：`QueryWithSession(ctx, prompt, sessionID)` 支持同一连接内的多会话隔离
- **CLI Session UUID**：`ResultMessage.SessionID` 返回持久化会话标识

**实现要点：**
- 使用 Client API（非 Query API）以获得完整的控制协议支持
- 维护会话列表（SessionID、创建时间、消息历史）
- 支持从历史会话恢复

### 2. Skills/Agents/Claude原生指令可视化

**SDK 支持情况：**
- **Slash Commands 发现**：
  - `DiscoverSlashCommands(ctx, opts)` - 独立发现函数
  - `Client.GetSlashCommands(ctx)` - 流式会话中获取
- **Skills 系统**：
  - `WithSkillsAll()` / `WithSkillsList(names...)` - 启用 Skills
  - `WithSkillRegistry(root, names...)` - 外部 Skill 目录
  - `RegisterSkill(name, handler)` - 注册自定义 Skill
- **Agents 定义**：
  - `WithAgent(name, definition)` - 定义 Agent
  - `AgentDefinition` 包含 Description、Prompt、Tools、Model
- **MCP Tools 发现**：
  - `Client.GetMcpStatus(ctx)` - 获取 MCP 服务器状态和工具列表

**实现要点：**
- 启动时调用 `DiscoverSlashCommands` 获取原生指令列表
- 维护 Skills/Agents 注册表的可视化展示
- 支持一键调用 Skill/Agent

### 3. 权限切换与权限交互

**SDK 支持情况：**
- **权限模式**：
  - `PermissionModeDefault` - 标准模式（触发回调）
  - `PermissionModeAcceptEdits` - 自动批准编辑操作
  - `PermissionModePlan` - 计划模式（只读）
  - `PermissionModeBypassPermissions` - 绕过所有检查
- **权限回调**：
  - `WithCanUseTool(callback)` - 设置权限决策回调
  - `ToolPermissionContext` 包含工具名、输入、建议
  - `PermissionResultAllow/Deny` - 返回决策结果
- **动态权限更新**：
  - `PermissionUpdate` 支持添加/替换/删除规则
  - 可在回调中返回 `UpdatedPermissions`
- **运行时切换**：
  - `Client.SetPermissionMode(ctx, mode)` - 动态切换模式

**实现要点：**
- UI 提供权限模式切换按钮
- 权限请求时弹出交互对话框
- 支持批量批准/拒绝操作

---

## 技术架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (UI Layer)                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Session     │  │ Command     │  │ Permission          │  │
│  │ Manager UI  │  │ Palette UI  │  │ Dialog UI           │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
└─────────┼────────────────┼────────────────────┼─────────────┘
          │                │                    │
          ▼                ▼                    ▼
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Session     │  │ Discovery   │  │ Permission          │  │
│  │ Manager     │  │ Service     │  │ Manager             │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
│         │                │                    │              │
│         └────────────────┼────────────────────┘              │
│                          ▼                                   │
│              ┌─────────────────────┐                         │
│              │ Claude Client       │                         │
│              │ Wrapper             │                         │
│              └──────────┬──────────┘                         │
└─────────────────────────┼───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                    SDK Layer (claude-agent-sdk-go)           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Client API  │  │ Control     │  │ Types & Options     │  │
│  │ (Streaming) │  │ Protocol    │  │                     │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## 开发计划

### Phase 1: 基础架构与核心客户端封装 (Week 1-2)

#### Task 1.1: 项目初始化
- [ ] 创建项目目录结构
- [ ] 初始化 Go module
- [ ] 添加 claude-agent-sdk-go 依赖
- [ ] 配置基础构建工具 (Makefile)

#### Task 1.2: Claude Client Wrapper
- [ ] 实现 `ClaudeClientWrapper` 结构体
  - 封装 SDK 的 `Client` 接口
  - 管理连接生命周期
  - 提供消息流通道
- [ ] 实现消息处理管道
  - 将 SDK 的 `MessageIterator` 转换为应用层事件
  - 处理 `AssistantMessage`、`UserMessage`、`ResultMessage`
- [ ] 实现错误处理与重连机制

#### Task 1.3: 会话管理器 (Session Manager)
- [ ] 定义 `Session` 结构体
  ```go
  type Session struct {
      ID           string           // CLI Session UUID
      ContextID    string           // Context Session ID
      CreatedAt    time.Time
      LastActiveAt time.Time
      Messages     []MessageRecord
      WorkingDir   string
  }
  ```
- [ ] 实现 `SessionManager`
  - `CreateSession()` - 创建新会话
  - `ResumeSession(sessionID)` - 恢复会话
  - `ListSessions()` - 列出历史会话
  - `ForkSession(sessionID)` - 分叉会话
- [ ] 实现会话持久化（JSON 文件存储）

### Phase 2: 发现服务与命令面板 (Week 3-4)

#### Task 2.1: Discovery Service
- [ ] 实现 `DiscoveryService`
  - `DiscoverSlashCommands()` - 发现 Claude 原生指令
  - `GetMcpTools()` - 获取 MCP 工具列表
  - `GetRegisteredSkills()` - 获取已注册 Skills
  - `GetAgents()` - 获取 Agent 定义
- [ ] 实现缓存机制
  - 缓存发现结果，避免重复查询
  - 支持手动刷新

#### Task 2.2: Command Registry
- [ ] 定义统一的命令接口
  ```go
  type Command interface {
      Name() string
      Description() string
      Type() CommandType  // SlashCommand, Skill, Agent, MCPTool
      Execute(ctx context.Context, args string) error
  }
  ```
- [ ] 实现命令注册表
  - 注册 Slash Commands
  - 注册 Skills
  - 注册 Agents
  - 注册 MCP Tools

#### Task 2.3: Command Palette Service
- [ ] 实现命令搜索/过滤
- [ ] 实现命令执行调度
- [ ] 提供命令补全建议

### Phase 3: 权限管理系统 (Week 5-6)

#### Task 3.1: Permission Manager
- [ ] 实现 `PermissionManager`
  - 维护当前权限模式
  - 管理权限规则列表
  - 处理权限请求回调
- [ ] 实现权限决策逻辑
  ```go
  func (pm *PermissionManager) HandlePermissionRequest(
      ctx context.Context,
      toolName string,
      input map[string]any,
      permCtx ToolPermissionContext,
  ) (PermissionResult, error)
  ```
- [ ] 实现权限规则持久化

#### Task 3.2: Permission UI Backend
- [ ] 实现权限请求事件通道
  - 将 SDK 回调转换为 UI 事件
- [ ] 实现权限决策 API
  - `ApproveToolUse(requestID)`
  - `DenyToolUse(requestID, reason)`
  - `ApproveAllForTool(toolName)`
- [ ] 实现权限模式切换 API
  - `SetMode(mode PermissionMode)`

#### Task 3.3: 批量权限操作
- [ ] 实现批量批准/拒绝
- [ ] 实现权限规则模板
- [ ] 实现信任工具列表管理

### Phase 4: UI 层实现 (Week 7-10)

#### Task 4.1: UI 框架选型与搭建
- [ ] 选择 UI 框架（推荐：Fyne / Wails / Gio）
  - **Fyne**: 纯 Go，跨平台，适合桌面应用
  - **Wails**: Go + Web 技术，现代化 UI
  - **Gio**: 即时模式 GUI，高性能
- [ ] 搭建基础窗口框架
- [ ] 实现主题系统

#### Task 4.2: 会话管理 UI
- [ ] 会话列表视图
  - 显示历史会话
  - 支持搜索/过滤
  - 支持恢复/删除
- [ ] 会话详情视图
  - 消息历史展示
  - 消息输入区域
  - 流式输出显示

#### Task 4.3: 命令面板 UI
- [ ] 命令搜索弹窗
  - 快捷键唤起 (Cmd/Ctrl+K)
  - 模糊搜索
  - 分类展示
- [ ] 命令详情展示
  - 描述、参数、用法
- [ ] 命令执行反馈

#### Task 4.4: 权限交互 UI
- [ ] 权限请求对话框
  - 显示工具名称、输入参数
  - 显示 CLI 建议
  - 批准/拒绝按钮
  - "信任此工具"选项
- [ ] 权限模式切换栏
  - 四种模式快速切换
  - 当前模式指示器
- [ ] 权限规则管理界面
  - 规则列表
  - 添加/编辑/删除规则

### Phase 5: 集成与优化 (Week 11-12)

#### Task 5.1: 端到端集成测试
- [ ] 会话创建与恢复测试
- [ ] 命令发现与执行测试
- [ ] 权限交互流程测试

#### Task 5.2: 性能优化
- [ ] 消息流处理优化
- [ ] UI 渲染性能优化
- [ ] 内存使用优化

#### Task 5.3: 文档与打包
- [ ] 用户文档
- [ ] API 文档
- [ ] 跨平台打包脚本

---

## 关键代码示例

### 1. Claude Client Wrapper 初始化

```go
package app

import (
    "context"
    "github.com/tea4go/claude-agent-sdk-go"
)

type ClaudeClientWrapper struct {
    client    claudecode.Client
    msgChan   chan MessageEvent
    errChan   chan error
}

func NewClaudeClientWrapper(opts ...claudecode.Option) (*ClaudeClientWrapper, error) {
    w := &ClaudeClientWrapper{
        msgChan: make(chan MessageEvent, 100),
        errChan: make(chan error, 1),
    }

    // 添加必要的选项
    opts = append(opts,
        claudecode.WithCanUseTool(w.handlePermission),
        claudecode.WithPermissionPromptToolName("stdio"),
    )

    client, err := claudecode.NewClient(opts...)
    if err != nil {
        return nil, err
    }
    w.client = client

    return w, nil
}

func (w *ClaudeClientWrapper) Connect(ctx context.Context) error {
    return w.client.Connect(ctx)
}

func (w *ClaudeClientWrapper) Query(ctx context.Context, prompt string) error {
    iter := w.client.Query(ctx, prompt)
    go w.processMessages(iter)
    return nil
}

func (w *ClaudeClientWrapper) processMessages(iter claudecode.MessageIterator) {
    for iter.Next() {
        msg := iter.Message()
        switch m := msg.(type) {
        case *claudecode.AssistantMessage:
            w.msgChan <- MessageEvent{Type: "assistant", Message: m}
        case *claudecode.UserMessage:
            w.msgChan <- MessageEvent{Type: "user", Message: m}
        case *claudecode.ResultMessage:
            w.msgChan <- MessageEvent{Type: "result", Message: m}
        }
    }
    if err := iter.Err(); err != nil {
        w.errChan <- err
    }
}
```

### 2. 会话管理器

```go
package app

import (
    "context"
    "encoding/json"
    "os"
    "path/filepath"
    "time"

    "github.com/google/uuid"
    "github.com/tea4go/claude-agent-sdk-go"
)

type Session struct {
    ID           string          `json:"id"`
    CLISessionID string          `json:"cli_session_id,omitempty"`
    CreatedAt    time.Time       `json:"created_at"`
    LastActiveAt time.Time       `json:"last_active_at"`
    Messages     []MessageRecord `json:"messages"`
    WorkingDir   string          `json:"working_dir"`
}

type SessionManager struct {
    sessions    map[string]*Session
    active      *Session
    storagePath string
    client      *ClaudeClientWrapper
}

func NewSessionManager(storagePath string, client *ClaudeClientWrapper) *SessionManager {
    sm := &SessionManager{
        sessions:    make(map[string]*Session),
        storagePath: storagePath,
        client:      client,
    }
    sm.loadSessions()
    return sm
}

func (sm *SessionManager) CreateSession(ctx context.Context, workingDir string) (*Session, error) {
    session := &Session{
        ID:           uuid.New().String(),
        CreatedAt:    time.Now(),
        LastActiveAt: time.Now(),
        WorkingDir:   workingDir,
        Messages:     []MessageRecord{},
    }
    sm.sessions[session.ID] = session
    sm.active = session
    sm.saveSessions()
    return session, nil
}

func (sm *SessionManager) ResumeSession(ctx context.Context, sessionID string) (*Session, error) {
    session, ok := sm.sessions[sessionID]
    if !ok {
        return nil, fmt.Errorf("session not found: %s", sessionID)
    }

    // 使用 SDK 的 Resume 功能
    opts := []claudecode.Option{
        claudecode.WithResume(session.CLISessionID),
        claudecode.WithWorkingDir(session.WorkingDir),
    }

    // 重新创建客户端连接
    client, err := NewClaudeClientWrapper(opts...)
    if err != nil {
        return nil, err
    }

    if err := client.Connect(ctx); err != nil {
        return nil, err
    }

    sm.client = client
    sm.active = session
    session.LastActiveAt = time.Now()
    sm.saveSessions()

    return session, nil
}
```

### 3. Discovery Service

```go
package app

import (
    "context"
    "sync"

    "github.com/tea4go/claude-agent-sdk-go"
)

type DiscoveryService struct {
    mu              sync.RWMutex
    slashCommands   []claudecode.SlashCommand
    mcpTools        []MCPToolInfo
    cached          bool
}

func (ds *DiscoveryService) DiscoverSlashCommands(ctx context.Context, opts ...claudecode.Option) error {
    commands, err := claudecode.DiscoverSlashCommands(ctx, opts...)
    if err != nil {
        return err
    }

    ds.mu.Lock()
    ds.slashCommands = commands
    ds.cached = true
    ds.mu.Unlock()

    return nil
}

func (ds *DiscoveryService) GetSlashCommands() []claudecode.SlashCommand {
    ds.mu.RLock()
    defer ds.mu.RUnlock()
    return ds.slashCommands
}

func (ds *DiscoveryService) GetMcpTools(ctx context.Context, client claudecode.Client) ([]MCPToolInfo, error) {
    status, err := client.GetMcpStatus(ctx)
    if err != nil {
        return nil, err
    }

    var tools []MCPToolInfo
    for _, server := range status.Servers {
        if server.Status == claudecode.McpServerConnectionStatusConnected {
            for _, tool := range server.Tools {
                tools = append(tools, MCPToolInfo{
                    ServerName: server.Name,
                    ToolName:   tool.Name,
                    Description: tool.Description,
                })
            }
        }
    }

    ds.mu.Lock()
    ds.mcpTools = tools
    ds.mu.Unlock()

    return tools, nil
}
```

### 4. Permission Manager

```go
package app

import (
    "context"
    "sync"

    "github.com/tea4go/claude-agent-sdk-go"
    "github.com/tea4go/claude-agent-sdk-go/internal/control"
)

type PermissionRequest struct {
    ID          string
    ToolName    string
    Input       map[string]any
    Suggestions []PermissionUpdate
    Response    chan PermissionDecision
}

type PermissionManager struct {
    mu            sync.RWMutex
    mode          claudecode.PermissionMode
    trustedTools  map[string]bool
    pending       map[string]*PermissionRequest
    requestChan   chan *PermissionRequest
}

func NewPermissionManager() *PermissionManager {
    return &PermissionManager{
        mode:         claudecode.PermissionModeDefault,
        trustedTools: make(map[string]bool),
        pending:      make(map[string]*PermissionRequest),
        requestChan:  make(chan *PermissionRequest, 10),
    }
}

func (pm *PermissionManager) HandlePermissionRequest(
    ctx context.Context,
    toolName string,
    input map[string]any,
    permCtx control.ToolPermissionContext,
) (control.PermissionResult, error) {
    pm.mu.RLock()
    if pm.mode == claudecode.PermissionModeBypassPermissions {
        pm.mu.RUnlock()
        return control.NewPermissionResultAllow(), nil
    }

    // 检查是否为信任工具
    if pm.trustedTools[toolName] {
        pm.mu.RUnlock()
        return control.NewPermissionResultAllow(), nil
    }
    pm.mu.RUnlock()

    // 创建权限请求并发送到 UI
    req := &PermissionRequest{
        ID:          uuid.New().String(),
        ToolName:    toolName,
        Input:       input,
        Suggestions: permCtx.Suggestions,
        Response:    make(chan PermissionDecision, 1),
    }

    pm.requestChan <- req

    // 等待 UI 响应
    select {
    case decision := <-req.Response:
        if decision.Approved {
            if decision.TrustTool {
                pm.mu.Lock()
                pm.trustedTools[toolName] = true
                pm.mu.Unlock()
            }
            return control.NewPermissionResultAllow(), nil
        }
        return control.NewPermissionResultDeny(decision.Reason), nil
    case <-ctx.Done():
        return control.NewPermissionResultDeny("timeout"), ctx.Err()
    }
}

func (pm *PermissionManager) SetMode(mode claudecode.PermissionMode) {
    pm.mu.Lock()
    pm.mode = mode
    pm.mu.Unlock()
}

func (pm *PermissionManager) GetRequestChan() <-chan *PermissionRequest {
    return pm.requestChan
}
```

---

## UI 框架推荐

### 方案 A: Wails (推荐)

**优点：**
- 使用 Web 技术（React/Vue/Svelte）构建现代化 UI
- Go 后端，性能优秀
- 原生窗口，跨平台支持
- 活跃的社区

**适合场景：** 需要现代化 UI、复杂交互

### 方案 B: Fyne

**优点：**
- 纯 Go 实现
- 跨平台支持
- Material Design 风格
- 学习曲线平缓

**适合场景：** 纯 Go 技术栈、快速开发

### 方案 C: Gio

**优点：**
- 即时模式 GUI
- 高性能
- 纯 Go

**适合场景：** 需要极致性能、自定义渲染

---

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| SDK API 变更 | 高 | 使用接口抽象，便于适配 |
| UI 框架学习曲线 | 中 | 选择熟悉的框架，预留学习时间 |
| 权限交互复杂度 | 中 | 设计清晰的交互流程，充分测试 |
| 跨平台兼容性 | 中 | 使用跨平台框架，多平台测试 |

---

## 里程碑

| 里程碑 | 时间 | 交付物 |
|--------|------|--------|
| M1: 核心架构 | Week 2 | Client Wrapper + Session Manager |
| M2: 发现服务 | Week 4 | Discovery Service + Command Registry |
| M3: 权限系统 | Week 6 | Permission Manager + 回调集成 |
| M4: UI 原型 | Week 8 | 基础 UI 框架 + 会话管理 UI |
| M5: 功能完整 | Week 10 | 命令面板 + 权限交互 UI |
| M6: 发布就绪 | Week 12 | 测试完成 + 文档 + 打包 |

---

## 下一步行动

1. **确认 UI 框架选择** - Wails vs Fyne vs Gio
2. **创建项目仓库** - 初始化项目结构
3. **实现 Phase 1** - 核心架构与客户端封装
