package claudecode

import (
	"context"
	"io"
	"os"

	"github.com/tea4go/claude-agent-sdk-go/internal/control"
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

// Options contains configuration for Claude Code CLI interactions.
//
// Options 包含与 Claude Code CLI 交互的配置。
type Options = shared.Options

// PermissionMode defines the permission handling mode.
//
// PermissionMode 定义权限处理模式。
type PermissionMode = shared.PermissionMode

// McpServerType defines the type of MCP server.
//
// McpServerType 定义 MCP 服务器的类型。
type McpServerType = shared.McpServerType

// McpServerConfig represents an MCP server configuration.
//
// McpServerConfig 表示一个 MCP 服务器配置。
type McpServerConfig = shared.McpServerConfig

// McpStdioServerConfig represents a stdio MCP server configuration.
//
// McpStdioServerConfig 表示一个 stdio 类型的 MCP 服务器配置。
type McpStdioServerConfig = shared.McpStdioServerConfig

// McpSSEServerConfig represents an SSE MCP server configuration.
//
// McpSSEServerConfig 表示一个 SSE 类型的 MCP 服务器配置。
type McpSSEServerConfig = shared.McpSSEServerConfig

// McpHTTPServerConfig represents an HTTP MCP server configuration.
//
// McpHTTPServerConfig 表示一个 HTTP 类型的 MCP 服务器配置。
type McpHTTPServerConfig = shared.McpHTTPServerConfig

// SdkBeta represents a beta feature identifier.
//
// SdkBeta 表示一个 beta 特性标识符。
type SdkBeta = shared.SdkBeta

// ToolsPreset represents a preset tools configuration.
//
// ToolsPreset 表示一个预设的工具配置。
type ToolsPreset = shared.ToolsPreset

// SettingSource represents a settings source location.
//
// SettingSource 表示一个设置来源位置。
type SettingSource = shared.SettingSource

// SandboxSettings configures sandbox behavior for bash command execution.
//
// SandboxSettings 配置 bash 命令执行的沙箱行为。
type SandboxSettings = shared.SandboxSettings

// SandboxNetworkConfig configures network access within sandbox.
//
// SandboxNetworkConfig 配置沙箱内的网络访问。
type SandboxNetworkConfig = shared.SandboxNetworkConfig

// SandboxIgnoreViolations specifies patterns to ignore during sandbox violations.
//
// SandboxIgnoreViolations 指定在沙箱违规时应忽略的模式。
type SandboxIgnoreViolations = shared.SandboxIgnoreViolations

// SdkPluginType represents the type of SDK plugin.
//
// SdkPluginType 表示 SDK 插件的类型。
type SdkPluginType = shared.SdkPluginType

// SdkPluginConfig represents a plugin configuration.
//
// SdkPluginConfig 表示一个插件配置。
type SdkPluginConfig = shared.SdkPluginConfig

// SkillRegistryConfig represents an external directory of Skills that should be
// exposed through a temporary local plugin wrapper.
//
// SkillRegistryConfig 表示一个外部技能目录，应通过临时本地插件封装对外暴露。
type SkillRegistryConfig = shared.SkillRegistryConfig

// OutputFormat specifies the format for structured output.
//
// OutputFormat 指定结构化输出的格式。
type OutputFormat = shared.OutputFormat

// EffortLevel controls how many tokens Claude spends per response.
//
// EffortLevel 控制 Claude 在每次响应中花费多少 token。
type EffortLevel = shared.EffortLevel

// CanUseToolCallback is invoked when CLI requests permission to use a tool.
// The callback receives tool name, input parameters, and permission context.
// Return PermissionResultAllow to permit, PermissionResultDeny to deny.
// The callback must be thread-safe as it may be invoked concurrently.
//
// CanUseToolCallback 在 CLI 请求使用某工具的权限时被调用。
// 回调接收工具名、输入参数与权限上下文；返回 PermissionResultAllow 表示允许，
// PermissionResultDeny 表示拒绝。回调可能被并发调用，必须线程安全。
type CanUseToolCallback = control.CanUseToolCallback

// PermissionResult is the interface for permission callback results.
// Implementations are PermissionResultAllow and PermissionResultDeny.
//
// PermissionResult 是权限回调结果的接口；实现为 PermissionResultAllow 与 PermissionResultDeny。
type PermissionResult = control.PermissionResult

// PermissionResultAllow permits tool execution with optional modifications.
// Use NewPermissionResultAllow() to create with proper defaults.
//
// PermissionResultAllow 允许工具执行，并可附带可选修改。请使用 NewPermissionResultAllow() 创建。
type PermissionResultAllow = control.PermissionResultAllow

// PermissionResultDeny prevents tool execution.
// Use NewPermissionResultDeny(message) to create with proper defaults.
//
// PermissionResultDeny 阻止工具执行。请使用 NewPermissionResultDeny(message) 创建。
type PermissionResultDeny = control.PermissionResultDeny

// ToolPermissionContext provides context for permission callbacks.
// Contains suggestions from CLI for permission decisions.
//
// ToolPermissionContext 为权限回调提供上下文，包含 CLI 对权限决策的建议。
type ToolPermissionContext = control.ToolPermissionContext

// PermissionUpdate represents a dynamic permission rule update.
//
// PermissionUpdate 表示一次动态的权限规则更新。
type PermissionUpdate = control.PermissionUpdate

// PermissionRuleValue represents a permission rule.
//
// PermissionRuleValue 表示一条权限规则。
type PermissionRuleValue = control.PermissionRuleValue

// PermissionUpdateType specifies the type of permission update.
//
// PermissionUpdateType 指定权限更新的类型。
type PermissionUpdateType = control.PermissionUpdateType

// Re-export constants
//
// 重新导出常量。
const (
	PermissionModeDefault           = shared.PermissionModeDefault
	PermissionModeAcceptEdits       = shared.PermissionModeAcceptEdits
	PermissionModePlan              = shared.PermissionModePlan
	PermissionModeBypassPermissions = shared.PermissionModeBypassPermissions
	McpServerTypeStdio              = shared.McpServerTypeStdio
	McpServerTypeSSE                = shared.McpServerTypeSSE
	McpServerTypeHTTP               = shared.McpServerTypeHTTP
	SdkBetaContext1M                = shared.SdkBetaContext1M
	SettingSourceUser               = shared.SettingSourceUser
	SettingSourceProject            = shared.SettingSourceProject
	SettingSourceLocal              = shared.SettingSourceLocal
	SdkPluginTypeLocal              = shared.SdkPluginTypeLocal
	EffortLow                       = shared.EffortLow
	EffortMedium                    = shared.EffortMedium
	EffortHigh                      = shared.EffortHigh
	EffortXHigh                     = shared.EffortXHigh
	EffortMax                       = shared.EffortMax
)

// Permission update type constants
//
// 权限更新类型常量。
const (
	PermissionUpdateTypeAddRules          = control.PermissionUpdateTypeAddRules
	PermissionUpdateTypeReplaceRules      = control.PermissionUpdateTypeReplaceRules
	PermissionUpdateTypeRemoveRules       = control.PermissionUpdateTypeRemoveRules
	PermissionUpdateTypeSetMode           = control.PermissionUpdateTypeSetMode
	PermissionUpdateTypeAddDirectories    = control.PermissionUpdateTypeAddDirectories
	PermissionUpdateTypeRemoveDirectories = control.PermissionUpdateTypeRemoveDirectories
)

// Option configures Options using the functional options pattern.
//
// Option 使用函数式选项模式配置 Options。
type Option func(*Options)

// WithAllowedTools sets the allowed tools list.
//
// WithAllowedTools 设置允许使用的工具列表。
func WithAllowedTools(tools ...string) Option {
	return func(o *Options) {
		o.AllowedTools = tools
	}
}

// WithDisallowedTools sets the disallowed tools list.
//
// WithDisallowedTools 设置禁止使用的工具列表。
func WithDisallowedTools(tools ...string) Option {
	return func(o *Options) {
		o.DisallowedTools = tools
	}
}

// WithSkillImplementations sets in-process skill implementations directly.
//
// WithSkillImplementations 直接设置进程内的技能实现。
func WithSkillImplementations(skills map[string]func(context.Context, string) (string, error)) Option {
	return func(o *Options) {
		o.SkillImplementations = skills
	}
}

// WithSkillImplementation registers a single in-process skill implementation.
//
// WithSkillImplementation 注册单个进程内技能实现。
func WithSkillImplementation(name string, handler func(context.Context, string) (string, error)) Option {
	return func(o *Options) {
		if o.SkillImplementations == nil {
			o.SkillImplementations = map[string]func(context.Context, string) (string, error){}
		}
		o.SkillImplementations[name] = handler
	}
}

// WithTools sets available tools as a list of tool names.
//
// WithTools 以工具名列表的形式设置可用工具。
func WithTools(tools ...string) Option {
	return func(o *Options) {
		o.Tools = tools
	}
}

// WithToolsPreset sets tools to a preset configuration.
//
// WithToolsPreset 将工具设置为一个预设配置。
func WithToolsPreset(preset string) Option {
	return func(o *Options) {
		o.Tools = ToolsPreset{
			Type:   "preset",
			Preset: preset,
		}
	}
}

// WithClaudeCodeTools sets tools to the claude_code preset.
//
// WithClaudeCodeTools 将工具设置为 claude_code 预设。
func WithClaudeCodeTools() Option {
	return WithToolsPreset("claude_code")
}

// WithSystemPrompt sets the system prompt.
//
// WithSystemPrompt 设置系统提示词。
func WithSystemPrompt(prompt string) Option {
	return func(o *Options) {
		o.SystemPrompt = &prompt
	}
}

// WithAppendSystemPrompt sets the append system prompt.
//
// WithAppendSystemPrompt 设置追加到系统提示词后的内容。
func WithAppendSystemPrompt(prompt string) Option {
	return func(o *Options) {
		o.AppendSystemPrompt = &prompt
	}
}

// WithModel sets the model to use.
//
// WithModel 设置要使用的模型。
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = &model
	}
}

