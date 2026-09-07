package claudecode

import (
	"context"
	"strings"
	"sync"
)

var registeredSkillsMu sync.RWMutex
var registeredSkills = map[string]func(context.Context, string) (string, error){}

// RegisterSkill 注册一个全局技能处理函数；name 为空或 handler 为 nil 时忽略。
func RegisterSkill(name string, handler func(context.Context, string) (string, error)) {
	if name == "" || handler == nil {
		return
	}
	registeredSkillsMu.Lock()
	registeredSkills[name] = handler
	registeredSkillsMu.Unlock()
}

// RegisterSkills 批量注册多个全局技能处理函数；名称为空或处理函数为 nil 的条目被跳过。
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

// parseSkillCommand 解析形如 "/name args" 的技能命令，返回名称、参数及是否为合法技能命令。
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

// skillIterator 将预先生成的消息列表包装为 MessageIterator，供技能命令的同步输出使用。
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

// getSkillHandler 查找指定名称的技能处理函数：优先使用 options.SkillImplementations，
// 否则回退到全局注册表。
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

// tryRunSkill 尝试将 prompt 解析为技能命令并就地执行；命中时返回包含结果的迭代器与 true。
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
