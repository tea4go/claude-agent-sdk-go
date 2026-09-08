// Package main demonstrates Plugin Configuration.
//
// This example shows how to configure local plugins for the Claude Code
// CLI using the SDK. Plugins extend Claude's capabilities with custom
// commands, tools, and integrations. Plugin configuration enables:
// - Loading custom plugins from local paths
// - Configuring multiple plugins simultaneously
// - Extending Claude Code with domain-specific functionality
// - Integration with existing plugin ecosystems
//
// Key components:
// - WithLocalPlugin: Convenience function for local plugin paths
// - WithPlugin: Add a single plugin with explicit configuration
// - WithPlugins: Add multiple plugins at once
// - SdkPluginConfig: Plugin configuration structure
// - SdkPluginTypeLocal: Plugin type constant for local plugins
//
// NOTE: This example demonstrates the API configuration without requiring
// actual plugins to exist. In production, plugins must be valid Claude Code
// plugin directories.
//
// Run: go run main.go
//
// Package main 演示插件配置接口，重点说明本地插件、显式配置和批量配置三种常见写法。
package main

import (
	"fmt"

	claudecode "github.com/tea4go/claude-agent-sdk-go"
)

func main() {
	fmt.Println("Claude Agent SDK - Plugins Configuration Example")
	fmt.Println("=================================================")
	fmt.Println()

	// 示例 1：最简洁的单插件配置方式。
	fmt.Println("--- Example 1: Single Plugin Configuration ---")
	fmt.Println("Using WithLocalPlugin convenience function...")
	demonstrateSinglePlugin()

	// 示例 2：显式构造插件配置对象。
	fmt.Println()
	fmt.Println("--- Example 2: Explicit Plugin Configuration ---")
	fmt.Println("Using WithPlugin with SdkPluginConfig...")
	demonstrateExplicitConfig()

	// 示例 3：一次性配置多个插件。
	fmt.Println()
	fmt.Println("--- Example 3: Multiple Plugins ---")
	fmt.Println("Configuring multiple plugins with WithPlugins...")
	demonstrateMultiplePlugins()

	// 示例 4：总结几种常见的组合写法。
	fmt.Println()
	fmt.Println("--- Example 4: Configuration Patterns ---")
	fmt.Println("Common plugin configuration patterns...")
	showConfigurationPatterns()

	fmt.Println()
	fmt.Println("Plugins configuration example completed!")
}

// demonstrateSinglePlugin shows the WithLocalPlugin convenience function.
//
// demonstrateSinglePlugin 展示最简单的本地插件配置方式。
func demonstrateSinglePlugin() {
	// WithLocalPlugin 适合只追加一个本地插件路径的场景。
	pluginPath := "/path/to/my-plugin"

	// 这里只演示配置对象如何挂到 client 上，不实际连接 CLI。
	client := claudecode.NewClient(
		claudecode.WithLocalPlugin(pluginPath),
	)

	fmt.Printf("Configured plugin path: %s\n", pluginPath)
	fmt.Printf("Client created with plugin configuration\n")

	// 纯配置示例无需真正连接，重点在查看调用形式。
	_ = client
}

// demonstrateExplicitConfig shows using SdkPluginConfig directly.
//
// demonstrateExplicitConfig 演示如何显式构造 SdkPluginConfig。
func demonstrateExplicitConfig() {
	// 当需要先在代码里组织配置，再统一传入时，这种方式更直观。
	pluginConfig := claudecode.SdkPluginConfig{
		Type: claudecode.SdkPluginTypeLocal,
		Path: "/path/to/custom-plugin",
	}

	fmt.Printf("Plugin Type: %s\n", pluginConfig.Type)
	fmt.Printf("Plugin Path: %s\n", pluginConfig.Path)
	fmt.Println()

	// 通过 WithPlugin 把显式配置追加到客户端。
	client := claudecode.NewClient(
		claudecode.WithPlugin(pluginConfig),
	)

	fmt.Println("Client created with explicit plugin configuration")
	_ = client
}

// demonstrateMultiplePlugins shows configuring multiple plugins.
//
// demonstrateMultiplePlugins 演示批量配置多个插件。
func demonstrateMultiplePlugins() {
	// 批量场景下通常先准备一个切片，再统一传给 WithPlugins。
	plugins := []claudecode.SdkPluginConfig{
		{
			Type: claudecode.SdkPluginTypeLocal,
			Path: "/plugins/code-formatter",
		},
		{
			Type: claudecode.SdkPluginTypeLocal,
			Path: "/plugins/test-runner",
		},
		{
			Type: claudecode.SdkPluginTypeLocal,
			Path: "/plugins/deployment-helper",
		},
	}

	fmt.Printf("Configuring %d plugins:\n", len(plugins))
	for i, p := range plugins {
		fmt.Printf("  %d. [%s] %s\n", i+1, p.Type, p.Path)
	}
	fmt.Println()

	// 一次性挂入全部插件配置。
	client := claudecode.NewClient(
		claudecode.WithPlugins(plugins),
	)

	fmt.Println("Client created with multiple plugins")
	_ = client
}

// showConfigurationPatterns demonstrates common plugin patterns.
//
// showConfigurationPatterns 汇总几种常见的插件配置模式。
func showConfigurationPatterns() {
	fmt.Println("Pattern 1: Chaining WithLocalPlugin calls")
	fmt.Println("  claudecode.NewClient(")
	fmt.Println("      claudecode.WithLocalPlugin(\"/plugins/formatter\"),")
	fmt.Println("      claudecode.WithLocalPlugin(\"/plugins/linter\"),")
	fmt.Println("  )")
	fmt.Println()

	fmt.Println("Pattern 2: Using WithPlugins for bulk configuration")
	fmt.Println("  plugins := []claudecode.SdkPluginConfig{...}")
	fmt.Println("  claudecode.NewClient(claudecode.WithPlugins(plugins))")
	fmt.Println()

	fmt.Println("Pattern 3: Combining with other options")
	fmt.Println("  claudecode.WithClient(ctx, handler,")
	fmt.Println("      claudecode.WithLocalPlugin(\"/path/to/plugin\"),")
	fmt.Println("      claudecode.WithAllowedTools(\"Read\", \"Write\"),")
	fmt.Println("      claudecode.WithMaxTurns(5),")
	fmt.Println("  )")
	fmt.Println()

	fmt.Println("Available plugin types:")
	fmt.Printf("  - SdkPluginTypeLocal: %q\n", claudecode.SdkPluginTypeLocal)
}