// WithFallbackModel sets the fallback model when primary model is unavailable.
//
// WithFallbackModel 设置主模型不可用时的回退模型。
func WithFallbackModel(model string) Option {
	return func(o *Options) {
		o.FallbackModel = &model
	}
}

// WithEffort sets the effort level (--effort), controlling how many tokens
// Claude spends per response. Known levels are EffortLow..EffortMax, but any
// value is passed through to the CLI to tolerate future levels.
//
// WithEffort 设置努力级别（--effort），控制 Claude 每次响应花费多少 token。
// 已知级别为 EffortLow..EffortMax，但任意值都会直通传递给 CLI 以兼容未来级别。
func WithEffort(effort EffortLevel) Option {
	return func(o *Options) {
		s := string(effort)
		o.Effort = &s
	}
}

// WithMaxBudgetUSD sets the maximum budget in USD for API usage.
//
// WithMaxBudgetUSD 设置 API 使用的最大预算（美元）。
func WithMaxBudgetUSD(budget float64) Option {
	return func(o *Options) {
		o.MaxBudgetUSD = &budget
	}
}

// WithUser sets the user identifier for tracking and billing.
//
// WithUser 设置用于追踪与计费的用户标识符。
func WithUser(user string) Option {
	return func(o *Options) {
		o.User = &user
	}
}

