package claudecode

import (
	"github.com/tea4go/claude-agent-sdk-go/internal/session"
)

// SDKSessionInfo holds metadata about a session.
//
// SDKSessionInfo 保存一个会话的元数据。
type SDKSessionInfo = session.SDKSessionInfo

// SessionMessage represents a message from a session transcript.
//
// SessionMessage 表示会话记录中的一条消息。
type SessionMessage = session.Message

// SessionContentBlock represents a typed content block from a session message.
//
// SessionContentBlock 表示会话消息中一个带类型的内容块。
type SessionContentBlock = session.ContentBlock

// SessionMessageContent is a sum type representing the content of a session message.
//
// SessionMessageContent 是表示会话消息内容的和类型（sum type）。
type SessionMessageContent = session.MessageContent

// SessionContentType discriminates the SessionMessageContent union.
//
// SessionContentType 用于判别 SessionMessageContent 联合体的实际类型。
type SessionContentType = session.ContentType

// SessionContentType constants.
//
// SessionContentType 常量。
const (
	SessionContentTypeString = session.ContentTypeString
	SessionContentTypeBlocks = session.ContentTypeBlocks
)

// Session content block type constants.
//
// 会话内容块类型常量。
const (
	SessionBlockTypeText                         = session.BlockTypeText
	SessionBlockTypeThinking                     = session.BlockTypeThinking
	SessionBlockTypeRedactedThinking             = session.BlockTypeRedactedThinking
	SessionBlockTypeToolUse                      = session.BlockTypeToolUse
	SessionBlockTypeServerToolUse                = session.BlockTypeServerToolUse
	SessionBlockTypeToolResult                   = session.BlockTypeToolResult
	SessionBlockTypeImage                        = session.BlockTypeImage
	SessionBlockTypeWebSearchToolResult          = session.BlockTypeWebSearchToolResult
	SessionBlockTypeWebFetchToolResult           = session.BlockTypeWebFetchToolResult
	SessionBlockTypeCodeExecutionToolResult      = session.BlockTypeCodeExecutionToolResult
	SessionBlockTypeBashCodeExecutionToolResult  = session.BlockTypeBashCodeExecutionToolResult
	SessionBlockTypeTextEditorCodeExecToolResult = session.BlockTypeTextEditorCodeExecToolResult
	SessionBlockTypeToolSearchToolResult         = session.BlockTypeToolSearchToolResult
	SessionBlockTypeContainerUpload              = session.BlockTypeContainerUpload
)

// SessionOption configures session query behavior.
//
// SessionOption 用于配置会话查询行为。
type SessionOption = session.Option

// WithSessionDirectory scopes the query to a specific project directory.
// When omitted, sessions across all projects are searched.
//
// WithSessionDirectory 将查询限定到特定项目目录；省略时则搜索所有项目的会话。
var WithSessionDirectory = session.WithSessionDirectory

// WithSessionLimit sets the maximum number of results to return.
//
// WithSessionLimit 设置返回结果的最大数量。
var WithSessionLimit = session.WithSessionLimit

// WithSessionOffset skips the first n messages (GetSessionMessages only).
//
// WithSessionOffset 跳过前 n 条消息（仅对 GetSessionMessages 有效）。
var WithSessionOffset = session.WithSessionOffset

// WithIncludeWorktrees controls whether git worktree directories are included
// when searching for sessions. Defaults to true. Only has effect when a
// directory is specified via WithSessionDirectory.
//
// WithIncludeWorktrees 控制搜索会话时是否包含 git worktree 目录，默认为 true。
// 仅当通过 WithSessionDirectory 指定了目录时才生效。
var WithIncludeWorktrees = session.WithIncludeWorktrees

// ListSessions returns metadata for sessions, sorted by LastModified descending.
// Use WithSessionDirectory to scope to a specific project, or omit to list all.
//
// ListSessions 返回会话元数据，按 LastModified 降序排列。
// 使用 WithSessionDirectory 限定到特定项目，省略则列出全部。
//
// Example:
//
//	// List 10 most recent sessions in a project
//	sessions, err := claudecode.ListSessions(
//	    claudecode.WithSessionDirectory("/path/to/project"),
//	    claudecode.WithSessionLimit(10),
//	)
//
//	// List all sessions across all projects
//	sessions, err := claudecode.ListSessions()
func ListSessions(opts ...SessionOption) ([]SDKSessionInfo, error) {
	return session.ListSessions(opts...)
}

// GetSessionMessages reads user and assistant messages from a session transcript.
//
// GetSessionMessages 从会话记录中读取用户与助手消息。
//
// Example:
//
//	messages, err := claudecode.GetSessionMessages(sessionID,
//	    claudecode.WithSessionDirectory("/path/to/project"),
//	    claudecode.WithSessionLimit(20),
//	)
func GetSessionMessages(sessionID string, opts ...SessionOption) ([]SessionMessage, error) {
	return session.GetMessages(sessionID, opts...)
}

// GetSessionInfo returns metadata for a single session by ID.
// Returns nil (not an error) if the session is not found.
//
// GetSessionInfo 按 ID 返回单个会话的元数据；会话不存在时返回 nil（而非错误）。
//
// Example:
//
//	info, err := claudecode.GetSessionInfo(sessionID)
//	if info != nil {
//	    fmt.Println(info.Summary)
//	}
func GetSessionInfo(sessionID string, opts ...SessionOption) (*SDKSessionInfo, error) {
	return session.GetSessionInfo(sessionID, opts...)
}
