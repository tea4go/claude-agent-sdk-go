package subprocess

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tea4go/claude-agent-sdk-go/internal/control"
	"github.com/tea4go/claude-agent-sdk-go/internal/parser"
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

// handleStdout processes stdout in a separate goroutine
//
// handleStdout 在独立 goroutine 中处理 stdout（JSONL 流），
// 将解析出的消息发送到 msgChan，并将控制消息路由给控制协议。
func (t *Transport) handleStdout() {
	defer t.wg.Done()
	defer close(t.msgChan)
	defer close(t.errChan)
	defer t.validator.MarkStreamEnd() // Mark stream end for validation // 标记流结束以供校验

	// Stdout is JSONL. Use Reader rather than Scanner so oversized messages
	// fail through the SDK buffer limit instead of Scanner's token limit.
	// Stdout 为 JSONL。使用 Reader 而非 Scanner，以便超大消息通过 SDK 缓冲区
	// 限制报错，而不是受限于 Scanner 的 token 上限。
	reader := bufio.NewReader(t.stdout)
	lineLimit := parser.MaxBufferSize
	if t.options != nil && t.options.MaxBufferSize != nil {
		lineLimit = *t.options.MaxBufferSize
	}

	parsedAny := false
	var lineBuilder strings.Builder

	processLine := func(line string) bool {
		select {
		case <-t.ctx.Done():
			return false
		default:
		}

		if line == "" {
			return true
		}

		// Parse line with the parser
		messages, err := t.parser.ProcessLine(line)
		if err != nil {
			select {
			case t.errChan <- err:
			case <-t.ctx.Done():
				return false
			}
			return true
		}

		// Send parsed messages and track for validation
		for _, msg := range messages {
			if msg == nil {
				continue
			}

			// If this is an error ResultMessage before we're fully connected,
			// it means the CLI failed during init (e.g., invalid session ID).
			// Route the error to the control protocol to unblock Initialize().
			// 若在尚未完全连接前就收到错误 ResultMessage，说明 CLI 在初始化阶段
			// 失败（如无效会话 ID），将错误路由给控制协议以解除 Initialize() 阻塞。
			t.routeInitError(msg)

			// Check if this is a control message that should be routed to the protocol
			// 检查是否为应路由给控制协议的控制消息
			if rawCtrl, ok := msg.(*shared.RawControlMessage); ok {
				// Route control messages to the protocol for request/response correlation
				if t.protocol != nil {
					// HandleIncomingMessage routes control responses to pending requests
					// and forwards non-control messages to the protocol's message stream
					_ = t.protocol.HandleIncomingMessage(t.ctx, rawCtrl.Data)
				}
				// Don't send control messages to msgChan - they're internal to the protocol
				continue
			}

			// Track regular message for stream validation
			t.validator.TrackMessage(msg)
			t.cacheSlashCommands(msg)

			select {
			case t.msgChan <- msg:
				parsedAny = true
			case <-t.ctx.Done():
				return false
			}
		}
		return true
	}

	for {
		fragment, err := reader.ReadSlice('\n')
		if len(fragment) > 0 {
			lineComplete := fragment[len(fragment)-1] == '\n'
			if lineComplete {
				fragment = trimStdoutLineEnding(fragment)
			}

			if lineBuilder.Len()+len(fragment) > lineLimit {
				bufferSize := lineBuilder.Len() + len(fragment)
				select {
				case t.errChan <- shared.NewJSONDecodeError(
					"buffer overflow",
					0,
					fmt.Errorf("buffer size %d exceeds limit %d", bufferSize, lineLimit),
				):
				case <-t.ctx.Done():
					return
				}
				lineBuilder.Reset()
				if !lineComplete {
					if drainErr := drainStdoutLine(reader); drainErr != nil && !errors.Is(drainErr, io.EOF) {
						select {
						case t.errChan <- fmt.Errorf("stdout read error: %w", drainErr):
						case <-t.ctx.Done():
						}
						return
					}
				}
			} else {
				lineBuilder.Write(fragment)
				if lineComplete {
					if !processLine(lineBuilder.String()) {
						return
					}
					lineBuilder.Reset()
				}
			}
		}

		if err == nil {
			continue
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) {
			if lineBuilder.Len() > 0 && !processLine(lineBuilder.String()) {
				return
			}
			break
		}
		select {
		case t.errChan <- fmt.Errorf("stdout read error: %w", err):
		case <-t.ctx.Done():
		}
		return
	}

	if !parsedAny {
		if stderrText := t.readStderrText(64 * 1024); stderrText != "" {
			select {
			case t.errChan <- fmt.Errorf("claude cli produced no stdout messages; stderr:\n%s", stderrText):
			case <-t.ctx.Done():
			}
			return
		}

		select {
		case t.errChan <- fmt.Errorf("claude cli produced no output (no stdout messages). Set WithDebugWriter or WithStderrCallback to inspect stderr, and verify claude CLI is installed/authenticated"):
		case <-t.ctx.Done():
		}
	}
}

func trimStdoutLineEnding(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line
}

func drainStdoutLine(reader *bufio.Reader) error {
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(fragment) > 0 && fragment[len(fragment)-1] == '\n' {
			return nil
		}
		if err == nil || errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return err
	}
}

func (t *Transport) cacheSlashCommands(msg shared.Message) {
	systemMsg, ok := msg.(*shared.SystemMessage)
	if !ok || systemMsg.Subtype != "init" {
		return
	}

	commands := parseSlashCommands(systemMsg.Data["slash_commands"])
	t.mu.Lock()
	t.slashCommands = commands
	t.mu.Unlock()
	t.slashCommandsReadyOnce.Do(func() {
		close(t.slashCommandsReady)
	})
}

