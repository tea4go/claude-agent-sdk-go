package claudecode

import (
	"context"
	"testing"
)

func TestDiscoverSlashCommandsFromIterator(t *testing.T) {
	ctx := context.Background()
	iterator := &slashCommandsFakeIterator{
		messages: []Message{
			&SystemMessage{
				Subtype: "init",
				Data: map[string]any{
					"slash_commands": []any{
						"compact",
						"/clear",
						map[string]any{
							"name":        "context",
							"description": "Show context usage",
						},
					},
				},
			},
		},
	}

	commands, err := discoverSlashCommandsFromIterator(ctx, iterator)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(commands))
	}
	assertSlashCommand(t, commands[0], "/compact", "")
	assertSlashCommand(t, commands[1], "/clear", "")
	assertSlashCommand(t, commands[2], "/context", "Show context usage")
}

func TestDiscoverSlashCommandsFromIteratorMissingInit(t *testing.T) {
	ctx := context.Background()
	iterator := &slashCommandsFakeIterator{
		messages: []Message{
			&AssistantMessage{},
		},
	}

	_, err := discoverSlashCommandsFromIterator(ctx, iterator)
	if err == nil {
		t.Fatal("expected error for missing slash_commands, got nil")
	}
}

func assertSlashCommand(t *testing.T, command SlashCommand, name, description string) {
	t.Helper()
	if command.Name != name {
		t.Errorf("expected name %q, got %q", name, command.Name)
	}
	if command.Description != description {
		t.Errorf("expected description %q, got %q", description, command.Description)
	}
}

type slashCommandsFakeIterator struct {
	messages []Message
	index    int
}

func (f *slashCommandsFakeIterator) Next(_ context.Context) (Message, error) {
	if f.index >= len(f.messages) {
		return nil, ErrNoMoreMessages
	}
	msg := f.messages[f.index]
	f.index++
	return msg, nil
}

func (f *slashCommandsFakeIterator) Close() error {
	return nil
}

var _ MessageIterator = (*slashCommandsFakeIterator)(nil)