// WithMaxBufferSize sets the maximum buffer size for CLI output.
//
// WithMaxBufferSize 设置 CLI 输出的最大缓冲区大小。
func WithMaxBufferSize(size int) Option {
	return func(o *Options) {
		o.MaxBufferSize = &size
	}
}

// WithMaxThinkingTokens sets the maximum thinking tokens.
//
// WithMaxThinkingTokens 设置最大思考 token 数。
func WithMaxThinkingTokens(tokens int) Option {
	return func(o *Options) {
		o.MaxThinkingTokens = tokens
	}
}

// WithPermissionMode sets the permission mode.
//
// WithPermissionMode 设置权限模式。
func WithPermissionMode(mode PermissionMode) Option {
	return func(o *Options) {
		o.PermissionMode = &mode
	}
}

// WithPermissionPromptToolName sets the permission prompt tool name.
//
// WithPermissionPromptToolName 设置权限提示工具的名称。
func WithPermissionPromptToolName(toolName string) Option {
	return func(o *Options) {
		o.PermissionPromptToolName = &toolName
	}
}

// WithContinueConversation enables conversation continuation.
//
// WithContinueConversation 启用对话延续。
func WithContinueConversation(continueConversation bool) Option {
	return func(o *Options) {
		o.ContinueConversation = continueConversation
	}
}

// WithResume sets the session ID to resume.
//
// WithResume 设置要恢复（resume）的会话 ID。
func WithResume(sessionID string) Option {
	return func(o *Options) {
		o.Resume = &sessionID
	}
}

// WithSessionID sets a specific session ID for a new conversation by mapping
// to the CLI --session-id flag. The ID must be a valid UUID and is mutually
// exclusive with WithResume (enforced by Options.Validate before the CLI
// subprocess starts).
//
// WithSessionID 为新对话设置特定的会话 ID，对应 CLI 的 --session-id 参数。
// ID 必须是合法 UUID，且与 WithResume 互斥（在 CLI 子进程启动前由 Options.Validate 强制校验）。
func WithSessionID(sessionID string) Option {
	return func(o *Options) {
		o.SessionID = &sessionID
	}
}

// WithCwd sets the working directory.
//
// WithCwd 设置工作目录。
func WithCwd(cwd string) Option {
	return func(o *Options) {
		o.Cwd = &cwd
	}
}

// WithAddDirs adds directories to the context.
//
// WithAddDirs 向上下文中添加目录。
func WithAddDirs(dirs ...string) Option {
	return func(o *Options) {
		o.AddDirs = dirs
	}
}

// WithMcpServers sets the MCP server configurations.
//
// WithMcpServers 设置 MCP 服务器配置。
func WithMcpServers(servers map[string]McpServerConfig) Option {
	return func(o *Options) {
		o.McpServers = servers
	}
}

