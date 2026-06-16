// Package main demonstrates loading Skills from an external registry directory.
//
// The example uses a pure Skills directory where each direct child is a Skill
// package containing SKILL.md plus optional scripts, JSON, or other assets. The
// SDK exposes the selected Skills through a temporary Claude plugin wrapper, so
// nothing is copied into the project or ~/.claude. Use WithSkillsList or
// WithSkillsAll alongside WithSkillRegistry when you also want project/global
// Skills to remain available.
//
// Run:
//
//	go run main.go
//
// Optional:
//
//	CLAUDE_SKILL_REGISTRY_ROOT=/path/to/skills go run main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	claudecode "github.com/tea4go/claude-agent-sdk-go"
)

const defaultSkillRegistryRoot = "/Users/zhangym/Library/Preferences/WhaleTerm/skills"

func main() {
	registryRoot := os.Getenv("CLAUDE_SKILL_REGISTRY_ROOT")
	if registryRoot == "" {
		registryRoot = defaultSkillRegistryRoot
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	iterator, err := claudecode.Query(
		ctx,
		// "Use the find-skills skill to briefly explain what it helps with.",
		"请校验/Users/zhangym/code/work/utils/claude-agent-sdk-go/examples/23_skill_registry/test-data.json的格式，并且告诉我你是否有调用一些辅助agent skill工具来协助校验",
		claudecode.WithSkillRegistry(
			registryRoot,
			"zym-skills",
		),
		// Optional: append specific user/project Skills that Claude already
		// discovers from ~/.claude/skills or .claude/skills.
		// claudecode.WithSkillsList("project-review", "validate-json"),
		//
		// Optional: allow every discovered user/project Skill while still
		// adding only the registry Skills selected above.
		// claudecode.WithSkillsAll(),
		claudecode.WithDebugWriter(os.Stderr),
	)
	if err != nil {
		panic(err)
	}
	defer iterator.Close()

	for {
		msg, err := iterator.Next(ctx)
		if err != nil {
			if errors.Is(err, claudecode.ErrNoMoreMessages) {
				break
			}
			panic(err)
		}

		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				if text, ok := block.(*claudecode.TextBlock); ok {
					fmt.Println(text.Text)
				}
			}
		case *claudecode.ResultMessage:
			if m.IsError {
				fmt.Printf("error: %v\n", m.Errors)
				return
			}
		}
	}
}
