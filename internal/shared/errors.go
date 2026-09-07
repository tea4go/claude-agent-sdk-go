// Package shared provides shared types and interfaces used across internal packages.
//
// shared 包提供在各 internal 子包之间共享的类型与接口（消息、选项、
// 错误、流校验等），是 SDK 内部各模块的公共地基。
package shared

import (
	"errors"
	"fmt"
)

// SDKError is the base interface for all Claude Agent SDK errors.
//
// SDKError 是所有 Claude Agent SDK 错误的基础接口：除 error 外额外提供
// Type() 用于区分错误类别，便于调用方统一处理。
type SDKError interface {
	error
	Type() string
}

// BaseError provides common error functionality.
//
// BaseError 提供通用的错误能力（消息与包裹的底层错误），供具体错误类型嵌入复用。
type BaseError struct {
	message string // 错误描述
	cause   error  // 被包裹的底层错误（可为 nil）
}

// Error 实现 error 接口：若存在底层错误则一并拼接。
func (e *BaseError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

// Unwrap 返回被包裹的底层错误，支持 errors.Is/errors.As 链式解包。
func (e *BaseError) Unwrap() error {
	return e.cause
}

// Type returns the error type for BaseError.
//
// Type 返回 BaseError 的错误类型标识。
func (e *BaseError) Type() string {
	return "base_error"
}

// ConnectionError represents connection-related failures.
//
// ConnectionError 表示与 CLI 建立/保持连接相关的失败。
type ConnectionError struct {
	BaseError
}

// Type returns the error type for ConnectionError.
//
// Type 返回 ConnectionError 的错误类型标识。
func (e *ConnectionError) Type() string {
	return "connection_error"
}

// NewConnectionError creates a new ConnectionError.
//
// NewConnectionError 创建一个新的 ConnectionError，cause 可为 nil。
func NewConnectionError(message string, cause error) *ConnectionError {
	return &ConnectionError{
		BaseError: BaseError{message: message, cause: cause},
	}
}

// IsConnectionError reports whether err is or wraps a ConnectionError.
//
// IsConnectionError 报告 err 本身或其包裹的错误是否为 ConnectionError。
func IsConnectionError(err error) bool {
	var target *ConnectionError
	return errors.As(err, &target)
}

// AsConnectionError returns the error as a *ConnectionError if it is one,
// or nil otherwise. This allows convenient field access after type checking.
//
// AsConnectionError 尝试将 err 转为 *ConnectionError，成功返回具体错误，
// 否则返回 nil，便于在类型判断后直接访问其字段。
func AsConnectionError(err error) *ConnectionError {
	var target *ConnectionError
	if errors.As(err, &target) {
		return target
	}
	return nil
}

// CLINotFoundError indicates the Claude CLI was not found.
//
// CLINotFoundError 表示未能定位到 Claude CLI 可执行文件；Path 记录尝试的路径。
type CLINotFoundError struct {
	BaseError
	Path string
}

// Type returns the error type for CLINotFoundError.
//
// Type 返回 CLINotFoundError 的错误类型标识。
func (e *CLINotFoundError) Type() string {
	return "cli_not_found_error"
}

// NewCLINotFoundError creates a new CLINotFoundError.
//
// NewCLINotFoundError 创建一个新的 CLINotFoundError；为对齐 Python 行为，
// 若提供了 path 则将消息格式化为 "message: path"。
func NewCLINotFoundError(path, message string) *CLINotFoundError {
	// Match Python behavior: if path provided, format as "message: path"
	if path != "" {
		message = fmt.Sprintf("%s: %s", message, path)
	}
	return &CLINotFoundError{
		BaseError: BaseError{message: message},
		Path:      path,
	}
}

// IsCLINotFoundError reports whether err is or wraps a CLINotFoundError.
//
// IsCLINotFoundError 报告 err 本身或其包裹的错误是否为 CLINotFoundError。
func IsCLINotFoundError(err error) bool {
	var target *CLINotFoundError
	return errors.As(err, &target)
}

// AsCLINotFoundError returns the error as a *CLINotFoundError if it is one,
// or nil otherwise. This allows convenient field access after type checking.
//
// AsCLINotFoundError 尝试将 err 转为 *CLINotFoundError，失败时返回 nil。
func AsCLINotFoundError(err error) *CLINotFoundError {
	var target *CLINotFoundError
	if errors.As(err, &target) {
		return target
	}
	return nil
}

// ProcessError represents subprocess execution failures.
//
// ProcessError 表示子进程执行失败；ExitCode 为退出码，Stderr 为错误输出。
type ProcessError struct {
	BaseError
	ExitCode int
	Stderr   string
}

// Type returns the error type for ProcessError.
//
// Type 返回 ProcessError 的错误类型标识。
func (e *ProcessError) Type() string {
	return "process_error"
}

// Error 将退出码与错误输出（若有）拼接到消息中以便定位问题。
func (e *ProcessError) Error() string {
	message := e.message
	if e.ExitCode != 0 {
		message = fmt.Sprintf("%s (exit code: %d)", message, e.ExitCode)
	}
	if e.Stderr != "" {
		message = fmt.Sprintf("%s\nError output: %s", message, e.Stderr)
	}
	return message
}

// NewProcessError creates a new ProcessError.
//
// NewProcessError 创建一个新的 ProcessError。
func NewProcessError(message string, exitCode int, stderr string) *ProcessError {
	return &ProcessError{
		BaseError: BaseError{message: message},
		ExitCode:  exitCode,
		Stderr:    stderr,
	}
}

// IsProcessError reports whether err is or wraps a ProcessError.
func IsProcessError(err error) bool {
	var target *ProcessError
	return errors.As(err, &target)
}

// AsProcessError returns the error as a *ProcessError if it is one,
// or nil otherwise. This allows convenient field access after type checking.
func AsProcessError(err error) *ProcessError {
	var target *ProcessError
	if errors.As(err, &target) {
		return target
	}
	return nil
}

// JSONDecodeError represents JSON parsing failures.
//
// JSONDecodeError 表示 JSON 解析失败；Line 为原始行，Position 为位置，
// OriginalError 保留底层解析器的原始错误（不拼入消息，与 Python 行为一致）。
type JSONDecodeError struct {
	BaseError
	Line          string
	Position      int
	OriginalError error
}

// Type returns the error type for JSONDecodeError.
//
// Type 返回 JSONDecodeError 的错误类型标识。
func (e *JSONDecodeError) Type() string {
	return "json_decode_error"
}

// maxLineDisplayLength 限定错误消息中展示原始行的最大长度。
const maxLineDisplayLength = 100

// NewJSONDecodeError creates a new JSONDecodeError.
//
// NewJSONDecodeError 创建一个新的 JSONDecodeError；为对齐 Python 行为，
// 原始行会被截断到 maxLineDisplayLength 个字符并附加省略号。
func NewJSONDecodeError(line string, position int, cause error) *JSONDecodeError {
	// Match Python behavior: truncate line to maxLineDisplayLength chars and add ...
	truncatedLine := line
	if len(line) > maxLineDisplayLength {
		truncatedLine = line[:maxLineDisplayLength]
	}
	message := fmt.Sprintf("Failed to decode JSON: %s...", truncatedLine)

	return &JSONDecodeError{
		BaseError:     BaseError{message: message}, // 不将 cause 拼入消息
		Line:          line,
		Position:      position,
		OriginalError: cause, // 像 Python 那样单独保存
	}
}

// Unwrap 返回底层解析错误，支持 errors.Is/errors.As 解包。
func (e *JSONDecodeError) Unwrap() error {
	return e.OriginalError
}

// IsJSONDecodeError reports whether err is or wraps a JSONDecodeError.
func IsJSONDecodeError(err error) bool {
	var target *JSONDecodeError
	return errors.As(err, &target)
}

// AsJSONDecodeError returns the error as a *JSONDecodeError if it is one,
// or nil otherwise. This allows convenient field access after type checking.
func AsJSONDecodeError(err error) *JSONDecodeError {
	var target *JSONDecodeError
	if errors.As(err, &target) {
		return target
	}
	return nil
}

// MessageParseError represents message structure parsing failures.
//
// MessageParseError 表示消息结构解析失败；Data 保留导致解析失败的原始数据，便于排查。
type MessageParseError struct {
	BaseError
	Data any
}

// Type returns the error type for MessageParseError.
//
// Type 返回 MessageParseError 的错误类型标识。
func (e *MessageParseError) Type() string {
	return "message_parse_error"
}

// NewMessageParseError creates a new MessageParseError.
//
// NewMessageParseError 创建一个新的 MessageParseError，并保存原始数据。
func NewMessageParseError(message string, data any) *MessageParseError {
	return &MessageParseError{
		BaseError: BaseError{message: message},
		Data:      data,
	}
}

// IsMessageParseError reports whether err is or wraps a MessageParseError.
func IsMessageParseError(err error) bool {
	var target *MessageParseError
	return errors.As(err, &target)
}

// AsMessageParseError returns the error as a *MessageParseError if it is one,
// or nil otherwise. This allows convenient field access after type checking.
func AsMessageParseError(err error) *MessageParseError {
	var target *MessageParseError
	if errors.As(err, &target) {
		return target
	}
	return nil
}
