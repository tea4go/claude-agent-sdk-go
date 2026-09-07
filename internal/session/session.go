// Package session provides functions for reading Claude Code session data from disk.
//
// Sessions are stored as JSONL files at ~/.claude/projects/<encoded-cwd>/<session-id>.jsonl.
// The encoded-cwd replaces every non-alphanumeric character with "-".
//
// session 包提供从磁盘读取 Claude Code 会话数据的函数。
// 会话以 JSONL 文件形式存于 ~/.claude/projects/<编码后的工作目录>/<会话ID>.jsonl；
// 编码后的工作目录将每个非字母数字字符替换为 "-"。
package session

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tea4go/claude-agent-sdk-go/internal/cli"
)

// errSessionNotFound is a sentinel error returned by findSessionFile when
// the session JSONL file does not exist in any project directory.
//
// errSessionNotFound 是 findSessionFile 在任何项目目录中都找不到会话 JSONL 文件时
// 返回的哨兵错误。
var errSessionNotFound = errors.New("session not found")

// SDKSessionInfo holds metadata about a session.
//
// SDKSessionInfo 保存一个会话的元数据。
type SDKSessionInfo struct {
	SessionID    string  `json:"session_id"`
	Summary      string  `json:"summary"`
	LastModified int64   `json:"last_modified"`
	FileSize     *int64  `json:"file_size,omitempty"`
	CustomTitle  *string `json:"custom_title,omitempty"`
	AITitle      *string `json:"ai_title,omitempty"`
	FirstPrompt  *string `json:"first_prompt,omitempty"`
	GitBranch    *string `json:"git_branch,omitempty"`
	Cwd          *string `json:"cwd,omitempty"`
	Tag          *string `json:"tag,omitempty"`
	CreatedAt    *int64  `json:"created_at,omitempty"`
}

// ContentType discriminates the MessageContent union.
//
// ContentType 用于判别 MessageContent 联合体的实际类型。
type ContentType int

const (
	// ContentTypeString indicates the message content is a plain string.
	// ContentTypeString 表示消息内容为普通字符串。
	ContentTypeString ContentType = iota + 1
	// ContentTypeBlocks indicates the message content is an array of content blocks.
	// ContentTypeBlocks 表示消息内容为内容块数组。
	ContentTypeBlocks
)

// MessageContent is a sum type representing the content of a session message.
// Kind indicates which field is populated.
//
// MessageContent 是表示会话消息内容的和类型（sum type），Kind 指示哪个字段被填充。
type MessageContent struct {
	Kind   ContentType
	String string         // populated when Kind == ContentTypeString
	Blocks []ContentBlock // populated when Kind == ContentTypeBlocks
}

// Block type constants for ContentBlock.Type.
//
// ContentBlock.Type 的内容块类型常量。
const (
	BlockTypeText                         = "text"
	BlockTypeThinking                     = "thinking"
	BlockTypeRedactedThinking             = "redacted_thinking"
	BlockTypeToolUse                      = "tool_use"
	BlockTypeServerToolUse                = "server_tool_use"
	BlockTypeToolResult                   = "tool_result"
	BlockTypeImage                        = "image"
	BlockTypeWebSearchToolResult          = "web_search_tool_result"
	BlockTypeWebFetchToolResult           = "web_fetch_tool_result"
	BlockTypeCodeExecutionToolResult      = "code_execution_tool_result"
	BlockTypeBashCodeExecutionToolResult  = "bash_code_execution_tool_result"
	BlockTypeTextEditorCodeExecToolResult = "text_editor_code_execution_tool_result"
	BlockTypeToolSearchToolResult         = "tool_search_tool_result"
	BlockTypeContainerUpload              = "container_upload"
)

// ContentBlock represents a typed content block from a session message.
// The Type field discriminates the variant. Unknown types are preserved in Raw.
//
// ContentBlock 表示会话消息中一个带类型的内容块。
// Type 字段用于判别变体；未知类型保留在 Raw 中。
type ContentBlock struct {
	// Type discriminates the block variant.
	// Use the BlockType* constants to compare against known types.
	Type string `json:"type"`

	// Raw holds the full original map for all block types (always populated).
	Raw map[string]any `json:"-"`

	// text
	Text string `json:"text,omitempty"`

	// thinking
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`

	// redacted_thinking
	Data string `json:"data,omitempty"`

	// tool_use, server_tool_use
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`

	// tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   any    `json:"content,omitempty"` // string or nested blocks
	IsError   *bool  `json:"is_error,omitempty"`

	// image
	Source map[string]any `json:"source,omitempty"`
}

// Message represents a message from a session transcript.
//
// Message 表示会话记录（transcript）中的一条消息。
type Message struct {
	Type      string `json:"type"` // "user", "assistant", etc. (there are many other types beyond just these two)
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
	// *should* be true for system-injected messages,
	// but nothing in claude api enforces it,
	// so some implementations like claude-vscode inject messages without this.
	IsMeta          bool            `json:"is_meta"`
	RawMessage      map[string]any  `json:"message"`                      // raw message data
	Content         *MessageContent `json:"-"`                            // parsed content
	ParentToolUseID *string         `json:"parent_tool_use_id,omitempty"` // reserved
}

