// Package main demonstrates discovering native slash commands for UI suggestions.
package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	claudecode "github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Slash Commands Example")
	fmt.Println("Discover native slash commands from the system/init message")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	commands, err := claudecode.DiscoverSlashCommands(ctx)
	if err != nil {
		if cliErr := claudecode.AsCLINotFoundError(err); cliErr != nil {
			fmt.Printf("Claude CLI not found: %v\n", cliErr)
			fmt.Println("Install with: npm install -g @anthropic-ai/claude-code")
			return
		}
		if connErr := claudecode.AsConnectionError(err); connErr != nil {
			fmt.Printf("Connection failed: %v\n", connErr)
			return
		}
		log.Fatalf("Slash commands example failed: %v", err)
	}

	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})

	fmt.Printf("\nFound %d slash commands:\n", len(commands))
	for _, command := range commands {
		printSlashCommand(command)
	}

	fmt.Println("\nFrontend usage:")
	fmt.Println("  Cache this list after session init, then show matching names when the user types '/'.")
}

func printSlashCommand(command claudecode.SlashCommand) {
	if command.Description == "" {
		fmt.Printf("  %s\n", command.Name)
		return
	}
	fmt.Printf("  %-18s %s\n", command.Name, command.Description)
}
