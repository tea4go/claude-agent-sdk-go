package claudecode

import (
	"context"
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

type SkillRegistryConfig = shared.SkillRegistryConfig

func WithSessionID(sessionID string) Option {
	return func(o *Options) { o.SessionID = &sessionID }
}

func WithSkillImplementations(skills map[string]func(context.Context, string) (string, error)) Option {
	return func(o *Options) { o.SkillImplementations = skills }
}

func WithSkillImplementation(name string, handler func(context.Context, string) (string, error)) Option {
	return func(o *Options) {
		if o.SkillImplementations == nil {
			o.SkillImplementations = make(map[string]func(context.Context, string) (string, error))
		}
		o.SkillImplementations[name] = handler
	}
}

func WithSkillRegistry(root string, names ...string) Option {
	return func(o *Options) {
		o.SkillRegistries = append(o.SkillRegistries, SkillRegistryConfig{Root: root, Names: append([]string(nil), names...)})
	}
}

func WithSkillRegistryAll(root string) Option {
	return func(o *Options) {
		o.SkillRegistries = append(o.SkillRegistries, SkillRegistryConfig{Root: root, Names: []string{}})
	}
}
