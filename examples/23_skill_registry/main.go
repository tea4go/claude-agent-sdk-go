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
//
// Package main 演示如何从外部 Skill 注册表目录加载技能，
// 并通过临时插件封装把它们暴露给 Claude 使用。
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
		// 没有显式环境变量时，回退到示例里的默认技能目录。
		registryRoot = defaultSkillRegistryRoot
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	iterator, err := claudecode.Query(
		ctx,
		// "Use the find-skills skill to briefly explain what it helps with.",
		// 这里给出一个真实的查询例子，用来验证外部 Skill 是否被成功暴露。
		"请校验/Users/zhangym/code/work/utils/claude-agent-sdk-go/examples/23_skill_registry/test-data.json的格式，并且告诉我你是否有调用一些辅助agent skill工具来协助校验",
		claudecode.WithSkillRegistry(
			registryRoot,
			"zym-skills",
		),
		// 可选：额外启用用户级或项目级已存在的技能。
		// claudecode.WithSkillsList("project-review", "validate-json"),
		//
		// 可选：保留全部已发现的用户/项目技能，同时再挂入上面的注册表技能。
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

		// 这里只关心技能输出正文和最终结果状态。
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