// Option configures session query behavior.
//
// Option 用于配置会话查询行为。
type Option func(*sessionOpts)

type sessionOpts struct {
	directory        string
	limit            int
	offset           int
	includeWorktrees *bool // nil means default (true)
}

func defaultOpts() sessionOpts {
	return sessionOpts{}
}

// includeWorktreesEnabled returns whether worktree scanning is enabled.
// Defaults to true when not explicitly set.
//
// includeWorktreesEnabled 返回是否启用 worktree 扫描；未显式设置时默认为 true。
func (o sessionOpts) includeWorktreesEnabled() bool {
	if o.includeWorktrees == nil {
		return true
	}
	return *o.includeWorktrees
}

// WithSessionDirectory scopes the query to a specific project directory.
// When omitted, sessions across all projects are searched.
//
// WithSessionDirectory 将查询限定到特定项目目录；省略时则搜索所有项目的会话。
func WithSessionDirectory(dir string) Option {
	return func(o *sessionOpts) {
		o.directory = dir
	}
}

// WithSessionLimit sets the maximum number of results to return.
//
// WithSessionLimit 设置返回结果的最大数量。
func WithSessionLimit(n int) Option {
	return func(o *sessionOpts) {
		o.limit = n
	}
}

// WithSessionOffset skips the first n messages (GetMessages only).
//
// WithSessionOffset 跳过前 n 条消息（仅对 GetMessages 有效）。
func WithSessionOffset(n int) Option {
	return func(o *sessionOpts) {
		o.offset = n
	}
}

// WithIncludeWorktrees controls whether git worktree directories are included
// when searching for sessions. Defaults to true. Only has effect when a
// directory is specified via WithSessionDirectory.
//
// WithIncludeWorktrees 控制搜索会话时是否包含 git worktree 目录，默认为 true。
// 仅当通过 WithSessionDirectory 指定了目录时才生效。
func WithIncludeWorktrees(include bool) Option {
	return func(o *sessionOpts) {
		o.includeWorktrees = &include
	}
}

// ListSessions returns metadata for sessions, sorted by LastModified descending.
// Unreadable project directories and individual session files are silently skipped
// to provide best-effort results (matches the Python SDK's behavior).
//
// ListSessions 返回会话元数据，按 LastModified 降序排列。
// 不可读的项目目录与单个会话文件会被静默跳过，以提供尽力而为的结果（与 Python SDK 一致）。
func ListSessions(opts ...Option) ([]SDKSessionInfo, error) {
	o := defaultOpts()
	for _, fn := range opts {
		fn(&o)
	}

	dirs, err := projectDirsForOpts(o)
	if err != nil {
		return nil, err
	}

	var sessions []SDKSessionInfo
	for _, dir := range dirs {
		infos, err := listSessionsInDir(dir)
		if err != nil {
			continue // skip unreadable directories
		}
		sessions = append(sessions, infos...)
	}

	// Deduplicate sessions that appear in multiple worktree project dirs.
	sessions = deduplicateBySessionID(sessions)

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastModified > sessions[j].LastModified
	})

	if o.limit > 0 && len(sessions) > o.limit {
		sessions = sessions[:o.limit]
	}

	return sessions, nil
}

// GetMessages reads user and assistant messages from a session transcript.
//
// GetMessages 从会话记录中读取用户与助手消息。
func GetMessages(sessionID string, opts ...Option) ([]Message, error) {
	o := defaultOpts()
	for _, fn := range opts {
		fn(&o)
	}

	path, err := findSessionFile(sessionID, o)
	if err != nil {
		return nil, err
	}

	entries, err := parseJSONLFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading session %s: %w", sessionID, err)
	}

	messages := buildMessages(sessionID, entries)

	if o.offset > 0 {
		if o.offset >= len(messages) {
			return nil, nil
		}
		messages = messages[o.offset:]
	}

	if o.limit > 0 && len(messages) > o.limit {
		messages = messages[:o.limit]
	}

	return messages, nil
}

// GetSessionInfo returns metadata for a single session by ID.
// Returns nil (not an error) if the session is not found.
//
// GetSessionInfo 按 ID 返回单个会话的元数据；会话不存在时返回 nil（而非错误）。
func GetSessionInfo(sessionID string, opts ...Option) (*SDKSessionInfo, error) {
	o := defaultOpts()
	for _, fn := range opts {
		fn(&o)
	}

	path, err := findSessionFile(sessionID, o)
	if errors.Is(err, errSessionNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return buildSessionInfoFromFile(sessionID, path)
}

// configDir returns the Claude configuration directory.
//
// configDir 返回 Claude 配置目录（优先使用 CLAUDE_CONFIG_DIR，否则为 ~/.claude）。
func configDir() (string, error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

// worktreeTimeout is the maximum time to wait for `git worktree list --porcelain`.
const worktreeTimeout = 5 * time.Second

// getWorktreePaths runs `git worktree list --porcelain` in the given directory
// and returns all worktree paths. Returns an empty slice (not an error) if git
// is not available, the directory is not a git repo, or the command fails.
// Matches the Python SDK's _get_worktree_paths.
//
// getWorktreePaths 在指定目录运行 `git worktree list --porcelain` 并返回所有 worktree 路径。
// 若 git 不可用、目录非 git 仓库或命令失败，则返回空切片（而非错误）。
func getWorktreePaths(dir string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), worktreeTimeout)
	defer cancel()

	cmd := cli.NewExecCommandContext(ctx, []string{"git", "worktree", "list", "--porcelain"})
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			p := strings.TrimPrefix(line, "worktree ")
			if p != "" {
				paths = append(paths, p)
			}
		}
	}
	return paths
}

