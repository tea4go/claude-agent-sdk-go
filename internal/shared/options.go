package shared

import (
	"context"
	"fmt"
	"io"
	"regexp"
)

const (
	// DefaultMaxThinkingTokens is the default maximum number of thinking tokens.
	// DefaultMaxThinkingTokens 是思考 token 的默认上限。
	DefaultMaxThinkingTokens = 8000
)

// PermissionMode represents the different permission handling modes.
//
// PermissionMode 表示不同的权限处理模式。
type PermissionMode string

const (
	// PermissionModeDefault is the standard permission handling mode.
	// PermissionModeDefault 为标准权限处理模式。
	PermissionModeDefault PermissionMode = "default"
	// PermissionModeAcceptEdits automatically accepts all edit permissions.
	// PermissionModeAcceptEdits 自动批准所有编辑类权限。
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"
	// PermissionModePlan enables plan mode for task execution.
	// PermissionModePlan 启用任务执行的计划模式。
	PermissionModePlan PermissionMode = "plan"
	// PermissionModeBypassPermissions bypasses all permission checks.
	// PermissionModeBypassPermissions 绕过所有权限检查。
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// SdkBeta represents a beta feature identifier.
// See https://docs.anthropic.com/en/api/beta-headers
//
// SdkBeta 表示一个 beta 特性标识符。
type SdkBeta string

const (
	// SdkBetaContext1M enables the 1M context window beta feature.
	// SdkBetaContext1M 启用 100 万上下文窗口的 beta 特性。
	SdkBetaContext1M SdkBeta = "context-1m-2025-08-07"
)

// ToolsPreset represents a preset tools configuration.
//
// ToolsPreset 表示一种预设的工具配置。
type ToolsPreset struct {
	Type   string `json:"type"`   // 固定为 "preset"
	Preset string `json:"preset"` // 如 "claude_code"
}

// SkillsAll is the sentinel string value for Options.Skills that enables every
// discovered Skill. Mirrors the Python SDK's skills="all" value.
//
// SkillsAll 是 Options.Skills 的哨兵字符串值，启用所有被发现的 Skill，
// 对应 Python SDK 的 skills="all"。
const SkillsAll = "all"

// SettingSource represents a settings source location.
//
// SettingSource 表示设置的来源位置。
type SettingSource string

const (
	// SettingSourceUser loads user-level settings.
	// SettingSourceUser 加载用户级别设置。
	SettingSourceUser SettingSource = "user"
	// SettingSourceProject loads project-level settings.
	// SettingSourceProject 加载项目级别设置。
	SettingSourceProject SettingSource = "project"
	// SettingSourceLocal loads local/workspace-level settings.
	// SettingSourceLocal 加载本地/工作区级别设置。
	SettingSourceLocal SettingSource = "local"
)

// SandboxNetworkConfig configures network access within sandbox.
//
// SandboxNetworkConfig 配置沙箱内的网络访问。
type SandboxNetworkConfig struct {
	// AllowUnixSockets specifies Unix socket paths accessible in sandbox.
	AllowUnixSockets []string `json:"allowUnixSockets,omitempty"`
	// AllowAllUnixSockets allows all Unix sockets (less secure).
	AllowAllUnixSockets bool `json:"allowAllUnixSockets,omitempty"`
	// AllowLocalBinding allows binding to localhost ports (macOS only).
	AllowLocalBinding bool `json:"allowLocalBinding,omitempty"`
	// HTTPProxyPort is the HTTP proxy port if using custom proxy.
	HTTPProxyPort *int `json:"httpProxyPort,omitempty"`
	// SOCKSProxyPort is the SOCKS5 proxy port if using custom proxy.
	SOCKSProxyPort *int `json:"socksProxyPort,omitempty"`
}

// SandboxIgnoreViolations specifies patterns to ignore during sandbox violations.
//
// SandboxIgnoreViolations 指定在沙箱违规时应忽略的文件/网络模式。
type SandboxIgnoreViolations struct {
	// File paths for which violations should be ignored.
	// 应忽略违规的文件路径。
	File []string `json:"file,omitempty"`
	// Network hosts for which violations should be ignored.
	// 应忽略违规的网络主机。
	Network []string `json:"network,omitempty"`
}

// SandboxSettings configures sandbox behavior for bash command execution.
//
// SandboxSettings 配置 bash 命令执行的沙箱行为。
type SandboxSettings struct {
	// Enabled enables bash sandboxing (macOS/Linux only).
	Enabled bool `json:"enabled,omitempty"`
	// AutoAllowBashIfSandboxed auto-approves bash when sandboxed.
	AutoAllowBashIfSandboxed bool `json:"autoAllowBashIfSandboxed,omitempty"`
	// ExcludedCommands are commands that always bypass sandbox automatically.
	ExcludedCommands []string `json:"excludedCommands,omitempty"`
	// AllowUnsandboxedCommands allows commands to bypass sandbox.
	AllowUnsandboxedCommands bool `json:"allowUnsandboxedCommands,omitempty"`
	// Network configures network access in sandbox.
	Network *SandboxNetworkConfig `json:"network,omitempty"`
	// IgnoreViolations configures which violations to ignore.
	IgnoreViolations *SandboxIgnoreViolations `json:"ignoreViolations,omitempty"`
	// EnableWeakerNestedSandbox for unprivileged Docker (Linux only).
	EnableWeakerNestedSandbox bool `json:"enableWeakerNestedSandbox,omitempty"`
}

// SdkPluginType represents the type of SDK plugin.
//
// SdkPluginType 表示 SDK 插件的类型。
type SdkPluginType string

const (
	// SdkPluginTypeLocal represents a local plugin loaded from the filesystem.
	// SdkPluginTypeLocal 表示从文件系统加载的本地插件。
	SdkPluginTypeLocal SdkPluginType = "local"
)

// SdkPluginConfig represents a plugin configuration.
//
// SdkPluginConfig 表示一个插件配置。
type SdkPluginConfig struct {
	// Type is the plugin type (currently only "local" is supported).
	// Type 为插件类型（目前仅支持 "local"）。
	Type SdkPluginType `json:"type"`
	// Path is the filesystem path to the plugin directory.
	// Path 为插件目录的文件系统路径。
	Path string `json:"path"`
}

// SkillRegistryConfig configures an external directory containing Skill
// directories. Each selected Skill directory is exposed to Claude CLI through a
// temporary local plugin wrapper.
//
// SkillRegistryConfig 配置一个包含多个 Skill 子目录的外部目录。每个被选中的
// Skill 目录会通过一个临时本地插件包装器暴露给 Claude CLI。
type SkillRegistryConfig struct {
	// Root is the filesystem directory containing one subdirectory per Skill.
	// Root 是包含多个 Skill 子目录的文件系统目录。
	Root string `json:"root"`
	// Names is the list of Skill directory names to expose. An empty list means
	// expose every direct child directory that contains SKILL.md.
	// Names 是要暴露的 Skill 目录名列表。空列表表示暴露所有包含 SKILL.md 的直接子目录。
	Names []string `json:"names,omitempty"`
	// PluginName is the generated local plugin name used for scoped Skill tools.
	// PluginName 是为带作用域的 Skill 工具生成的本地插件名。
	PluginName string `json:"plugin_name,omitempty"`
}

// OutputFormat specifies the format for structured output.
// Wire format: {"type": "json_schema", "schema": {...}}
//
// OutputFormat 指定结构化输出的格式。线格式为 {"type": "json_schema", "schema": {...}}。
type OutputFormat struct {
	Type   string         `json:"type"`   // 固定为 "json_schema"
	Schema map[string]any `json:"schema"` // JSON Schema 定义
}

// AgentModel represents the model to use for an agent.
//
// AgentModel 表示为某个子代理使用的模型。
type AgentModel string

const (
	// AgentModelSonnet specifies Claude Sonnet model for the agent.
	// AgentModelSonnet 为子代理指定 Claude Sonnet 模型。
	AgentModelSonnet AgentModel = "sonnet"
	// AgentModelOpus specifies Claude Opus model for the agent.
	// AgentModelOpus 为子代理指定 Claude Opus 模型。
	AgentModelOpus AgentModel = "opus"
	// AgentModelHaiku specifies Claude Haiku model for the agent.
	// AgentModelHaiku 为子代理指定 Claude Haiku 模型。
	AgentModelHaiku AgentModel = "haiku"
	// AgentModelInherit specifies the agent should inherit the parent's model.
	// AgentModelInherit 指定子代理继承父级的模型。
	AgentModelInherit AgentModel = "inherit"
)

// EffortLevel controls how many tokens Claude spends per response, trading off
// thoroughness against token efficiency. Maps to the CLI's --effort flag.
//
// EffortLevel 控制 Claude 每次响应投入的 token 量，在“周全”与“token 效率”之间
// 权衡，对应 CLI 的 --effort 参数。
type EffortLevel string

const (
	// EffortLow minimizes token usage.
	// EffortLow 最小化 token 用量。
	EffortLow EffortLevel = "low"
	// EffortMedium balances token usage and thoroughness.
	// EffortMedium 在 token 用量与周全性之间取平衡。
	EffortMedium EffortLevel = "medium"
	// EffortHigh favors thoroughness over token efficiency.
	// EffortHigh 优先周全性而非 token 效率。
	EffortHigh EffortLevel = "high"
	// EffortXHigh requests the highest effort (model-dependent).
	// EffortXHigh 请求最高投入（依赖具体模型）。
	EffortXHigh EffortLevel = "xhigh"
	// EffortMax requests maximum effort (model-dependent, session-only).
	// EffortMax 请求最大投入（依赖具体模型，仅会话内有效）。
	EffortMax EffortLevel = "max"
)

// AgentDefinition defines a programmatic subagent.
//
// AgentDefinition 定义一个以编程方式声明的子代理。
type AgentDefinition struct {
	// Description is a brief description of the agent's purpose.
	Description string `json:"description"`

	// Prompt is the agent's system prompt.
	Prompt string `json:"prompt"`

	// Tools is an optional list of tools available to the agent.
	Tools []string `json:"tools,omitempty"`

	// Model specifies which model the agent should use.
	Model AgentModel `json:"model,omitempty"`
}

// Options configures the Claude Agent SDK behavior.
//
// Options 集中配置 Claude Agent SDK 的行为，涵盖工具控制、系统提示与模型、
// 权限与安全、会话与状态、MCP 集成、沙箱、插件、回调与钩子等。
// 带 json:"-" 的字段不会序列化给 CLI，仅在进程内使用。
type Options struct {
	// Tool Control
	// 工具控制：允许/禁止的工具名列表
	AllowedTools    []string `json:"allowed_tools,omitempty"`
	DisallowedTools []string `json:"disallowed_tools,omitempty"`

	// Tools configures available tools.
	// Can be []string (list of tool names) or ToolsPreset (preset configuration).
	Tools any `json:"tools,omitempty"`

	// Beta Features
	// beta 特性开关列表
	Betas []SdkBeta `json:"betas,omitempty"`

	// System Prompts & Model
	// 系统提示与模型配置
	SystemPrompt       *string `json:"system_prompt,omitempty"`
	AppendSystemPrompt *string `json:"append_system_prompt,omitempty"`
	Model              *string `json:"model,omitempty"`
	FallbackModel      *string `json:"fallback_model,omitempty"`
	Effort             *string `json:"effort,omitempty"`
	MaxThinkingTokens  int     `json:"max_thinking_tokens,omitempty"`

	// Budget & Billing
	// 预算与计费
	MaxBudgetUSD *float64 `json:"max_budget_usd,omitempty"`
	User         *string  `json:"user,omitempty"`

	// Buffer Configuration (internal)
	// 缓冲区配置（内部）
	MaxBufferSize *int `json:"max_buffer_size,omitempty"`

	// Permission & Safety System
	// 权限与安全系统
	PermissionMode           *PermissionMode `json:"permission_mode,omitempty"`
	PermissionPromptToolName *string         `json:"permission_prompt_tool_name,omitempty"`

	// Session & State Management
	// 会话与状态管理
	ContinueConversation bool            `json:"continue_conversation,omitempty"`
	Resume               *string         `json:"resume,omitempty"`
	SessionID            *string         `json:"session_id,omitempty"`
	MaxTurns             int             `json:"max_turns,omitempty"`
	Settings             *string         `json:"settings,omitempty"`
	ForkSession          bool            `json:"fork_session,omitempty"`
	SettingSources       []SettingSource `json:"setting_sources,omitempty"`

	// Skills controls which filesystem-discovered Skills are exposed to the model.
	// Accepts the string "all" (SkillsAll) to enable every discovered Skill, a
	// []string of Skill names to enable only those, or an empty []string{} to
	// disable all. When non-nil and SettingSources is unset, SettingSources
	// defaults to [user, project] so the CLI discovers installed Skills.
	// Matches the Python SDK's skills option (see _apply_skills_defaults).
	Skills any `json:"skills,omitempty"`

	// Partial Message Streaming
	IncludePartialMessages bool `json:"include_partial_messages,omitempty"`

	// EnableFileCheckpointing enables file change tracking for rewind support.
	// When enabled, files can be rewound to their state at any user message
	// using Client.RewindFiles().
	EnableFileCheckpointing bool `json:"enable_file_checkpointing,omitempty"`

	// Agent Definitions
	// 子代理定义
	Agents map[string]AgentDefinition `json:"agents,omitempty"`

	// File System & Context
	// 文件系统与上下文：工作目录与额外可访问目录
	Cwd     *string  `json:"cwd,omitempty"`
	AddDirs []string `json:"add_dirs,omitempty"`

	// MCP Integration
	// MCP 集成：名称到服务器配置的映射
	McpServers map[string]McpServerConfig `json:"mcp_servers,omitempty"`

	// Sandbox Configuration
	// 沙箱配置
	Sandbox *SandboxSettings `json:"sandbox,omitempty"`

	// Plugin Configurations
	// 插件配置列表
	Plugins []SdkPluginConfig `json:"plugins,omitempty"`

	// SkillRegistries configures external directories of Skills to expose via
	// temporary local plugin wrappers.
	SkillRegistries []SkillRegistryConfig `json:"skill_registries,omitempty"`

	// Extensibility
	// 可扩展性：直接透传给 CLI 的额外参数
	ExtraArgs map[string]*string `json:"extra_args,omitempty"`

	// ExtraEnv specifies additional environment variables for the subprocess.
	// These are merged with the system environment variables.
	ExtraEnv map[string]string `json:"extra_env,omitempty"`

	// In-process skill implementations (not serialized to CLI)
	// Maps skill names to their execution handlers
	SkillImplementations map[string]func(context.Context, string) (string, error) `json:"-"`

	// OutputFormat specifies structured output format with JSON schema.
	// When set, Claude's response will conform to the provided schema.
	OutputFormat *OutputFormat `json:"output_format,omitempty"`

	// CLI Path (for testing and custom installations)
	CLIPath *string `json:"cli_path,omitempty"`

	// DebugWriter specifies where to write debug output from the CLI subprocess.
	// If nil (default), stderr is isolated to a temporary file to prevent deadlocks.
	// Common values: os.Stderr, io.Discard, or a custom io.Writer.
	DebugWriter io.Writer `json:"-"` // Not serialized

	// StderrCallback receives CLI stderr output line-by-line.
	// If set, takes precedence over DebugWriter for stderr handling.
	// Each line is stripped of trailing whitespace and empty lines are skipped.
	// Callback panics are silently recovered to prevent crashing the SDK.
	StderrCallback func(string) `json:"-"` // Not serialized

	// CanUseTool is invoked when CLI requests permission to use a tool.
	// The callback receives the tool name, input parameters, and permission context.
	// Return PermissionResultAllow to permit, PermissionResultDeny to deny.
	// If nil, all tool requests are denied (secure default).
	// Callback panics are recovered to prevent crashing the SDK.
	// Note: The actual types are defined in internal/control to avoid import cycles.
	// Use the claudecode package's WithCanUseTool option for type-safe configuration.
	CanUseTool func(
		ctx context.Context,
		toolName string,
		input map[string]any,
		permCtx any, // Actually control.ToolPermissionContext
	) (any, error) `json:"-"` // Not serialized

	// Hooks contains lifecycle event hook registrations.
	// The actual type is map[control.HookEvent][]control.HookMatcher.
	// Stored as any to avoid import cycles with internal/control package.
	// Use the claudecode package's WithHook option for type-safe configuration.
	Hooks any `json:"-"` // Not serialized
}

// McpServerType represents the type of MCP server.
//
// McpServerType 表示 MCP 服务器的类型。
type McpServerType string

const (
	// McpServerTypeStdio represents a stdio-based MCP server.
	// McpServerTypeStdio 表示基于 stdio 的 MCP 服务器。
	McpServerTypeStdio McpServerType = "stdio"
	// McpServerTypeSSE represents a Server-Sent Events MCP server.
	// McpServerTypeSSE 表示基于 Server-Sent Events 的 MCP 服务器。
	McpServerTypeSSE McpServerType = "sse"
	// McpServerTypeHTTP represents an HTTP-based MCP server.
	// McpServerTypeHTTP 表示基于 HTTP 的 MCP 服务器。
	McpServerTypeHTTP McpServerType = "http"
)

// McpServerConfig represents MCP server configuration.
//
// McpServerConfig 是 MCP 服务器配置的统一接口，GetType 返回具体类型。
type McpServerConfig interface {
	GetType() McpServerType
}

// McpStdioServerConfig configures an MCP stdio server.
//
// McpStdioServerConfig 配置一个基于 stdio 的 MCP 服务器。
type McpStdioServerConfig struct {
	Type    McpServerType     `json:"type"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	// AlwaysLoad, when true, opts the server out of tool-search deferral so
	// all of its tools are always available without a ToolSearch round-trip.
	// Requires Claude Code CLI 2.1.121 or later.
	AlwaysLoad bool `json:"alwaysLoad,omitempty"`
}

// GetType returns the server type for McpStdioServerConfig.
//
// GetType 返回 McpStdioServerConfig 的服务器类型。
func (c *McpStdioServerConfig) GetType() McpServerType {
	return McpServerTypeStdio
}

// McpSSEServerConfig configures an MCP Server-Sent Events server.
//
// McpSSEServerConfig 配置一个基于 Server-Sent Events 的 MCP 服务器。
type McpSSEServerConfig struct {
	Type    McpServerType     `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	// AlwaysLoad, when true, opts the server out of tool-search deferral so
	// all of its tools are always available without a ToolSearch round-trip.
	// Requires Claude Code CLI 2.1.121 or later.
	AlwaysLoad bool `json:"alwaysLoad,omitempty"`
}

// GetType returns the server type for McpSSEServerConfig.
//
// GetType 返回 McpSSEServerConfig 的服务器类型。
func (c *McpSSEServerConfig) GetType() McpServerType {
	return McpServerTypeSSE
}

// McpHTTPServerConfig configures an MCP HTTP server.
//
// McpHTTPServerConfig 配置一个基于 HTTP 的 MCP 服务器。
type McpHTTPServerConfig struct {
	Type    McpServerType     `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	// AlwaysLoad, when true, opts the server out of tool-search deferral so
	// all of its tools are always available without a ToolSearch round-trip.
	// Requires Claude Code CLI 2.1.121 or later.
	AlwaysLoad bool `json:"alwaysLoad,omitempty"`
}

// GetType returns the server type for McpHTTPServerConfig.
//
// GetType 返回 McpHTTPServerConfig 的服务器类型。
func (c *McpHTTPServerConfig) GetType() McpServerType {
	return McpServerTypeHTTP
}

// McpServerTypeSdk represents an in-process SDK MCP server.
//
// McpServerTypeSdk 表示进程内的 SDK MCP 服务器。
const McpServerTypeSdk McpServerType = "sdk"

// McpServer is the interface for in-process SDK MCP servers.
// Implementations must be thread-safe as methods may be called concurrently.
//
// McpServer 是进程内 SDK MCP 服务器的接口。由于方法可能被并发调用，
// 实现必须是线程安全的。
type McpServer interface {
	// Name returns the server name.
	// Name 返回服务器名称。
	Name() string
	// Version returns the server version.
	// Version 返回服务器版本。
	Version() string
	// ListTools returns the available tools.
	// ListTools 返回可用的工具列表。
	ListTools(ctx context.Context) ([]McpToolDefinition, error)
	// CallTool executes a tool by name with the given arguments.
	// CallTool 按名称使用给定参数执行一个工具。
	CallTool(ctx context.Context, name string, args map[string]any) (*McpToolResult, error)
}

// McpSdkServerConfig configures an in-process SDK MCP server.
// The Instance field contains the actual server implementation and is
// excluded from JSON serialization (not sent to CLI).
//
// McpSdkServerConfig 配置一个进程内的 SDK MCP 服务器。Instance 字段包含实际的
// 服务器实现，不参与 JSON 序列化（不会发送给 CLI）。
type McpSdkServerConfig struct {
	Type     McpServerType `json:"type"`
	Name     string        `json:"name"`
	Instance McpServer     `json:"-"` // 不参与 CLI 序列化
	// AlwaysLoad, when true, opts the server out of tool-search deferral so
	// all of its tools are always available without a ToolSearch round-trip.
	// Requires Claude Code CLI 2.1.121 or later.
	AlwaysLoad bool `json:"alwaysLoad,omitempty"`
}

// GetType returns the server type for McpSdkServerConfig.
//
// GetType 返回 McpSdkServerConfig 的服务器类型。
func (c *McpSdkServerConfig) GetType() McpServerType {
	return McpServerTypeSdk
}

// ToolAnnotations carries optional MCP-spec behavioral hints attached by
// a tool author when defining an SDK MCP tool. Sent to the CLI as part of
// the JSONRPC tools/list response under the "annotations" key.
//
// All fields are pointers so unset fields are omitted from the wire format.
// See MCP spec:
// https://modelcontextprotocol.io/specification/2025-03-26/server/tools#tool
//
// This is the authoring counterpart to McpToolAnnotations in the control
// package, which describes annotations as reported back by the CLI in
// GetMcpStatus responses. The two are kept separate because the CLI strips
// the "Hint" suffix on status responses, so the wire field sets differ.
type ToolAnnotations struct {
	Title           *string `json:"title,omitempty"`
	ReadOnlyHint    *bool   `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool   `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool   `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool   `json:"openWorldHint,omitempty"`
}

// McpToolDefinition describes a tool exposed by an MCP server.
//
// McpToolDefinition 描述 MCP 服务器暴露的一个工具。
type McpToolDefinition struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	InputSchema map[string]any   `json:"inputSchema"`
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
}

