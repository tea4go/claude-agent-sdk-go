package claudecode

import (
	"context"
	"strings"
	"sync"
)

var registeredSkillsMu sync.RWMutex
var registeredSkills = map[string]func(context.Context, string) (string, error){}

func RegisterSkill(name string, handler func(context.Context, string) (string, error)) {
	if name == "" || handler == nil {
		return
	}
	registeredSkillsMu.Lock()
	registeredSkills[name] = handler
	registeredSkillsMu.Unlock()
}

func RegisterSkills(skills map[string]func(context.Context, string) (string, error)) {
	if len(skills) == 0 {
		return
	}
	registeredSkillsMu.Lock()
	for name, handler := range skills {
		if name == "" || handler == nil {
			continue
		}
		registeredSkills[name] = handler
	}
	registeredSkillsMu.Unlock()
}

func resetRegisteredSkillsForTest() {
	registeredSkillsMu.Lock()
	registeredSkills = map[string]func(context.Context, string) (string, error){}
	registeredSkillsMu.Unlock()
}

func parseSkillCommand(input string) (string, string, bool) {
	if !strings.HasPrefix(input, "/") {
		return "", "", false
	}

	rest := input[1:]
	if rest == "" {
		return "", "", false
	}

	nameEnd := strings.IndexAny(rest, " \t\r\n")
	if nameEnd == 0 {
		return "", "", false
	}

	var name string
	var args string
	if nameEnd == -1 {
		name = rest
		args = ""
	} else {
		name = rest[:nameEnd]
		args = strings.TrimLeft(rest[nameEnd:], " \t")
	}

	if name == "" {
		return "", "", false
	}

	return name, args, true
}

type skillIterator struct {
	messages []Message
	index    int
	closed   bool
}

func (it *skillIterator) Next(ctx context.Context) (Message, error) {
	if it.closed {
		return nil, ErrNoMoreMessages
	}

	select {
	case <-ctx.Done():
		it.closed = true
		return nil, ctx.Err()
	default:
	}

	if it.index >= len(it.messages) {
		it.closed = true
		return nil, ErrNoMoreMessages
	}

	msg := it.messages[it.index]
	it.index++
	return msg, nil
}

func (it *skillIterator) Close() error {
	it.closed = true
	return nil
}

func getSkillHandler(options *Options, name string) (func(context.Context, string) (string, error), bool) {
	if options != nil && options.SkillImplementations != nil {
		handler, exists := options.SkillImplementations[name]
		return handler, exists && handler != nil
	}

	registeredSkillsMu.RLock()
	handler, exists := registeredSkills[name]
	registeredSkillsMu.RUnlock()
	return handler, exists && handler != nil
}

func tryRunSkill(ctx context.Context, prompt string, options *Options, sessionID string) (MessageIterator, bool) {
	name, args, ok := parseSkillCommand(prompt)
	if !ok {
		return nil, false
	}

	handler, exists := getSkillHandler(options, name)
	if !exists {
		return nil, false
	}

	output, err := handler(ctx, args)

	if sessionID == "" {
		sessionID = defaultSessionID
	}

	model := "sdk-skill"

	if err != nil {
		errText := err.Error()
		return &skillIterator{
			messages: []Message{
				&AssistantMessage{
					Content: []ContentBlock{&TextBlock{Text: errText}},
					Model:   model,
				},
				&ResultMessage{
					Subtype:       "error",
					DurationMs:    0,
					DurationAPIMs: 0,
					IsError:       true,
					Errors:        []string{errText},
					NumTurns:      0,
					SessionID:     sessionID,
					Result:        &errText,
				},
			},
		}, true
	}

	return &skillIterator{
		messages: []Message{
			&AssistantMessage{
				Content: []ContentBlock{&TextBlock{Text: output}},
				Model:   model,
			},
			&ResultMessage{
				Subtype:       "success",
				DurationMs:    0,
				DurationAPIMs: 0,
				IsError:       false,
				NumTurns:      0,
				SessionID:     sessionID,
			},
		},
	}, true
}
