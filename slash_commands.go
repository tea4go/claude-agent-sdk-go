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
// Slash commands are dynamic: they depend on the current directory, settings,
// plugins, and Skills. The CLI exposes them on session initialization rather
// than through a standalone control request, so this helper opens a temporary
// query, reads init, then closes it immediately.
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

func slashCommandsFromNames(names []string) []SlashCommand {
	commands := make([]SlashCommand, 0, len(names))
	for _, name := range names {
		commands = append(commands, SlashCommand{Name: normalizeSlashName(name)})
	}
	return commands
}

func normalizeSlashName(name string) string {
	if name == "" || strings.HasPrefix(name, "/") {
		return name
	}
	return "/" + name
}