func parseSlashCommands(value any) []control.SlashCommand {
	items, ok := value.([]any)
	if !ok {
		if strings, ok := value.([]string); ok {
			return slashCommandsFromNames(strings)
		}
		return []control.SlashCommand{}
	}

	commands := make([]control.SlashCommand, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case string:
			commands = append(commands, control.SlashCommand{Name: normalizeSlashCommandName(typed)})
		case map[string]any:
			if name, ok := typed["name"].(string); ok && name != "" {
				command := control.SlashCommand{Name: normalizeSlashCommandName(name)}
				if desc, ok := typed["description"].(string); ok {
					command.Description = desc
				}
				commands = append(commands, command)
			}
		}
	}
	return commands
}

func slashCommandsFromNames(names []string) []control.SlashCommand {
	commands := make([]control.SlashCommand, 0, len(names))
	for _, name := range names {
		commands = append(commands, control.SlashCommand{Name: normalizeSlashCommandName(name)})
	}
	return commands
}

func normalizeSlashCommandName(name string) string {
	if name == "" || strings.HasPrefix(name, "/") {
		return name
	}
	return "/" + name
}

// handleStderrCallback processes stderr in a separate goroutine.
// Reads line-by-line, strips trailing whitespace, skips empty lines, and
// silently ignores scanner errors.
//
// handleStderrCallback 在独立 goroutine 中处理 stderr：逐行读取、去除尾部空白、
// 跳过空行，并静默忽略扫描错误。
func (t *Transport) handleStderrCallback() {
	defer t.wg.Done()

	scanner := bufio.NewScanner(t.stderrPipe)

	for scanner.Scan() {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		// Strip trailing whitespace (matches Python's rstrip())
		line := strings.TrimRight(scanner.Text(), " \t\r\n")

		// Skip empty lines (matches Python SDK behavior)
		if line == "" {
			continue
		}

		// Call the callback synchronously (matches Python SDK)
		// Recover from panics to prevent crashing the SDK
		func() {
			defer func() {
				_ = recover() // Silently ignore callback panics (matches Python's pass)
			}()
			t.options.StderrCallback(line)
		}()
	}
	// Silently ignore scanner errors (matches Python SDK's except Exception: pass)
}

// routeInitError checks if a message is an error ResultMessage arriving before
// the transport is fully connected, and routes it to the control protocol to
// unblock Initialize().
//
// routeInitError 检查消息是否为在 transport 完全连接前到达的错误 ResultMessage，
// 并将其路由给控制协议以解除 Initialize() 阻塞。
func (t *Transport) routeInitError(msg shared.Message) {
	resultMsg, ok := msg.(*shared.ResultMessage)
	if !ok || t.connected || !resultMsg.IsError || t.protocol == nil {
		return
	}
	t.protocol.HandleControlInitErr(errors.New(formatInitError(resultMsg)))
}

// formatInitError builds a meaningful error string from a ResultMessage that
// arrived during initialization. Prefers Errors, falls back to Result, then Subtype.
//
// formatInitError 从初始化期间到达的 ResultMessage 构建有意义的错误字符串：
// 优先使用 Errors，其次 Result，最后回退到 Subtype。
func formatInitError(msg *shared.ResultMessage) string {
	if len(msg.Errors) > 0 {
		return strings.Join(msg.Errors, "; ")
	}
	if msg.Result != nil && *msg.Result != "" {
		return *msg.Result
	}
	return fmt.Sprintf("initialization failed with subtype: %s", msg.Subtype)
}

func (t *Transport) readStderrText(limit int64) string {
	if t.stderr == nil || limit <= 0 {
		return ""
	}

	_ = t.stderr.Sync()
	if _, err := t.stderr.Seek(0, 0); err != nil {
		return ""
	}

	data, err := io.ReadAll(io.LimitReader(t.stderr, limit))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

// setupStderr configures stderr handling based on options.
// Precedence: StderrCallback > DebugWriter > temp file (default).
//
// setupStderr 根据 options 配置 stderr 处理方式。
// 优先级：StderrCallback > DebugWriter > 临时文件（默认）。
func (t *Transport) setupStderr() error {
	switch {
	case t.options != nil && t.options.StderrCallback != nil:
		// Create pipe for callback-based stderr handling
		stderrPipe, err := t.cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("failed to create stderr pipe: %w", err)
		}
		t.stderrPipe = stderrPipe
	case t.options != nil && t.options.DebugWriter != nil:
		// Use custom debug writer provided by user
		t.cmd.Stderr = t.options.DebugWriter
	default:
		// Isolate stderr using temporary file to prevent deadlocks
		// This matches Python SDK pattern to avoid subprocess pipe deadlocks
		stderrFile, err := os.CreateTemp("", "claude_stderr_*.log")
		if err != nil {
			return fmt.Errorf("failed to create stderr file: %w", err)
		}
		t.stderr = stderrFile
		t.cmd.Stderr = t.stderr
	}
	return nil
}

// setupIoPipes configures stdin, stdout, and stderr pipes for the subprocess.
// For streaming mode, creates a stdin pipe for sending messages. Always creates
// stdout pipe for receiving responses. Stderr is configured via setupStderr.
//
// setupIoPipes 为子进程配置 stdin、stdout、stderr 管道。
// 流式模式下创建 stdin 管道用于发送消息；总是创建 stdout 管道用于接收响应；
// stderr 通过 setupStderr 配置。
func (t *Transport) setupIoPipes() error {
	var err error
	if t.promptArg == nil {
		// Only create stdin pipe if we need to send messages via stdin
		t.stdin, err = t.cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdin pipe: %w", err)
		}
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Handle stderr configuration
	if err := t.setupStderr(); err != nil {
		return err
	}

	return nil
}
