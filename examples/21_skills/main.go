// Package main demonstrates invoking registered in-process skills through the
// SDK query interface.
//
// Package main 演示如何通过 Query 接口调用已注册的进程内技能。
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	claudecode "github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	// 这里保留了注册技能的最小示例，默认注释掉，便于按需打开做本地实验。
	// claudecode.RegisterSkill("echo", func(_ context.Context, args string) (string, error) {
	// 	return "E:" + args, nil
	// })
	// claudecode.RegisterSkill("fail", func(_ context.Context, _ string) (string, error) {
	// 	return "", errors.New("skill failed")
	// })

	// 直接用 "/skill args" 形式发起查询，SDK 会在本地尝试解析并执行技能。
	iterator, err := claudecode.Query(
		ctx,
		"/find-skills superpowers",
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
			fmt.Printf("error: %v\n", err)
			panic(err)
		}

		// 助手消息输出技能正文；结果消息补充最终状态。
		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				if tb, ok := block.(*claudecode.TextBlock); ok {
					fmt.Println(tb.Text)
				}
			}
		case *claudecode.ResultMessage:
			if m.IsError {
				if len(m.Errors) > 0 {
					fmt.Printf("error: %s\n", m.Errors[0])
					return
				}
				if m.Result != nil && *m.Result != "" {
					fmt.Printf("error: %s\n", *m.Result)
					return
				}
				fmt.Println("error: unknown")
				return
			}
			if m.Result != nil && *m.Result != "" {
				fmt.Println(*m.Result)
			}
		}
	}
}