// WithSdkMcpServer adds an in-process SDK MCP server by name.
// This is a convenience method for adding SDK MCP servers created with CreateSDKMcpServer.
// Multiple calls accumulate servers.
//
// WithSdkMcpServer 按名称添加一个进程内的 SDK MCP 服务器。
// 这是添加由 CreateSDKMcpServer 创建的 SDK MCP 服务器的便捷方法；多次调用会累积服务器。
//
// Example:
//
//	calculator := claudecode.CreateSDKMcpServer("calculator", "1.0.0", addTool, sqrtTool)
//	client := claudecode.NewClient(
//	    claudecode.WithSdkMcpServer("calc", calculator),
//	    claudecode.WithAllowedTools("mcp__calc__add", "mcp__calc__sqrt"),
//	)
func WithSdkMcpServer(name string, server *McpSdkServerConfig) Option {
	return func(o *Options) {
		if o.McpServers == nil {
			o.McpServers = make(map[string]McpServerConfig)
		}
		o.McpServers[name] = server
	}
}

// WithMaxTurns sets the maximum number of conversation turns.
//
// WithMaxTurns 设置对话的最大轮数。
func WithMaxTurns(turns int) Option {
	return func(o *Options) {
		o.MaxTurns = turns
	}
}

// WithSettings sets the settings file path or JSON string.
//
// WithSettings 设置配置文件路径或 JSON 字符串。
func WithSettings(settings string) Option {
	return func(o *Options) {
		o.Settings = &settings
	}
}

// WithForkSession enables forking to a new session ID when resuming.
// When true, resumed sessions fork to a new session ID rather than
// continuing the previous session.
//
// WithForkSession 启用在恢复会话时派生（fork）到新的会话 ID。
// 为 true 时，被恢复的会话会派生到新的会话 ID，而不是延续原会话。
func WithForkSession(fork bool) Option {
	return func(o *Options) {
		o.ForkSession = fork
	}
}

// WithSettingSources sets which settings sources to load.
// Valid sources are SettingSourceUser, SettingSourceProject, and SettingSourceLocal.
//
// WithSettingSources 设置要加载的配置来源。
// 合法来源为 SettingSourceUser、SettingSourceProject 与 SettingSourceLocal。
func WithSettingSources(sources ...SettingSource) Option {
	return func(o *Options) {
		if sources == nil {
			o.SettingSources = []SettingSource{}
			return
		}
		o.SettingSources = append([]SettingSource{}, sources...)
	}
}

// SkillsAll is the sentinel value for enabling every discovered Skill.
// Passing this to WithSkills enables all Skills found on the filesystem.
//
// SkillsAll 是用于启用所有已发现技能（Skill）的哨兵值。
// 将其传给 WithSkills 会启用文件系统上发现的全部技能。
const SkillsAll = shared.SkillsAll

// WithSkills sets the Skills configuration directly.
// Accepts the string "all" (use SkillsAll) to enable every discovered Skill,
// a []string of Skill names to enable only those, or []string{} to disable all.
// When set, SettingSources defaults to [user, project] if unset so the CLI
// discovers installed Skills. Mirrors the Python SDK's skills option.
//
// WithSkills 直接设置技能（Skill）配置。
// 可接受字符串 "all"（使用 SkillsAll）以启用全部已发现技能、[]string 技能名列表以仅启用这些技能、
// 或 []string{} 以禁用全部技能。设置后，若 SettingSources 未设置则默认为 [user, project]，
// 以便 CLI 发现已安装的技能。对应 Python SDK 的 skills 选项。
func WithSkills(skills any) Option {
	return func(o *Options) {
		o.Skills = skills
	}
}

// WithSkillsAll enables every discovered Skill in the session.
//
// WithSkillsAll 在会话中启用所有已发现的技能。
func WithSkillsAll() Option {
	return func(o *Options) {
		o.Skills = SkillsAll
	}
}

// WithSkillsList enables only the named Skills.
// Names match the name field in SKILL.md or the Skill's directory name.
// Use "plugin:skill" for plugin-provided Skills.
//
// WithSkillsList 仅启用指定名称的技能。
// 名称匹配 SKILL.md 中的 name 字段或技能所在目录名；插件提供的技能使用 "plugin:skill" 形式。
func WithSkillsList(names ...string) Option {
	return func(o *Options) {
		// Always store a non-nil slice so callers can distinguish from unset.
		list := make([]string, len(names))
		copy(list, names)
		o.Skills = list
	}
}

// WithSkillsDisabled disables all Skills in the session.
//
// WithSkillsDisabled 在会话中禁用所有技能。
func WithSkillsDisabled() Option {
	return func(o *Options) {
		o.Skills = []string{}
	}
}

// WithExtraArgs sets arbitrary CLI flags via ExtraArgs.
//
// WithExtraArgs 通过 ExtraArgs 设置任意的 CLI 命令行参数。
func WithExtraArgs(args map[string]*string) Option {
	return func(o *Options) {
		o.ExtraArgs = args
	}
}