// encodeCwd encodes a directory path by replacing non-alphanumeric characters with "-".
//
// encodeCwd 将目录路径中的非字母数字字符替换为 "-" 进行编码。
func encodeCwd(cwd string) string {
	var b strings.Builder
	b.Grow(len(cwd))
	for _, r := range cwd {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// projectDirsForOpts returns the project directories to search based on options.
// When a directory is specified and includeWorktrees is enabled, it also includes
// project directories for all git worktree paths.
//
// projectDirsForOpts 根据选项返回需要搜索的项目目录。
// 当指定了目录且启用 includeWorktrees 时，还会包含所有 git worktree 路径对应的项目目录。
func projectDirsForOpts(o sessionOpts) ([]string, error) {
	cfgDir, err := configDir()
	if err != nil {
		return nil, err
	}
	projectsDir := filepath.Join(cfgDir, "projects")

	if o.directory != "" {
		abs, err := filepath.Abs(o.directory)
		if err != nil {
			return nil, fmt.Errorf("resolving directory: %w", err)
		}

		// Collect all candidate directories: user's dir first, then worktrees.
		candidatePaths := []string{abs}
		if o.includeWorktreesEnabled() {
			candidatePaths = append(candidatePaths, getWorktreePaths(abs)...)
		}

		// Encode each candidate path and collect existing project dirs.
		seen := make(map[string]bool)
		var dirs []string
		for _, p := range candidatePaths {
			encoded := encodeCwd(p)
			dir := filepath.Join(projectsDir, encoded)
			if seen[dir] {
				continue
			}
			seen[dir] = true
			if _, err := os.Stat(dir); err == nil {
				dirs = append(dirs, dir)
			}
		}
		if len(dirs) == 0 {
			return nil, nil
		}
		return dirs, nil
	}

	// List all project directories.
	names, err := readDirNames(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading projects directory: %w", err)
	}

	var dirs []string
	for _, name := range names {
		full := filepath.Join(projectsDir, name)
		fi, err := os.Stat(full)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			dirs = append(dirs, full)
		}
	}
	return dirs, nil
}

// readDirNames is a replacement for os.ReadDir which is vulnerable to GO-2026-4602.
// See https://pkg.go.dev/vuln/GO-2026-4602. Remove once upgraded to go1.26.0 or later.
//
// readDirNames 是 os.ReadDir 的替代实现（os.ReadDir 存在 GO-2026-4602 漏洞）。
// 参见 https://pkg.go.dev/vuln/GO-2026-4602；升级到 go1.26.0 及以上后可移除。
func readDirNames(path string) (names []string, err error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing directory: %w", cerr)
		}
	}()
	return f.Readdirnames(-1)
}

// deduplicateBySessionID deduplicates sessions by SessionID, keeping the entry
// with the newest LastModified value. Matches the Python SDK's _deduplicate_by_session_id.
//
// deduplicateBySessionID 按 SessionID 去重，保留 LastModified 最新的条目。
func deduplicateBySessionID(sessions []SDKSessionInfo) []SDKSessionInfo {
	if len(sessions) == 0 {
		return sessions
	}
	best := make(map[string]SDKSessionInfo, len(sessions))
	for _, s := range sessions {
		if existing, ok := best[s.SessionID]; !ok || s.LastModified > existing.LastModified {
			best[s.SessionID] = s
		}
	}
	deduped := make([]SDKSessionInfo, 0, len(best))
	for _, s := range best {
		deduped = append(deduped, s)
	}
	return deduped
}

// listSessionsInDir lists all sessions in a single project directory.
// Individual session files that fail to parse (corrupt JSONL, permission errors)
// are silently skipped to provide best-effort results.
//
// listSessionsInDir 列出单个项目目录中的所有会话。
// 解析失败的单个会话文件（JSONL 损坏、权限错误）会被静默跳过，以提供尽力而为的结果。
func listSessionsInDir(dir string) ([]SDKSessionInfo, error) {
	names, err := readDirNames(dir)
	if err != nil {
		return nil, err
	}

	var sessions []SDKSessionInfo
	for _, name := range names {
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		sessionID := strings.TrimSuffix(name, ".jsonl")
		path := filepath.Join(dir, name)

		info, err := buildSessionInfoFromFile(sessionID, path)
		if err != nil {
			continue // skip unreadable sessions
		}
		sessions = append(sessions, *info)
	}
	return sessions, nil
}

