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

	// claudecode.RegisterSkill("echo", func(_ context.Context, args string) (string, error) {
	// 	return "E:" + args, nil
	// })
	// claudecode.RegisterSkill("fail", func(_ context.Context, _ string) (string, error) {
	// 	return "", errors.New("skill failed")
	// })

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