// WithCLIPath sets a custom CLI path.
//
// WithCLIPath 设置自定义的 CLI 可执行文件路径。
func WithCLIPath(path string) Option {
	return func(o *Options) {
		o.CLIPath = &path
	}
}

// WithEnv sets environment variables for the subprocess.
// Multiple calls to WithEnv or WithEnvVar merge the values.
// Later calls override earlier ones for the same key.
//
// WithEnv 为子进程设置环境变量。
// 多次调用 WithEnv 或 WithEnvVar 会合并取值；对于相同的键，后续调用会覆盖先前的值。
func WithEnv(env map[string]string) Option {
	return func(o *Options) {
		if o.ExtraEnv == nil {
			o.ExtraEnv = make(map[string]string)
		}
		// Merge pattern - idiomatic Go
		for k, v := range env {
			o.ExtraEnv[k] = v
		}
	}
}

// WithEnvVar sets a single environment variable for the subprocess.
// This is a convenience method for setting individual variables.
//
// WithEnvVar 为子进程设置单个环境变量，是设置单个变量的便捷方法。
func WithEnvVar(key, value string) Option {
	return func(o *Options) {
		if o.ExtraEnv == nil {
			o.ExtraEnv = make(map[string]string)
		}
		o.ExtraEnv[key] = value
	}
}

// WithBetas sets the SDK beta features to enable.
// See https://docs.anthropic.com/en/api/beta-headers
//
// WithBetas 设置要启用的 SDK beta 特性。参见 https://docs.anthropic.com/en/api/beta-headers
func WithBetas(betas ...SdkBeta) Option {
	return func(o *Options) {
		o.Betas = betas
	}
}

// WithSandbox sets the sandbox settings for bash command isolation.
//
// WithSandbox 设置用于隔离 bash 命令执行的沙箱配置。
func WithSandbox(sandbox *SandboxSettings) Option {
	return func(o *Options) {
		o.Sandbox = sandbox
	}
}

// WithSandboxEnabled enables or disables sandbox.
// If sandbox settings don't exist, they are initialized.
//
// WithSandboxEnabled 启用或禁用沙箱；若沙箱配置不存在则会被初始化。
func WithSandboxEnabled(enabled bool) Option {
	return func(o *Options) {
		if o.Sandbox == nil {
			o.Sandbox = &SandboxSettings{}
		}
		o.Sandbox.Enabled = enabled
	}
}

// WithAutoAllowBashIfSandboxed sets whether to auto-approve bash when sandboxed.
// If sandbox settings don't exist, they are initialized.
//
// WithAutoAllowBashIfSandboxed 设置在沙箱模式下是否自动批准 bash 命令；若沙箱配置不存在则会被初始化。
func WithAutoAllowBashIfSandboxed(autoAllow bool) Option {
	return func(o *Options) {
		if o.Sandbox == nil {
			o.Sandbox = &SandboxSettings{}
		}
		o.Sandbox.AutoAllowBashIfSandboxed = autoAllow
	}
}

// WithSandboxExcludedCommands sets commands that always bypass sandbox.
// If sandbox settings don't exist, they are initialized.
//
// WithSandboxExcludedCommands 设置始终绕过沙箱的命令；若沙箱配置不存在则会被初始化。
func WithSandboxExcludedCommands(commands ...string) Option {
	return func(o *Options) {
		if o.Sandbox == nil {
			o.Sandbox = &SandboxSettings{}
		}
		o.Sandbox.ExcludedCommands = commands
	}
}

// WithSandboxNetwork sets the network configuration for sandbox.
// If sandbox settings don't exist, they are initialized.
//
// WithSandboxNetwork 设置沙箱的网络配置；若沙箱配置不存在则会被初始化。
func WithSandboxNetwork(network *SandboxNetworkConfig) Option {
	return func(o *Options) {
		if o.Sandbox == nil {
			o.Sandbox = &SandboxSettings{}
		}
		o.Sandbox.Network = network
	}
}

// WithPlugins sets the plugin configurations.
// This replaces any previously configured plugins.
func WithPlugins(plugins []SdkPluginConfig) Option {
	return func(o *Options) {
		o.Plugins = plugins
	}
}

// WithPlugin appends a single plugin configuration.
// Multiple calls accumulate plugins.
func WithPlugin(plugin SdkPluginConfig) Option {
	return func(o *Options) {
		o.Plugins = append(o.Plugins, plugin)
	}
}

// WithLocalPlugin appends a local plugin by path.
// This is a convenience method for the common case of local plugins.
func WithLocalPlugin(path string) Option {
	return func(o *Options) {
		o.Plugins = append(o.Plugins, SdkPluginConfig{
			Type: SdkPluginTypeLocal,
			Path: path,
		})
	}
}

