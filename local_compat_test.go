package claudecode

import (
	"context"
	"errors"
	"testing"
)

func TestQueryNextContextCancelBeforeConnect(t *testing.T) {
	transport := newQueryMockTransport()
	iter, err := QueryWithTransport(context.Background(), "hello", transport)
	if err != nil {
		t.Fatal(err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := iter.Next(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Next() error = %v, want context.Canceled", err)
	}
	transport.mu.RLock()
	connected := transport.connected
	transport.mu.RUnlock()
	if connected {
		t.Fatal("cancelled Next started the CLI transport")
	}
}

func TestSessionIDOptionValidation(t *testing.T) {
	valid := "123e4567-e89b-12d3-a456-426614174000"
	if err := NewOptions(WithSessionID(valid)).Validate(); err != nil {
		t.Fatal(err)
		return
	}
	if err := NewOptions(WithSessionID("invalid")).Validate(); err == nil {
		t.Fatal("invalid session ID accepted")
	}
	if err := NewOptions(WithSessionID(valid), WithResume(valid)).Validate(); err == nil {
		t.Fatal("session ID and resume accepted together")
	}
}

func TestMergedStreamDeliversBufferedResultBeforeExitError(t *testing.T) {
	transportMessages := make(chan Message, 2)
	transportErrors := make(chan error, 1)
	injected := make(chan Message, 1)
	out := make(chan Message, 3)
	errOut := make(chan error, 1)
	done := make(chan struct{})
	transportMessages <- &AssistantMessage{Model: "cli"}
	transportMessages <- &ResultMessage{Subtype: "success"}
	close(transportMessages)
	exitErr := errors.New("CLI exited")
	transportErrors <- exitErr
	close(transportErrors)
	injected <- &AssistantMessage{Model: "sdk-skill"}
	go mergeMessageChannels(context.Background(), out, errOut, newStreamReader(transportMessages, transportErrors), injected, done)
	reader := newStreamReader(out, errOut)
	seenResult := false
	seenSkill := false
	for {
		msg, err := reader.next(context.Background(), nil)
		if err != nil {
			if !errors.Is(err, exitErr) {
				t.Fatalf("terminal error = %v, want %v", err, exitErr)
			}
			if !seenResult || !seenSkill {
				t.Fatalf("missing buffered messages: result=%v skill=%v", seenResult, seenSkill)
			}
			break
		}
		if _, ok := msg.(*ResultMessage); ok {
			seenResult = true
		}
		if m, ok := msg.(*AssistantMessage); ok && m.Model == "sdk-skill" {
			seenSkill = true
		}
	}
	<-done
}
