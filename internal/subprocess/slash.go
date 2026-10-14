package subprocess

import (
	"context"
	"fmt"
	"strings"

	"github.com/tea4go/claude-agent-sdk-go/internal/control"
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

func (t *Transport) cacheSlashCommands(msg shared.Message) {
	system, ok := msg.(*shared.SystemMessage)
	if !ok || system.Subtype != "init" {
		return
	}
	commands := parseSlashCommands(system.Data["slash_commands"])
	t.slashMu.Lock()
	t.slashCommands = commands
	t.slashMu.Unlock()
	t.slashCommandsReadyOnce.Do(func() { close(t.slashCommandsReady) })
}

func parseSlashCommands(value any) []control.SlashCommand {
	var items []any
	switch v := value.(type) {
	case []any:
		items = v
	case []string:
		for _, name := range v {
			items = append(items, name)
		}
	}
	commands := make([]control.SlashCommand, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case string:
			commands = append(commands, control.SlashCommand{Name: normalizeSlashCommandName(v)})
		case map[string]any:
			if name, ok := v["name"].(string); ok && name != "" {
				command := control.SlashCommand{Name: normalizeSlashCommandName(name)}
				command.Description, _ = v["description"].(string)
				commands = append(commands, command)
			}
		}
	}
	return commands
}

func normalizeSlashCommandName(name string) string {
	if name == "" || strings.HasPrefix(name, "/") {
		return name
	}
	return "/" + name
}

func (t *Transport) copySlashCommands() []control.SlashCommand {
	t.slashMu.RLock()
	defer t.slashMu.RUnlock()
	return append([]control.SlashCommand(nil), t.slashCommands...)
}

func (t *Transport) GetSlashCommands(ctx context.Context) ([]control.SlashCommand, error) {
	t.mu.RLock()
	if !t.connected {
		t.mu.RUnlock()
		return nil, fmt.Errorf("transport not connected")
	}
	ready, transportCtx := t.slashCommandsReady, t.ctx
	t.mu.RUnlock()
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		select {
		case <-ready:
			return t.copySlashCommands(), nil
		default:
			return nil, fmt.Errorf("slash commands not available before system init")
		}
	}
	select {
	case <-ready:
		return t.copySlashCommands(), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-transportCtx.Done():
		return nil, fmt.Errorf("transport closed before slash commands were received")
	}
}