// WithSkillRegistry exposes selected Skills from an external registry
// directory. The registry root must contain one subdirectory per Skill, each
// with a SKILL.md file. The SDK presents the selected Skills to Claude CLI via
// a temporary local plugin wrapper without copying them into the project or
// user ~/.claude directory. Use SkillRegistryScopedName when explicitly
// instructing Claude to invoke one of these Skills.
func WithSkillRegistry(root string, names ...string) Option {
	return func(o *Options) {
		selected := append([]string(nil), names...)
		o.SkillRegistries = append(o.SkillRegistries, SkillRegistryConfig{
			Root:  root,
			Names: selected,
		})
	}
}

// WithSkillRegistryAll exposes every direct child Skill in an external
// registry directory. A child directory is considered a Skill when it contains
// SKILL.md.
func WithSkillRegistryAll(root string) Option {
	return func(o *Options) {
		o.SkillRegistries = append(o.SkillRegistries, SkillRegistryConfig{
			Root:  root,
			Names: []string{},
		})
	}
}

// WithAgents sets the programmatic agent definitions.
// This replaces any existing agents.
func WithAgents(agents map[string]AgentDefinition) Option {
	return func(o *Options) {
		o.Agents = agents
	}
}

// WithAgent adds or updates a single agent definition.
// Multiple calls merge agents (later calls override same-name agents).
func WithAgent(name string, agent AgentDefinition) Option {
	return func(o *Options) {
		if o.Agents == nil {
			o.Agents = make(map[string]AgentDefinition)
		}
		o.Agents[name] = agent
	}
}

const customTransportMarker = "custom_transport"

// WithTransport sets a custom transport for testing.
// Since Transport is not part of Options struct, this is handled in client creation.
func WithTransport(_ Transport) Option {
	return func(o *Options) {
		// This will be handled in client implementation
		// For now, we'll use a special marker in ExtraArgs
		if o.ExtraArgs == nil {
			o.ExtraArgs = make(map[string]*string)
		}
		marker := customTransportMarker
		o.ExtraArgs["__transport_marker__"] = &marker
	}
}

// NewOptions creates Options with default values using functional options pattern.
func NewOptions(opts ...Option) *Options {
	// Create options with defaults from shared package
	options := shared.NewOptions()

	// Apply functional options
	for _, opt := range opts {
		opt(options)
	}

	return options
}

// WithDebugWriter sets the writer for CLI debug output.
// If not set, stderr is isolated to a temporary file (default behavior).
// Common values: os.Stderr, io.Discard, or a custom io.Writer like bytes.Buffer.
func WithDebugWriter(w io.Writer) Option {
	return func(o *Options) {
		o.DebugWriter = w
	}
}

// WithDebugStderr redirects CLI debug output to os.Stderr.
// This is useful for seeing debug output in real-time during development.
func WithDebugStderr() Option {
	return WithDebugWriter(os.Stderr)
}

// WithDebugDisabled discards all CLI debug output.
// This is more explicit than the default nil behavior but has the same effect.
func WithDebugDisabled() Option {
	return WithDebugWriter(io.Discard)
}

// WithStderrCallback sets a callback for receiving CLI stderr output.
// The callback is invoked for each non-empty line of stderr output.
// Lines are stripped of trailing whitespace before being passed to the callback.
// This takes precedence over WithDebugWriter if both are set.
// Callback panics are silently recovered to prevent crashing the SDK.
func WithStderrCallback(callback func(string)) Option {
	return func(o *Options) {
		o.StderrCallback = callback
	}
}

// OutputFormatJSONSchema creates an OutputFormat for JSON schema constraints.
func OutputFormatJSONSchema(schema map[string]any) *OutputFormat {
	return &OutputFormat{
		Type:   "json_schema",
		Schema: schema,
	}
}

// WithOutputFormat sets the output format for structured responses.
func WithOutputFormat(format *OutputFormat) Option {
	return func(o *Options) {
		o.OutputFormat = format
	}
}

// WithJSONSchema is a convenience function that sets a JSON schema output format.
// This is equivalent to WithOutputFormat(OutputFormatJSONSchema(schema)).
func WithJSONSchema(schema map[string]any) Option {
	return func(o *Options) {
		if schema == nil {
			o.OutputFormat = nil
			return
		}
		o.OutputFormat = OutputFormatJSONSchema(schema)
	}
}

// WithIncludePartialMessages enables streaming of partial message updates.
// When true, StreamEvent messages are emitted during response generation,
// providing real-time progress as the model generates content.
func WithIncludePartialMessages(include bool) Option {
	return func(o *Options) {
		o.IncludePartialMessages = include
	}
}

// WithPartialStreaming is a convenience function that enables partial message streaming.
// Equivalent to WithIncludePartialMessages(true).
func WithPartialStreaming() Option {
	return WithIncludePartialMessages(true)
}

// WithEnableFileCheckpointing enables or disables file checkpointing.
// When enabled, file changes are tracked during the session and can be
// rewound to their state at any user message using Client.RewindFiles().
func WithEnableFileCheckpointing(enable bool) Option {
	return func(o *Options) {
		o.EnableFileCheckpointing = enable
	}
}

