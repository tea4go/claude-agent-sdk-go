package claudecode

import (
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

// SDKError represents the base interface for all SDK errors.
//
// SDKError 是 SDK 所有错误的基础接口。
type SDKError = shared.SDKError

// BaseError provides common error functionality across the SDK.
//
// BaseError 提供贯穿整个 SDK 的通用错误功能。
type BaseError = shared.BaseError

// ConnectionError represents errors that occur during CLI connection.
//
// ConnectionError 表示连接 CLI 过程中发生的错误。
type ConnectionError = shared.ConnectionError

// CLINotFoundError indicates that the Claude Code CLI was not found.
//
// CLINotFoundError 表示未找到 Claude Code CLI。
type CLINotFoundError = shared.CLINotFoundError

// ProcessError represents errors from the CLI process execution.
//
// ProcessError 表示 CLI 进程执行过程中产生的错误。
type ProcessError = shared.ProcessError

// JSONDecodeError represents JSON parsing errors from CLI responses.
//
// JSONDecodeError 表示解析 CLI 响应 JSON 时的错误。
type JSONDecodeError = shared.JSONDecodeError

// MessageParseError represents errors parsing message content.
//
// MessageParseError 表示解析消息内容时的错误。
type MessageParseError = shared.MessageParseError

// NewConnectionError creates a new connection error.
//
// NewConnectionError 创建一个新的连接错误。
var NewConnectionError = shared.NewConnectionError

// NewCLINotFoundError creates a new CLI not found error.
//
// NewCLINotFoundError 创建一个新的 CLI 未找到错误。
var NewCLINotFoundError = shared.NewCLINotFoundError

// NewProcessError creates a new process error.
//
// NewProcessError 创建一个新的进程错误。
var NewProcessError = shared.NewProcessError

// NewJSONDecodeError creates a new JSON decode error.
//
// NewJSONDecodeError 创建一个新的 JSON 解码错误。
var NewJSONDecodeError = shared.NewJSONDecodeError

// NewMessageParseError creates a new message parse error.
//
// NewMessageParseError 创建一个新的消息解析错误。
var NewMessageParseError = shared.NewMessageParseError

// Error type checking helpers (Go-specific, follows os.IsNotExist pattern).
// These use errors.As() internally to handle wrapped errors correctly.
//
// 错误类型判断辅助函数（Go 特有，仿照 os.IsNotExist 模式）；
// 内部使用 errors.As() 以正确处理被包裹的错误。

// IsConnectionError reports whether err is or wraps a ConnectionError.
//
// IsConnectionError 报告 err 是否为（或包裹）ConnectionError。
var IsConnectionError = shared.IsConnectionError

// IsCLINotFoundError reports whether err is or wraps a CLINotFoundError.
//
// IsCLINotFoundError 报告 err 是否为（或包裹）CLINotFoundError。
var IsCLINotFoundError = shared.IsCLINotFoundError

// IsProcessError reports whether err is or wraps a ProcessError.
//
// IsProcessError 报告 err 是否为（或包裹）ProcessError。
var IsProcessError = shared.IsProcessError

// IsJSONDecodeError reports whether err is or wraps a JSONDecodeError.
//
// IsJSONDecodeError 报告 err 是否为（或包裹）JSONDecodeError。
var IsJSONDecodeError = shared.IsJSONDecodeError

// IsMessageParseError reports whether err is or wraps a MessageParseError.
//
// IsMessageParseError 报告 err 是否为（或包裹）MessageParseError。
var IsMessageParseError = shared.IsMessageParseError

// Error type extraction helpers (Go-specific).
// Returns typed pointer for field access, or nil if not matching type.
//
// 错误类型提取辅助函数（Go 特有）；
// 返回带类型的指针以便访问字段，类型不匹配时返回 nil。

// AsConnectionError returns the error as a *ConnectionError if it is one,
// or nil otherwise.
//
// AsConnectionError 当 err 为 *ConnectionError 时返回它，否则返回 nil。
var AsConnectionError = shared.AsConnectionError

// AsCLINotFoundError returns the error as a *CLINotFoundError if it is one,
// or nil otherwise.
//
// AsCLINotFoundError 当 err 为 *CLINotFoundError 时返回它，否则返回 nil。
var AsCLINotFoundError = shared.AsCLINotFoundError

// AsProcessError returns the error as a *ProcessError if it is one,
// or nil otherwise.
//
// AsProcessError 当 err 为 *ProcessError 时返回它，否则返回 nil。
var AsProcessError = shared.AsProcessError

// AsJSONDecodeError returns the error as a *JSONDecodeError if it is one,
// or nil otherwise.
//
// AsJSONDecodeError 当 err 为 *JSONDecodeError 时返回它，否则返回 nil。
var AsJSONDecodeError = shared.AsJSONDecodeError

// AsMessageParseError returns the error as a *MessageParseError if it is one,
// or nil otherwise.
//
// AsMessageParseError 当 err 为 *MessageParseError 时返回它，否则返回 nil。
var AsMessageParseError = shared.AsMessageParseError
