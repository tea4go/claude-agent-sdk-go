package claudecode

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestParseSkillCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantArgs  string
		wantMatch bool
	}{
		{name: "not_command", input: "hello", wantMatch: false},
		{name: "slash_only", input: "/", wantMatch: false},
		{name: "name_only", input: "/echo", wantName: "echo", wantArgs: "", wantMatch: true},
		{name: "single_arg", input: "/echo hi", wantName: "echo", wantArgs: "hi", wantMatch: true},
		{name: "multi_spaces_preserved", input: "/echo    a   b", wantName: "echo", wantArgs: "a   b", wantMatch: true},
		{name: "slash_not_at_start", input: "hello /echo hi", wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotArgs, gotMatch := parseSkillCommand(tt.input)
			if gotMatch != tt.wantMatch {
				t.Fatalf("match: expected %v, got %v", tt.wantMatch, gotMatch)
			}
			if !tt.wantMatch {
				return
			}
			if gotName != tt.wantName {
				t.Fatalf("name: expected %q, got %q", tt.wantName, gotName)
			}
			if gotArgs != tt.wantArgs {
				t.Fatalf("args: expected %q, got %q", tt.wantArgs, gotArgs)
			}
		})
	}
}

func TestQuerySkillDispatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	skills := map[string]func(context.Context, string) (string, error){
		"echo": func(_ context.Context, args string) (string, error) {
			return "E:" + args, nil
		},
	}

	iter, err := QueryWithTransport(
		ctx,
		"/echo hi",
		newQueryMockTransport(WithQueryConnectError(errors.New("connect should not be called"))),
		func(o *Options) { o.SkillImplementations = skills },
	)
	if err != nil {
		t.Fatalf("QueryWithTransport error: %v", err)
	}
	defer func() { _ = iter.Close() }()

	messages := collectQueryMessages(ctx, t, iter)
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	assistant, ok := messages[0].(*AssistantMessage)
	if !ok {
		t.Fatalf("expected AssistantMessage, got %T", messages[0])
	}
	if len(assistant.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(assistant.Content))
	}
	tb, ok := assistant.Content[0].(*TextBlock)
	if !ok {
		t.Fatalf("expected TextBlock, got %T", assistant.Content[0])
	}
	if tb.Text != "E:hi" {
		t.Fatalf("expected text %q, got %q", "E:hi", tb.Text)
	}

	result, ok := messages[1].(*ResultMessage)
	if !ok {
		t.Fatalf("expected ResultMessage, got %T", messages[1])
	}
	if result.IsError {
		t.Fatalf("expected IsError=false, got true (result=%v)", result.Result)
	}
}

func TestQuerySkillDispatchWithGlobalRegistry(t *testing.T) {
	resetRegisteredSkillsForTest()
	defer resetRegisteredSkillsForTest()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	RegisterSkill("echo", func(_ context.Context, args string) (string, error) {
		return "E:" + args, nil
	})

	iter, err := QueryWithTransport(
		ctx,
		"/echo hi",
		newQueryMockTransport(WithQueryConnectError(errors.New("connect should not be called"))),
	)
	if err != nil {
		t.Fatalf("QueryWithTransport error: %v", err)
	}
	defer func() { _ = iter.Close() }()

	messages := collectQueryMessages(ctx, t, iter)
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	assistant, ok := messages[0].(*AssistantMessage)
	if !ok {
		t.Fatalf("expected AssistantMessage, got %T", messages[0])
	}
	tb, ok := assistant.Content[0].(*TextBlock)
	if !ok {
		t.Fatalf("expected TextBlock, got %T", assistant.Content[0])
	}
	if tb.Text != "E:hi" {
		t.Fatalf("expected text %q, got %q", "E:hi", tb.Text)
	}
}

func TestQueryUnknownSkillFallsBackToTransport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	skills := map[string]func(context.Context, string) (string, error){
		"echo": func(_ context.Context, args string) (string, error) {
			return "E:" + args, nil
		},
	}

	iter, err := QueryWithTransport(
		ctx,
		"/unknown hi",
		newQueryMockTransport(WithQueryConnectError(errors.New("connect called"))),
		func(o *Options) { o.SkillImplementations = skills },
	)
	if err != nil {
		t.Fatalf("QueryWithTransport error: %v", err)
	}
	defer func() { _ = iter.Close() }()

	_, nextErr := iter.Next(ctx)
	if nextErr == nil || nextErr.Error() != "failed to connect transport: connect called" {
		t.Fatalf("expected connect error, got %v", nextErr)
	}
}

func TestClientSkillDispatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := newClientMockTransport()
	skills := map[string]func(context.Context, string) (string, error){
		"echo": func(_ context.Context, args string) (string, error) {
			return "E:" + args, nil
		},
	}

	client := NewClientWithTransport(transport, func(o *Options) { o.SkillImplementations = skills })
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect error: %v", err)
	}
	defer func() { _ = client.Disconnect() }()

	if err := client.Query(ctx, "/echo hi"); err != nil {
		t.Fatalf("query error: %v", err)
	}

	if transport.getSentMessageCount() != 0 {
		t.Fatalf("expected 0 messages sent to transport, got %d", transport.getSentMessageCount())
	}

	iter := client.ReceiveResponse(ctx)
	if iter == nil {
		t.Fatal("expected iterator, got nil")
	}

	msg1, err := iter.Next(ctx)
	if err != nil {
		t.Fatalf("next1 error: %v", err)
	}
	assistant, ok := msg1.(*AssistantMessage)
	if !ok {
		t.Fatalf("expected AssistantMessage, got %T", msg1)
	}
	tb, ok := assistant.Content[0].(*TextBlock)
	if !ok || tb.Text != "E:hi" {
		t.Fatalf("expected text %q, got %T %v", "E:hi", assistant.Content[0], assistant.Content[0])
	}

	msg2, err := iter.Next(ctx)
	if err != nil {
		t.Fatalf("next2 error: %v", err)
	}
	result, ok := msg2.(*ResultMessage)
	if !ok {
		t.Fatalf("expected ResultMessage, got %T", msg2)
	}
	if result.IsError {
		t.Fatalf("expected IsError=false, got true")
	}
}

func TestClientSkillDispatchWithGlobalRegistry(t *testing.T) {
	resetRegisteredSkillsForTest()
	defer resetRegisteredSkillsForTest()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	RegisterSkill("echo", func(_ context.Context, args string) (string, error) {
		return "E:" + args, nil
	})

	transport := newClientMockTransport()
	client := NewClientWithTransport(transport)
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect error: %v", err)
	}
	defer func() { _ = client.Disconnect() }()

	if err := client.Query(ctx, "/echo hi"); err != nil {
		t.Fatalf("query error: %v", err)
	}

	if transport.getSentMessageCount() != 0 {
		t.Fatalf("expected 0 messages sent to transport, got %d", transport.getSentMessageCount())
	}

	iter := client.ReceiveResponse(ctx)
	if iter == nil {
		t.Fatal("expected iterator, got nil")
	}

	msg1, err := iter.Next(ctx)
	if err != nil {
		t.Fatalf("next1 error: %v", err)
	}
	assistant, ok := msg1.(*AssistantMessage)
	if !ok {
		t.Fatalf("expected AssistantMessage, got %T", msg1)
	}
	tb, ok := assistant.Content[0].(*TextBlock)
	if !ok || tb.Text != "E:hi" {
		t.Fatalf("expected text %q, got %T %v", "E:hi", assistant.Content[0], assistant.Content[0])
	}
}

func TestClientUnknownSkillFallsBackToTransport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := newClientMockTransport()
	skills := map[string]func(context.Context, string) (string, error){
		"echo": func(_ context.Context, args string) (string, error) {
			return "E:" + args, nil
		},
	}

	client := NewClientWithTransport(transport, func(o *Options) { o.SkillImplementations = skills })
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect error: %v", err)
	}
	defer func() { _ = client.Disconnect() }()

	if err := client.Query(ctx, "/unknown hi"); err != nil {
		t.Fatalf("query error: %v", err)
	}

	if transport.getSentMessageCount() != 1 {
		t.Fatalf("expected 1 message sent to transport, got %d", transport.getSentMessageCount())
	}
}