// WithFileCheckpointing enables file checkpointing.
// Equivalent to WithEnableFileCheckpointing(true).
// This is the recommended convenience function for enabling file checkpointing.
func WithFileCheckpointing() Option {
	return WithEnableFileCheckpointing(true)
}

// NewPermissionResultAllow creates an Allow result with proper defaults.
// Use this to permit tool execution.
//
// Example:
//
//	return claudecode.NewPermissionResultAllow(), nil
var NewPermissionResultAllow = control.NewPermissionResultAllow

// NewPermissionResultDeny creates a Deny result with proper defaults.
// Use this to deny tool execution with a reason message.
//
// Example:
//
//	return claudecode.NewPermissionResultDeny("Only Read tool is allowed"), nil
var NewPermissionResultDeny = control.NewPermissionResultDeny

// WithCanUseTool sets the permission callback for tool usage requests.
// The callback is invoked when Claude CLI requests permission to use a tool.
// It receives the tool name, input parameters, and context for decision-making.
//
// Example - Allow all Read tool calls, deny others:
//
//	client := claudecode.NewClient(
//	    claudecode.WithCanUseTool(func(
//	        ctx context.Context,
//	        toolName string,
//	        input map[string]any,
//	        permCtx claudecode.ToolPermissionContext,
//	    ) (claudecode.PermissionResult, error) {
//	        if toolName == "Read" {
//	            return claudecode.NewPermissionResultAllow(), nil
//	        }
//	        return claudecode.NewPermissionResultDeny("Only Read tool is allowed"), nil
//	    }),
//	)
//
// The callback must be thread-safe as it may be invoked concurrently.
// If no callback is set, all tool requests are denied (secure default).
func WithCanUseTool(callback CanUseToolCallback) Option {
	return func(o *Options) {
		// Handle nil callback explicitly
		if callback == nil {
			o.CanUseTool = nil
			return
		}
		// Store a wrapper that converts between control types and any types
		// to bridge the type boundary between shared.Options and control package
		o.CanUseTool = func(
			ctx context.Context,
			toolName string,
			input map[string]any,
			permCtx any,
		) (any, error) {
			// Convert permCtx back to strongly-typed ToolPermissionContext
			tpc, ok := permCtx.(control.ToolPermissionContext)
			if !ok {
				tpc = control.ToolPermissionContext{}
			}
			return callback(ctx, toolName, input, tpc)
		}
	}
}

// HookEvent represents lifecycle events that can trigger hooks.
type HookEvent = control.HookEvent

// Hook event constants.
const (
	// HookEventPreToolUse is triggered before a tool is executed.
	HookEventPreToolUse = control.HookEventPreToolUse
	// HookEventPostToolUse is triggered after a tool is executed.
	HookEventPostToolUse = control.HookEventPostToolUse
	// HookEventPostToolUseFailure is triggered after a tool execution fails.
	HookEventPostToolUseFailure = control.HookEventPostToolUseFailure
	// HookEventUserPromptSubmit is triggered when a user submits a prompt.
	HookEventUserPromptSubmit = control.HookEventUserPromptSubmit
	// HookEventStop is triggered when the session is stopping.
	HookEventStop = control.HookEventStop
	// HookEventSubagentStop is triggered when a subagent is stopping.
	HookEventSubagentStop = control.HookEventSubagentStop
	// HookEventPreCompact is triggered before context compaction.
	HookEventPreCompact = control.HookEventPreCompact
	// HookEventNotification is triggered when the CLI emits a notification.
	HookEventNotification = control.HookEventNotification
	// HookEventSubagentStart is triggered when a subagent starts.
	HookEventSubagentStart = control.HookEventSubagentStart
	// HookEventPermissionRequest is triggered when a permission is requested.
	HookEventPermissionRequest = control.HookEventPermissionRequest
)

// HookCallback is the function signature for hook callbacks.
type HookCallback = control.HookCallback

// HookMatcher defines which hooks to trigger for a given pattern.
type HookMatcher = control.HookMatcher

// HookContext provides context information for hook callbacks.
type HookContext = control.HookContext

// HookJSONOutput is the synchronous hook output structure.
type HookJSONOutput = control.HookJSONOutput

// AsyncHookJSONOutput indicates the hook will respond asynchronously.
type AsyncHookJSONOutput = control.AsyncHookJSONOutput