// findSessionFile locates the JSONL file for a session ID.
//
// findSessionFile 定位某个会话 ID 对应的 JSONL 文件。
func findSessionFile(sessionID string, o sessionOpts) (string, error) {
	dirs, err := projectDirsForOpts(o)
	if err != nil {
		return "", err
	}

	filename := sessionID + ".jsonl"
	for _, dir := range dirs {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%w: %s", errSessionNotFound, sessionID)
}

// metadataReadSize is the size of head/tail chunks for metadata extraction.
// Matches the Python SDK's LITE_READ_BUF_SIZE.
const metadataReadSize int64 = 64 * 1024

// buildSessionInfoFromFile builds SDKSessionInfo by reading a JSONL file.
// Uses head/tail reads for efficiency — only reads the first and last 64KB
// of the file rather than parsing the entire JSONL.
//
// buildSessionInfoFromFile 通过读取 JSONL 文件构建 SDKSessionInfo。
// 为提高效率，仅读取文件首尾各 64KB，而非解析整个 JSONL。
func buildSessionInfoFromFile(sessionID, path string) (*SDKSessionInfo, error) {
	path = filepath.Clean(path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	entries, err := parseJSONLHeadTail(path, metadataReadSize)
	if err != nil {
		return nil, err
	}

	return buildSessionInfo(sessionID, entries, fileInfo), nil
}

// jsonlEntry represents a parsed line from a JSONL file.
//
// jsonlEntry 表示从 JSONL 文件解析出的一行。
type jsonlEntry struct {
	entryType string
	raw       map[string]any
}

// parseJSONLFile reads and parses all lines from a JSONL file.
//
// parseJSONLFile 读取并解析 JSONL 文件的所有行。
func parseJSONLFile(path string) (entries []jsonlEntry, err error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing session file: %w", cerr)
		}
	}()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw map[string]any
		if uerr := json.Unmarshal(line, &raw); uerr != nil {
			continue // skip malformed lines
		}
		typ, _ := raw["type"].(string)
		entries = append(entries, jsonlEntry{entryType: typ, raw: raw})
	}
	if serr := scanner.Err(); serr != nil {
		return nil, fmt.Errorf("scanning JSONL file: %w", serr)
	}
	return entries, nil
}

// maxFirstPromptLen is the maximum length of the first prompt (truncated with ellipsis).
const maxFirstPromptLen = 200

// skipFirstPromptPattern matches auto-generated or system messages that should
// be skipped when extracting the first meaningful user prompt.
// Matches the Python SDK's _SKIP_FIRST_PROMPT_PATTERN.
// permalink:
// https://github.com/anthropics/claude-agent-sdk-python/blob/acd62ea8f7bba637f604e0bb09c21b86176ccb49/src/claude_agent_sdk/_internal/sessions.py#L40
var skipFirstPromptPattern = regexp.MustCompile(
	`^(?:<local-command-stdout>|<session-start-hook>|<tick>|<goal>|` +
		`\[Request interrupted by user[^\]]*\]|` +
		`\s*<ide_opened_file>[\s\S]*</ide_opened_file>\s*$|` +
		`\s*<ide_selection>[\s\S]*</ide_selection>\s*$)`,
)

// commandNamePattern matches <command-name>...</command-name> tags.
var commandNamePattern = regexp.MustCompile(`<command-name>(.*?)</command-name>`)

// extractFirstPrompt extracts the first meaningful user prompt from JSONL entries.
// Skips tool_result messages, isMeta, isCompactSummary, command-name messages,
// and auto-generated patterns. Truncates to maxFirstPromptLen characters.
//
// extractFirstPrompt 从 JSONL 条目中提取第一条有意义的用户提示。
// 会跳过 tool_result 消息、isMeta、isCompactSummary、command-name 消息及自动生成的模式，
// 并截断到 maxFirstPromptLen 个字符。
func extractFirstPrompt(entries []jsonlEntry) *string {
	for _, e := range entries {
		if e.entryType != entryTypeUser {
			continue
		}
		if meta, ok := e.raw["isMeta"].(bool); ok && meta {
			continue
		}
		if cs, ok := e.raw["isCompactSummary"].(bool); ok && cs {
			continue
		}

		rawMsg, ok := e.raw["message"].(map[string]any)
		if !ok {
			continue
		}

		text := extractTextFromMessage(rawMsg)
		if text == "" {
			continue
		}

		// Skip command-name messages.
		if commandNamePattern.MatchString(text) {
			continue
		}

		// Skip auto-generated patterns.
		if skipFirstPromptPattern.MatchString(text) {
			continue
		}

		// Collapse newlines and trim.
		text = strings.Join(strings.Fields(text), " ")
		if text == "" {
			continue
		}

		if len(text) > maxFirstPromptLen {
			text = text[:maxFirstPromptLen]
		}
		return &text
	}
	return nil
}

// extractTextFromMessage returns the text content of a message.
// For string content, returns it directly. For block content, returns
// the first text block's text. Returns "" if the content is only tool_result blocks.
//
// extractTextFromMessage 返回消息的文本内容。
// 对于字符串内容直接返回；对于块内容返回第一个文本块的文本；
// 若内容仅含 tool_result 块则返回 ""。
func extractTextFromMessage(msg map[string]any) string {
	content, ok := msg["content"]
	if !ok {
		return ""
	}

	// String content.
	if s, ok := content.(string); ok {
		return s
	}

	// Block content — extract first text block, but skip if only tool_result.
	blocks, ok := content.([]any)
	if !ok {
		return ""
	}

	hasNonToolResult := false
	var firstText string
	for _, block := range blocks {
		b, ok := block.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := b["type"].(string)
		if typ != "tool_result" {
			hasNonToolResult = true
		}
		if typ == "text" && firstText == "" {
			if t, ok := b["text"].(string); ok {
				firstText = t
			}
		}
	}

	if !hasNonToolResult {
		return "" // skip tool_result-only messages
	}
	return firstText
}