// McpToolResult represents the result of a tool call.
//
// McpToolResult 表示一次工具调用的结果；IsError 为 true 时表示调用失败。
type McpToolResult struct {
	Content []McpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// McpContent represents content returned by a tool.
// Supports both text and image content types.
//
// McpContent 表示工具返回的内容，同时支持文本与图像两种类型。
type McpContent struct {
	Type     string `json:"type"` // "text" 或 "image"
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`     // 图像的 base64 数据
	MimeType string `json:"mimeType,omitempty"` // 图像的 MIME 类型
}

// Validate checks the options for valid values and constraints.
// uuidPattern 用于校验 UUID 格式（8-4-4-4-12 十六进制）。
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isValidUUID reports whether value is a canonical lowercase/uppercase
// hyphenated UUID (8-4-4-4-12 hex digits).
//
// isValidUUID 报告 value 是否为规范的连字符形式 UUID（大小写均可，8-4-4-4-12 十六进制位）。
func isValidUUID(value string) bool {
	matched := uuidPattern.MatchString(value)
	return matched
}

// Validate 校验选项的取值与约束（非负 token/轮数、UUID 格式、会话互斥以及工具冲突）。
func (o *Options) Validate() error {
	// 校验 MaxThinkingTokens
	if o.MaxThinkingTokens < 0 {
		return fmt.Errorf("MaxThinkingTokens must be non-negative, got %d", o.MaxThinkingTokens)
	}

	// 校验 MaxTurns
	if o.MaxTurns < 0 {
		return fmt.Errorf("MaxTurns must be non-negative, got %d", o.MaxTurns)
	}

	// 校验 SessionID 及其与 Resume 的互斥关系
	if o.SessionID != nil {
		if !isValidUUID(*o.SessionID) {
			return fmt.Errorf("SessionID must be a valid UUID, got %q", *o.SessionID)
		}
		if o.Resume != nil {
			return fmt.Errorf("SessionID and Resume cannot be used together")
		}
	}

	// 校验工具冲突（同一工具同时出现在允许与禁止列表中）
	allowedSet := make(map[string]bool)
	for _, tool := range o.AllowedTools {
		allowedSet[tool] = true
	}

	for _, tool := range o.DisallowedTools {
		if allowedSet[tool] {
			return fmt.Errorf("tool '%s' cannot be in both AllowedTools and DisallowedTools", tool)
		}
	}

	return nil
}

// NewOptions creates Options with default values.
//
// NewOptions 创建带默认值的 Options（初始化各集合并设置默认思考 token 上限）。
func NewOptions() *Options {
	return &Options{
		AllowedTools:      []string{},
		DisallowedTools:   []string{},
		Betas:             []SdkBeta{},
		MaxThinkingTokens: DefaultMaxThinkingTokens,
		AddDirs:           []string{},
		McpServers:        make(map[string]McpServerConfig),
		Plugins:           []SdkPluginConfig{},
		ExtraArgs:         make(map[string]*string),
		ExtraEnv:          make(map[string]string),
	}
}
