package claudecode

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// DiscoverSlashCommands starts a short-lived discovery query and returns the
// native slash commands advertised by the CLI in the system/init message.
//
// DiscoverSlashCommands 发起一个短生命的发现查询，返回 CLI 在 system/init
// 消息中告知的原生斜杠命令。
//
// Slash commands are dynamic: they depend on the current directory, settings,
// plugins, and Skills. The CLI exposes them on session initialization rather
// than through a standalone control request, so this helper opens a temporary
// query, reads init, then closes it immediately.
//
// 斜杠命令是动态的：它们取决于当前目录、设置、插件与技能。CLI 在会话初始化时
// 而非通过独立的控制请求暴露它们，因此该辅助函数会打开一个临时查询、读取 init，然后立即关闭。
func DiscoverSlashCommands(ctx context.Context, opts ...Option) ([]SlashCommand, error) {
	discoveryOpts := append([]Option{}, opts...)
	discoveryOpts = append(discoveryOpts, WithMaxTurns(1))

	iterator, err := Query(ctx, "Hello Claude", discoveryOpts...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = iterator.Close() }()

	return discoverSlashCommandsFromIterator(ctx, iterator)
}

// discoverSlashCommandsFromIterator 从消息迭代器中读取直至遇到 system/init 消息，
// 并从其 slash_commands 字段解析出斜杠命令。
func discoverSlashCommandsFromIterator(ctx context.Context, iterator MessageIterator) ([]SlashCommand, error) {
	for {
		msg, err := iterator.Next(ctx)
		if err != nil {
			if errors.Is(err, ErrNoMoreMessages) {
				break
			}
			return nil, err
		}

		systemMsg, ok := msg.(*SystemMessage)
		if !ok || systemMsg.Subtype != "init" {
			continue
		}

		return slashCommandsFromInitValue(systemMsg.Data["slash_commands"]), nil
	}

	return nil, fmt.Errorf("system init message did not include slash_commands")
}

// slashCommandsFromInitValue 将 init 消息中的 slash_commands 原始值（字符串数组或对象数组）
// 解析为 SlashCommand 列表。
func slashCommandsFromInitValue(value any) []SlashCommand {
	items, ok := value.([]any)
	if !ok {
		if names, ok := value.([]string); ok {
			return slashCommandsFromNames(names)
		}
		return []SlashCommand{}
	}

	commands := make([]SlashCommand, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case string:
			commands = append(commands, SlashCommand{Name: normalizeSlashName(typed)})
		case map[string]any:
			if name, ok := typed["name"].(string); ok && name != "" {
				command := SlashCommand{Name: normalizeSlashName(name)}
				if desc, ok := typed["description"].(string); ok {
					command.Description = desc
				}
				commands = append(commands, command)
			}
		}
	}
	return commands
}

// slashCommandsFromNames 将命令名称列表转换为 SlashCommand 列表。
func slashCommandsFromNames(names []string) []SlashCommand {
	commands := make([]SlashCommand, 0, len(names))
	for _, name := range names {
		commands = append(commands, SlashCommand{Name: normalizeSlashName(name)})
	}
	return commands
}

// normalizeSlashName 为命令名称补全前导的 "/"；空名或已以 "/" 开头时原样返回。
func normalizeSlashName(name string) string {
	if name == "" || strings.HasPrefix(name, "/") {
		return name
	}
	return "/" + name
}