// parseJSONLHeadTail reads the first and last bufSize bytes of a JSONL file,
// parsing complete lines from each chunk. For files smaller than 2*bufSize,
// the entire file is read (equivalent to parseJSONLFile).
//
// This is used for metadata extraction (ListSessions, GetSessionInfo) where
// reading the full file is unnecessary — session metadata is in the head
// (timestamps, cwd, gitBranch, first_prompt) and tail (titles, tags).
//
// parseJSONLHeadTail 读取 JSONL 文件首尾各 bufSize 字节，并从每个块中解析完整行。
// 对于小于 2*bufSize 的文件，则读取整个文件（等同于 parseJSONLFile）。
//
// 用于元数据提取（ListSessions、GetSessionInfo），无需读取整个文件——
// 会话元数据位于首部（时间戳、cwd、gitBranch、first_prompt）与尾部（标题、标签）。
func parseJSONLHeadTail(path string, bufSize int64) (entries []jsonlEntry, err error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing session file: %w", cerr)
		}
	}()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fi.Size()

	// Small file — read everything.
	if fileSize <= 2*bufSize {
		return parseJSONLFromReader(f)
	}

	// Read head chunk.
	headBuf := make([]byte, bufSize)
	if _, err := io.ReadFull(f, headBuf); err != nil {
		return nil, fmt.Errorf("reading head: %w", err)
	}

	// Parse complete lines from head (discard last partial line).
	entries = parseLinesFromBytes(headBuf, true)

	// Read tail chunk.
	tailOffset := fileSize - bufSize
	if _, err := f.Seek(tailOffset, 0); err != nil {
		return nil, fmt.Errorf("seeking to tail: %w", err)
	}
	tailBuf := make([]byte, bufSize)
	if _, err := io.ReadFull(f, tailBuf); err != nil {
		return nil, fmt.Errorf("reading tail: %w", err)
	}

	// Parse complete lines from tail (discard first partial line).
	tailEntries := parseLinesFromBytes(tailBuf, false)
	entries = append(entries, tailEntries...)

	return entries, nil
}

// parseJSONLFromReader reads all lines from an already-opened file.
//
// parseJSONLFromReader 从已打开的文件中读取所有行。
func parseJSONLFromReader(f *os.File) ([]jsonlEntry, error) {
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var entries []jsonlEntry
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}
		typ, _ := raw["type"].(string)
		entries = append(entries, jsonlEntry{entryType: typ, raw: raw})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning JSONL file: %w", err)
	}
	return entries, nil
}

// parseLinesFromBytes parses complete JSONL lines from a byte buffer.
// If isHead is true, discards the last partial line (tail of head chunk).
// If isHead is false, discards the first partial line (start of tail chunk).
//
// parseLinesFromBytes 从字节缓冲区中解析完整的 JSONL 行。
// isHead 为 true 时丢弃最后一行不完整行（首部块的尾部）；
// isHead 为 false 时丢弃第一行不完整行（尾部块的开头）。
func parseLinesFromBytes(buf []byte, isHead bool) []jsonlEntry {
	var entries []jsonlEntry
	start := 0

	if !isHead {
		// Skip first partial line in tail chunk.
		idx := bytes.IndexByte(buf, '\n')
		if idx < 0 {
			return nil // no complete line
		}
		start = idx + 1
	}

	for start < len(buf) {
		end := start + bytes.IndexByte(buf[start:], '\n')
		if end < start {
			// No newline found — this is the last (potentially partial) line.
			if isHead {
				break // discard partial last line in head
			}
			end = len(buf)
		}

		line := buf[start:end]
		start = end + 1

		if len(line) == 0 {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}
		typ, _ := raw["type"].(string)
		entries = append(entries, jsonlEntry{entryType: typ, raw: raw})
	}
	return entries
}

// extractMetadataFromEntries populates session metadata fields by scanning JSONL entries.
// Extracts titles, tags, cwd, gitBranch, and createdAt timestamp.
//
// extractMetadataFromEntries 通过扫描 JSONL 条目填充会话元数据字段（标题、标签、cwd、gitBranch、createdAt）。
func extractMetadataFromEntries(info *SDKSessionInfo, entries []jsonlEntry) {
	for _, e := range entries {
		extractEntryMetadata(info, e)
	}
	info.FirstPrompt = extractFirstPrompt(entries)
}