// BaseHookInput and related types represent hook event inputs.
type (
	// BaseHookInput contains common fields present across all hook events.
	BaseHookInput = control.BaseHookInput
	// PreToolUseHookInput is the input for PreToolUse hook events.
	PreToolUseHookInput = control.PreToolUseHookInput
	// PostToolUseHookInput is the input for PostToolUse hook events.
	PostToolUseHookInput = control.PostToolUseHookInput
	// PostToolUseFailureHookInput is the input for PostToolUseFailure hook events.
	PostToolUseFailureHookInput = control.PostToolUseFailureHookInput
	// UserPromptSubmitHookInput is the input for UserPromptSubmit hook events.
	UserPromptSubmitHookInput = control.UserPromptSubmitHookInput
	// StopHookInput is the input for Stop hook events.
	StopHookInput = control.StopHookInput
	// SubagentStopHookInput is the input for SubagentStop hook events.
	SubagentStopHookInput = control.SubagentStopHookInput
	// PreCompactHookInput is the input for PreCompact hook events.
	PreCompactHookInput = control.PreCompactHookInput
	// NotificationHookInput is the input for Notification hook events.
	NotificationHookInput = control.NotificationHookInput
	// SubagentStartHookInput is the input for SubagentStart hook events.
	SubagentStartHookInput = control.SubagentStartHookInput
	// PermissionRequestHookInput is the input for PermissionRequest hook events.
	PermissionRequestHookInput = control.PermissionRequestHookInput
)

// PreToolUseHookSpecificOutput and related types contain hook-specific output fields.
type (
	// PreToolUseHookSpecificOutput contains PreToolUse-specific output fields.
	PreToolUseHookSpecificOutput = control.PreToolUseHookSpecificOutput
	// PostToolUseHookSpecificOutput contains PostToolUse-specific output fields.
	PostToolUseHookSpecificOutput = control.PostToolUseHookSpecificOutput
	// PostToolUseFailureHookSpecificOutput contains PostToolUseFailure-specific output fields.
	PostToolUseFailureHookSpecificOutput = control.PostToolUseFailureHookSpecificOutput
	// UserPromptSubmitHookSpecificOutput contains UserPromptSubmit-specific output fields.
	UserPromptSubmitHookSpecificOutput = control.UserPromptSubmitHookSpecificOutput
	// NotificationHookSpecificOutput contains Notification-specific output fields.
	NotificationHookSpecificOutput = control.NotificationHookSpecificOutput
	// SubagentStartHookSpecificOutput contains SubagentStart-specific output fields.
	SubagentStartHookSpecificOutput = control.SubagentStartHookSpecificOutput
	// PermissionRequestHookSpecificOutput contains PermissionRequest-specific output fields.
	PermissionRequestHookSpecificOutput = control.PermissionRequestHookSpecificOutput
)

// WithHooks sets the complete hook configuration for lifecycle events.
// This replaces any previously configured hooks.
//
// Example - Configure multiple hooks:
//
//	client := claudecode.NewClient(
//	    claudecode.WithHooks(map[claudecode.HookEvent][]claudecode.HookMatcher{
//	        claudecode.HookEventPreToolUse: {
//	            {Matcher: "Bash", Hooks: []claudecode.HookCallback{myCallback}},
//	        },
//	    }),
//	)
func WithHooks(hooks map[HookEvent][]HookMatcher) Option {
	return func(o *Options) {
		o.Hooks = hooks
	}
}

// WithHook adds a single hook callback for a specific event and tool pattern.
// Multiple calls accumulate hooks for the same event.
// Pass empty string for matcher to match all tools.
//
// Example - Add a PreToolUse hook for Bash commands:
//
//	client := claudecode.NewClient(
//	    claudecode.WithHook(claudecode.HookEventPreToolUse, "Bash", myCallback),
//	)
func WithHook(event HookEvent, matcher string, callback HookCallback) Option {
	return func(o *Options) {
		if o.Hooks == nil {
			o.Hooks = make(map[HookEvent][]HookMatcher)
		}
		hooks, ok := o.Hooks.(map[HookEvent][]HookMatcher)
		if !ok {
			// If not the expected type, initialize fresh
			hooks = make(map[HookEvent][]HookMatcher)
			o.Hooks = hooks
		}
		hooks[event] = append(hooks[event], HookMatcher{
			Matcher: matcher,
			Hooks:   []HookCallback{callback},
		})
	}
}

// WithPreToolUseHook is a convenience function to add a PreToolUse hook.
// Pass empty string for matcher to match all tools.
//
// Example:
//
//	client := claudecode.NewClient(
//	    claudecode.WithPreToolUseHook("Bash", func(ctx context.Context, input any, toolUseID *string, hookCtx claudecode.HookContext) (claudecode.HookJSONOutput, error) {
//	        // Log the bash command
//	        return claudecode.HookJSONOutput{}, nil
//	    }),
//	)
func WithPreToolUseHook(matcher string, callback HookCallback) Option {
	return WithHook(HookEventPreToolUse, matcher, callback)
}

// WithPostToolUseHook is a convenience function to add a PostToolUse hook.
// Pass empty string for matcher to match all tools.
func WithPostToolUseHook(matcher string, callback HookCallback) Option {
	return WithHook(HookEventPostToolUse, matcher, callback)
}