// extractEntryMetadata extracts metadata from a single JSONL entry into info.
//
// extractEntryMetadata 从单个 JSONL 条目中提取元数据填入 info。
func extractEntryMetadata(info *SDKSessionInfo, e jsonlEntry) {
	switch e.entryType {
	case "custom-title":
		if title, ok := e.raw["customTitle"].(string); ok && title != "" {
			info.CustomTitle = &title
		}
	case "ai-title":
		if title, ok := e.raw["aiTitle"].(string); ok && title != "" {
			info.AITitle = &title
		}
	case "tag":
		if tag, ok := e.raw["tag"].(string); ok {
			if tag == "" {
				info.Tag = nil // cleared
			} else {
				info.Tag = &tag
			}
		}
	case entryTypeUser:
		// Track cwd and gitBranch (last one wins)
		if branch, ok := e.raw["gitBranch"].(string); ok && branch != "" {
			info.GitBranch = &branch
		}
		if cwd, ok := e.raw["cwd"].(string); ok && cwd != "" {
			info.Cwd = &cwd
		}
	}

	extractCreatedAt(info, e)
}

// extractCreatedAt sets CreatedAt from the first entry that carries a valid
// RFC3339 timestamp. Once set, subsequent calls are no-ops.
//
// extractCreatedAt 从第一个携带有效 RFC3339 时间戳的条目设置 CreatedAt；一旦设置，后续调用为空操作。
func extractCreatedAt(info *SDKSessionInfo, e jsonlEntry) {
	if info.CreatedAt != nil {
		return
	}
	ts, ok := e.raw["timestamp"].(string)
	if !ok || ts == "" {
		return
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return
	}
	ms := t.UnixMilli()
	info.CreatedAt = &ms
}

// determineSummary sets the Summary field based on available metadata.
// Priority: custom_title > ai_title > first_prompt > timestamp fallback > session ID.
//
// determineSummary 根据可用元数据设置 Summary 字段。
// 优先级：custom_title > ai_title > first_prompt > 时间戳回退 > 会话 ID。
func determineSummary(info *SDKSessionInfo) {
	switch {
	case info.CustomTitle != nil:
		info.Summary = *info.CustomTitle
	case info.AITitle != nil:
		info.Summary = *info.AITitle
	case info.FirstPrompt != nil:
		info.Summary = *info.FirstPrompt
	case info.CreatedAt != nil:
		info.Summary = fmt.Sprintf("New session - %s", time.UnixMilli(*info.CreatedAt).UTC().Format(time.RFC3339))
	default:
		info.Summary = info.SessionID
	}
}

// buildSessionInfo constructs SDKSessionInfo from parsed JSONL entries.
//
// buildSessionInfo 从解析后的 JSONL 条目构建 SDKSessionInfo。
func buildSessionInfo(sessionID string, entries []jsonlEntry, fileInfo os.FileInfo) *SDKSessionInfo {
	info := &SDKSessionInfo{
		SessionID:    sessionID,
		LastModified: fileInfo.ModTime().UnixMilli(),
	}

	fileSize := fileInfo.Size()
	info.FileSize = &fileSize

	extractMetadataFromEntries(info, entries)
	determineSummary(info)

	return info
}

// parseMessageContent parses the content field of a message into a typed MessageContent.
//
// parseMessageContent 将消息的 content 字段解析为带类型的 MessageContent。
func parseMessageContent(msg map[string]any) *MessageContent {
	content, ok := msg["content"]
	if !ok {
		return nil
	}

	// String content.
	if s, ok := content.(string); ok {
		return &MessageContent{
			Kind:   ContentTypeString,
			String: s,
		}
	}

	// Content block array.
	blocks, ok := content.([]any)
	if !ok {
		return nil
	}

	parsed := make([]ContentBlock, 0, len(blocks))
	for _, block := range blocks {
		b, ok := block.(map[string]any)
		if !ok {
			continue
		}
		parsed = append(parsed, parseContentBlock(b))
	}

	return &MessageContent{
		Kind:   ContentTypeBlocks,
		Blocks: parsed,
	}
}

// parseContentBlock parses a single content block from a raw map.
// Known fields are extracted into typed struct fields; Raw is always populated.
//
// parseContentBlock 从原始 map 解析单个内容块。
// 已知字段被提取到带类型的结构体字段；Raw 总是被填充。
func parseContentBlock(raw map[string]any) ContentBlock {
	cb := ContentBlock{
		Raw: raw,
	}

	cb.Type, _ = raw["type"].(string)

	switch cb.Type {
	case BlockTypeText:
		cb.Text, _ = raw["text"].(string)

	case BlockTypeThinking:
		cb.Thinking, _ = raw["thinking"].(string)
		cb.Signature, _ = raw["signature"].(string)

	case BlockTypeRedactedThinking:
		cb.Data, _ = raw["data"].(string)

	case BlockTypeToolUse, BlockTypeServerToolUse:
		cb.ID, _ = raw["id"].(string)
		cb.Name, _ = raw["name"].(string)
		if input, ok := raw["input"].(map[string]any); ok {
			cb.Input = input
		}

	case BlockTypeToolResult:
		cb.ToolUseID, _ = raw["tool_use_id"].(string)
		cb.Content = raw["content"]
		if isErr, ok := raw["is_error"].(bool); ok {
			cb.IsError = &isErr
		}

	case BlockTypeImage:
		if source, ok := raw["source"].(map[string]any); ok {
			cb.Source = source
		}
	}

	return cb
}

// Entry type constants for JSONL entries.
//
// JSONL 条目的类型常量。
const (
	entryTypeUser      = "user"
	entryTypeAssistant = "assistant"
)

// transcriptEntryTypes are the JSONL entry types that carry uuid + parentUuid
// chain links, matching the Python SDK's _TRANSCRIPT_ENTRY_TYPES.
//
// transcriptEntryTypes 是携带 uuid + parentUuid 链接的 JSONL 条目类型集合。
var transcriptEntryTypes = map[string]bool{
	entryTypeUser: true, entryTypeAssistant: true, "progress": true, "system": true, "attachment": true,
}

// isTranscriptEntry returns true if the entry is a transcript message type with a uuid.
//
// isTranscriptEntry 在条目为携带 uuid 的记录消息类型时返回 true。
func isTranscriptEntry(e jsonlEntry) bool {
	if !transcriptEntryTypes[e.entryType] {
		return false
	}
	uuid, _ := e.raw["uuid"].(string)
	return uuid != ""
}

// isVisibleMessage returns true if the entry should be included in returned messages.
// Matches the Python SDK's _is_visible_message filter.
//
// isVisibleMessage 在条目应被包含在返回消息中时返回 true（过滤 isMeta/isSidechain/teamName）。
func isVisibleMessage(e jsonlEntry) bool {
	if e.entryType != entryTypeUser && e.entryType != entryTypeAssistant {
		return false
	}
	if meta, ok := e.raw["isMeta"].(bool); ok && meta {
		return false
	}
	if sc, ok := e.raw["isSidechain"].(bool); ok && sc {
		return false
	}
	if tn, ok := e.raw["teamName"].(string); ok && tn != "" {
		return false
	}
	return true
}

// entryUUID returns the uuid of a JSONL entry, or "".
//
// entryUUID 返回 JSONL 条目的 uuid，不存在时返回 ""。
func entryUUID(e jsonlEntry) string {
	uuid, _ := e.raw["uuid"].(string)
	return uuid
}

// entryParentUUID returns the parentUuid of a JSONL entry, or "".
//
// entryParentUUID 返回 JSONL 条目的 parentUuid，不存在时返回 ""。
func entryParentUUID(e jsonlEntry) string {
	parent, _ := e.raw["parentUuid"].(string)
	return parent
}

// leaf represents a user/assistant entry at the end of a conversation branch,
// used during chain reconstruction to pick the best (main) conversation path.
//
// leaf 表示位于对话分支末端的用户/助手条目，
// 在重建链时用于选择最佳（主）对话路径。
type leaf struct {
	idx         int
	isSidechain bool
	isTeamName  bool
	isMeta      bool
}

// findTerminals returns the indices of transcript entries that have no children
// (i.e., no other entry lists their uuid as a parentUuid).
//
// findTerminals 返回没有子节点的记录条目索引（即没有其他条目将其 uuid 作为 parentUuid）。
func findTerminals(transcriptEntries []jsonlEntry) []int {
	childrenOf := make(map[string]bool, len(transcriptEntries))
	for _, e := range transcriptEntries {
		if parent := entryParentUUID(e); parent != "" {
			childrenOf[parent] = true
		}
	}

	var terminals []int
	for i, e := range transcriptEntries {
		if !childrenOf[entryUUID(e)] {
			terminals = append(terminals, i)
		}
	}
	return terminals
}

// findLeaves walks back from each terminal via parentUuid to find the nearest
// user/assistant entry, collecting metadata about sidechain/teamName/isMeta status.
//
// findLeaves 从每个末端节点沿 parentUuid 回溯，找到最近的用户/助手条目，
// 并收集其 sidechain/teamName/isMeta 状态元数据。
func findLeaves(transcriptEntries []jsonlEntry, terminals []int, byUUID map[string]int) []leaf {
	var leaves []leaf
	for _, termIdx := range terminals {
		if l, ok := walkToLeaf(transcriptEntries, termIdx, byUUID); ok {
			leaves = append(leaves, l)
		}
	}
	return leaves
}

// walkToLeaf walks backwards from a terminal entry via parentUuid links until it
// finds a user or assistant entry, returning it as a leaf. Returns ok=false if
// no qualifying entry is found.
//
// walkToLeaf 从末端条目沿 parentUuid 链接向前回溯，直到找到用户或助手条目并作为 leaf 返回；
// 若未找到符合条件的条目则返回 ok=false。
func walkToLeaf(transcriptEntries []jsonlEntry, startIdx int, byUUID map[string]int) (leaf, bool) {
	seen := make(map[string]bool)
	cur := startIdx
	for cur >= 0 {
		uuid := entryUUID(transcriptEntries[cur])
		if seen[uuid] {
			break
		}
		seen[uuid] = true
		e := transcriptEntries[cur]
		if e.entryType == entryTypeUser || e.entryType == entryTypeAssistant {
			sc, _ := e.raw["isSidechain"].(bool)
			tn, _ := e.raw["teamName"].(string)
			meta, _ := e.raw["isMeta"].(bool)
			return leaf{idx: cur, isSidechain: sc, isTeamName: tn != "", isMeta: meta}, true
		}
		parent := entryParentUUID(e)
		if parent == "" {
			break
		}
		parentIdx, ok := byUUID[parent]
		if !ok {
			break
		}
		cur = parentIdx
	}
	return leaf{}, false
}

// pickBestLeaf selects the best leaf from candidates: prefers entries that are
// not sidechain/teamName/isMeta, breaking ties by highest file position (index).
//
// pickBestLeaf 从候选中选择最佳 leaf：优先选非 sidechain/teamName/isMeta 的条目，
// 并列时以文件位置（索引）最大者胜出。
func pickBestLeaf(leaves []leaf) leaf {
	var mainLeaves []leaf
	for _, l := range leaves {
		if !l.isSidechain && !l.isTeamName && !l.isMeta {
			mainLeaves = append(mainLeaves, l)
		}
	}
	candidates := mainLeaves
	if len(candidates) == 0 {
		candidates = leaves
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.idx > best.idx {
			best = c
		}
	}
	return best
}

// walkChainToRoot walks from the entry at startIdx to the root via parentUuid
// links, collecting entries along the way. Returns entries in chronological order
// (root first).
//
// walkChainToRoot 从 startIdx 处的条目沿 parentUuid 链接走向根节点，途中收集条目，
// 并按时间顺序（根节点在前）返回。
func walkChainToRoot(transcriptEntries []jsonlEntry, startIdx int, byUUID map[string]int) []jsonlEntry {
	var chain []jsonlEntry
	seen := make(map[string]bool)
	cur := startIdx
	for cur >= 0 {
		e := transcriptEntries[cur]
		uuid := entryUUID(e)
		if seen[uuid] {
			break
		}
		seen[uuid] = true
		chain = append(chain, e)
		parent := entryParentUUID(e)
		if parent == "" {
			break
		}
		parentIdx, ok := byUUID[parent]
		if !ok {
			break
		}
		cur = parentIdx
	}

	// Reverse to chronological order.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}

// buildConversationChain reconstructs the main conversation chain from transcript
// entries by walking parentUuid links, matching the Python SDK's _build_conversation_chain.
//
// Algorithm:
//  1. Index transcript entries by uuid
//  2. Find terminals (entries with no children)
//  3. From each terminal, walk back to find the nearest user/assistant leaf
//  4. Pick the best leaf: not sidechain/teamName/isMeta, highest file position
//  5. Walk from leaf to root via parentUuid, reverse to chronological order
//
// buildConversationChain 通过走 parentUuid 链接，从记录条目重建主对话链。
// 算法：
//  1. 按 uuid 为记录条目建索引
//  2. 找出末端节点（无子节点的条目）
//  3. 从每个末端节点回溯，找到最近的用户/助手 leaf
//  4. 选择最佳 leaf：非 sidechain/teamName/isMeta、文件位置最靠后
//  5. 从 leaf 沿 parentUuid 走向根节点，再反转为时间顺序
func buildConversationChain(transcriptEntries []jsonlEntry, byUUID map[string]int) []jsonlEntry {
	if len(transcriptEntries) == 0 {
		return nil
	}

	terminals := findTerminals(transcriptEntries)
	leaves := findLeaves(transcriptEntries, terminals, byUUID)
	if len(leaves) == 0 {
		return nil
	}

	best := pickBestLeaf(leaves)
	return walkChainToRoot(transcriptEntries, best.idx, byUUID)
}

// buildMessages extracts user and assistant messages from JSONL entries.
// If entries contain parentUuid fields, it reconstructs the conversation chain.
// Otherwise, it falls back to a flat scan with visibility filtering.
//
// buildMessages 从 JSONL 条目中提取用户与助手消息。
// 若条目包含 parentUuid 字段，则重建对话链；否则回退到带可见性过滤的平铺扫描。
func buildMessages(sessionID string, entries []jsonlEntry) []Message {
	// Check if any entry has parentUuid — determines chain vs flat-scan path.
	hasParentUUID := false
	var transcriptEntries []jsonlEntry
	byUUID := make(map[string]int)

	for _, e := range entries {
		if !isTranscriptEntry(e) {
			continue
		}
		idx := len(transcriptEntries)
		transcriptEntries = append(transcriptEntries, e)
		byUUID[entryUUID(e)] = idx
		if entryParentUUID(e) != "" {
			hasParentUUID = true
		}
	}

	var visible []jsonlEntry
	if hasParentUUID {
		chain := buildConversationChain(transcriptEntries, byUUID)
		for _, e := range chain {
			if isVisibleMessage(e) {
				visible = append(visible, e)
			}
		}
	} else {
		// Flat-scan fallback for sessions without parentUuid.
		for _, e := range entries {
			if isVisibleMessage(e) {
				visible = append(visible, e)
			}
		}
	}

	messages := make([]Message, 0, len(visible))
	for _, e := range visible {
		msg := Message{
			Type:      e.entryType,
			SessionID: sessionID,
		}
		if uuid, ok := e.raw["uuid"].(string); ok {
			msg.UUID = uuid
		}
		if meta, ok := e.raw["isMeta"].(bool); ok && meta {
			msg.IsMeta = true
		}
		if rawMsg, ok := e.raw["message"].(map[string]any); ok {
			msg.RawMessage = rawMsg
			msg.Content = parseMessageContent(rawMsg)
		}

		messages = append(messages, msg)
	}
	return messages
}
