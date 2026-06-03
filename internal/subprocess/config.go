package subprocess

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/tea4go/claude-agent-sdk-go/internal/cli"
	"github.com/tea4go/claude-agent-sdk-go/internal/control"
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

const defaultSkillRegistryPluginName = "sdk-skill-registry"

// generateMcpConfigFile creates a temporary MCP config file from options.McpServers.
// Returns the file path. The file is stored in t.mcpConfigFile for cleanup.
func (t *Transport) generateMcpConfigFile(options *shared.Options) (string, error) {
	// Build servers map, stripping Instance field from SDK servers for CLI serialization
	// The CLI doesn't need the Go instance - it routes mcp_message requests to the SDK
	serversForCLI := make(map[string]any)
	for name, config := range options.McpServers {
		if sdkConfig, ok := config.(*shared.McpSdkServerConfig); ok {
			// SDK servers: only send type and name to CLI (the Go Instance
			// stays in-process). AlwaysLoad must be propagated explicitly
			// since we're not relying on struct json tags here.
			entry := map[string]any{
				"type": string(sdkConfig.Type),
				"name": sdkConfig.Name,
			}
			if sdkConfig.AlwaysLoad {
				entry["alwaysLoad"] = true
			}
			serversForCLI[name] = entry
		} else {
			// External servers: pass as-is
			serversForCLI[name] = config
		}
	}

	// Create the MCP config structure matching Claude CLI expected format
	mcpConfig := map[string]interface{}{
		"mcpServers": serversForCLI,
	}

	// Marshal to JSON
	configData, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "claude_mcp_config_*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write config data
	if _, err := tmpFile.Write(configData); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write MCP config: %w", err)
	}

	// Sync to ensure data is written
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to sync MCP config file: %w", err)
	}

	// Store for cleanup later
	t.mcpConfigFile = tmpFile

	return tmpFile.Name(), nil
}

// GetValidator returns the stream validator for diagnostic purposes.
// This allows clients to check for validation issues like missing tool results.
func (t *Transport) GetValidator() *shared.StreamValidator {
	return t.validator
}

// SetModel changes the AI model during a streaming session.
// This method requires control protocol integration which is only available
// in streaming mode (when closeStdin is false).
func (t *Transport) SetModel(ctx context.Context, model *string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		return fmt.Errorf("transport not connected")
	}

	// Control protocol integration is only available in streaming mode
	if t.closeStdin {
		return fmt.Errorf("SetModel not available in one-shot mode")
	}

	// Delegate to control protocol
	if t.protocol == nil {
		return fmt.Errorf("control protocol not initialized")
	}

	return t.protocol.SetModel(ctx, model)
}

// SetPermissionMode changes the permission mode during a streaming session.
// This method requires control protocol integration which is only available
// in streaming mode (when closeStdin is false).
func (t *Transport) SetPermissionMode(ctx context.Context, mode shared.PermissionMode) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		return fmt.Errorf("transport not connected")
	}

	// Control protocol integration is only available in streaming mode
	if t.closeStdin {
		return fmt.Errorf("SetPermissionMode not available in one-shot mode")
	}

	// Delegate to control protocol
	if t.protocol == nil {
		return fmt.Errorf("control protocol not initialized")
	}

	return t.protocol.SetPermissionMode(ctx, string(mode))
}

// RewindFiles reverts tracked files to their state at a specific user message.
// This method requires control protocol integration which is only available
// in streaming mode (when closeStdin is false).
// Returns error if not connected, not in streaming mode, or protocol not initialized.
func (t *Transport) RewindFiles(ctx context.Context, userMessageID string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		return fmt.Errorf("transport not connected")
	}

	// Control protocol integration is only available in streaming mode
	if t.closeStdin {
		return fmt.Errorf("RewindFiles not available in one-shot mode")
	}

	// Delegate to control protocol
	if t.protocol == nil {
		return fmt.Errorf("control protocol not initialized")
	}

	return t.protocol.RewindFiles(ctx, userMessageID)
}

// GetMcpStatus returns the connection status of all configured MCP servers.
// This method requires control protocol integration which is only available
// in streaming mode (when closeStdin is false).
func (t *Transport) GetMcpStatus(ctx context.Context) (*control.McpStatusResponse, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.connected {
		return nil, fmt.Errorf("transport not connected")
	}

	if t.closeStdin {
		return nil, fmt.Errorf("GetMcpStatus not available in one-shot mode")
	}

	if t.protocol == nil {
		return nil, fmt.Errorf("internal error: transport connected but control protocol is nil")
	}

	return t.protocol.GetMcpStatus(ctx)
}

// GetSlashCommands returns slash commands available in the current session.
// Commands are discovered from the system/init message emitted by the CLI.
func (t *Transport) GetSlashCommands(ctx context.Context) ([]control.SlashCommand, error) {
	t.mu.RLock()
	if !t.connected {
		t.mu.RUnlock()
		return nil, fmt.Errorf("transport not connected")
	}
	if t.closeStdin {
		t.mu.RUnlock()
		return nil, fmt.Errorf("GetSlashCommands not available in one-shot mode")
	}
	ready := t.slashCommandsReady
	transportCtx := t.ctx
	t.mu.RUnlock()

	if ready == nil {
		return []control.SlashCommand{}, nil
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		select {
		case <-ready:
			return t.copySlashCommands(), nil
		default:
			return nil, fmt.Errorf("slash commands not available before system init; send a query or wait with a context deadline")
		}
	}

	select {
	case <-ready:
		return t.copySlashCommands(), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-transportCtx.Done():
		return nil, fmt.Errorf("transport closed before slash commands were received")
	}
}

func (t *Transport) copySlashCommands() []control.SlashCommand {
	t.mu.RLock()
	defer t.mu.RUnlock()
	commands := make([]control.SlashCommand, len(t.slashCommands))
	copy(commands, t.slashCommands)
	return commands
}

// buildProtocolOptions constructs control protocol options from transport configuration.
func (t *Transport) buildProtocolOptions() []control.ProtocolOption {
	var opts []control.ProtocolOption

	// Wire permission callback if configured
	if t.options != nil && t.options.CanUseTool != nil {
		// Create adapter that converts between shared.Options (any types)
		// and control package (strongly-typed) to avoid import cycles
		optionsCallback := t.options.CanUseTool
		opts = append(opts,
			control.WithCanUseToolCallback(func(
				ctx context.Context,
				toolName string,
				input map[string]any,
				permCtx control.ToolPermissionContext,
			) (control.PermissionResult, error) {
				// Call the Options callback with any-typed permCtx
				result, err := optionsCallback(ctx, toolName, input, permCtx)
				if err != nil {
					return nil, err
				}

				// Convert result back to strongly-typed PermissionResult
				if pr, ok := result.(control.PermissionResult); ok {
					return pr, nil
				}

				// Fallback: deny if result type is unexpected
				fmt.Fprintf(os.Stderr, "claude-agent-sdk: CanUseTool callback returned unexpected type %T, denying\n", result)
				return control.NewPermissionResultDeny("invalid permission result type"), nil
			}))
	}

	// Wire hooks if configured
	if t.options != nil && t.options.Hooks != nil {
		// Convert from any to strongly-typed hooks map
		if hooks, ok := t.options.Hooks.(map[control.HookEvent][]control.HookMatcher); ok {
			opts = append(opts, control.WithHooks(hooks))
		} else {
			fmt.Fprintf(os.Stderr, "claude-agent-sdk: Hooks option has unexpected type %T, hooks will not be registered\n", t.options.Hooks)
		}
	}

	// Wire SDK MCP servers to protocol.
	if t.options != nil && len(t.options.McpServers) > 0 {
		sdkServers := make(map[string]control.McpServer)
		for name, config := range t.options.McpServers {
			if sdkConfig, ok := config.(*shared.McpSdkServerConfig); ok && sdkConfig.Instance != nil {
				sdkServers[name] = sdkConfig.Instance
			}
		}
		if len(sdkServers) > 0 {
			opts = append(opts, control.WithSdkMcpServers(sdkServers))
		}
	}

	return opts
}

// hasSdkMcpServers checks if any SDK MCP servers are configured.
// Returns true if at least one SDK server with a valid Instance exists.
func (t *Transport) hasSdkMcpServers() bool {
	if t.options == nil || len(t.options.McpServers) == 0 {
		return false
	}
	for _, config := range t.options.McpServers {
		if sdkConfig, ok := config.(*shared.McpSdkServerConfig); ok && sdkConfig.Instance != nil {
			return true
		}
	}
	return false
}

// buildEnvironment constructs the environment variables for the subprocess.
func (t *Transport) buildEnvironment() []string {
	env := os.Environ()

	// Set entrypoint to identify SDK to CLI
	env = append(env, "CLAUDE_CODE_ENTRYPOINT="+t.entrypoint)

	// Enable file checkpointing if requested (matches Python SDK)
	if t.options != nil && t.options.EnableFileCheckpointing {
		env = append(env, "CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING=true")
	}

	// Add user-specified environment variables
	if t.options != nil && t.options.ExtraEnv != nil {
		for key, value := range t.options.ExtraEnv {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return env
}

// prepareRuntimeOptions generates temporary runtime files and returns modified options.
func (t *Transport) prepareRuntimeOptions() (*shared.Options, error) {
	opts, err := t.prepareSkillRegistries(t.options)
	if err != nil {
		return nil, err
	}
	return t.prepareMcpConfig(opts)
}

// prepareSkillRegistries creates temporary local plugin wrappers for external
// Skill registries and returns options with the generated plugins and scoped
// Skill tools added.
func (t *Transport) prepareSkillRegistries(options *shared.Options) (*shared.Options, error) {
	if options == nil || len(options.SkillRegistries) == 0 {
		return options, nil
	}

	optsCopy := *options
	optsCopy.Plugins = append([]shared.SdkPluginConfig(nil), options.Plugins...)
	optsCopy.AllowedTools = append([]string(nil), options.AllowedTools...)

	for i, registry := range options.SkillRegistries {
		pluginName := registry.PluginName
		if pluginName == "" {
			pluginName = defaultSkillRegistryPluginName
			if len(options.SkillRegistries) > 1 {
				pluginName = fmt.Sprintf("%s-%d", defaultSkillRegistryPluginName, i+1)
			}
		}

		names, err := resolveSkillRegistryNames(registry)
		if err != nil {
			t.cleanupSkillRegistryDirs()
			return nil, err
		}

		pluginDir, err := t.generateSkillRegistryPlugin(registry.Root, pluginName, names)
		if err != nil {
			t.cleanupSkillRegistryDirs()
			return nil, err
		}

		optsCopy.Plugins = append(optsCopy.Plugins, shared.SdkPluginConfig{
			Type: shared.SdkPluginTypeLocal,
			Path: pluginDir,
		})
		for _, name := range names {
			skillName, err := readSkillName(registry.Root, name)
			if err != nil {
				t.cleanupSkillRegistryDirs()
				return nil, err
			}
			tool := fmt.Sprintf("Skill(%s:%s)", pluginName, skillName)
			if !containsString(optsCopy.AllowedTools, tool) {
				optsCopy.AllowedTools = append(optsCopy.AllowedTools, tool)
			}
		}
	}

	return &optsCopy, nil
}

func resolveSkillRegistryNames(registry shared.SkillRegistryConfig) ([]string, error) {
	if registry.Root == "" {
		return nil, fmt.Errorf("skill registry root is required")
	}

	if len(registry.Names) > 0 {
		names := append([]string(nil), registry.Names...)
		for _, name := range names {
			if err := validateSkillDir(registry.Root, name); err != nil {
				return nil, err
			}
		}
		return names, nil
	}

	entries, err := os.ReadDir(registry.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill registry %s: %w", registry.Root, err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if err := validateSkillDir(registry.Root, name); err == nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func validateSkillDir(root, name string) error {
	if name == "" {
		return fmt.Errorf("skill registry contains an empty skill name")
	}
	skillPath := filepath.Join(root, name)
	info, err := os.Stat(skillPath)
	if err != nil {
		return fmt.Errorf("skill %q not found in registry %s: %w", name, root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("skill %q in registry %s is not a directory", name, root)
	}
	if _, err := os.Stat(filepath.Join(skillPath, "SKILL.md")); err != nil {
		return fmt.Errorf("skill %q in registry %s is missing SKILL.md: %w", name, root, err)
	}
	return nil
}

func readSkillName(root, dirName string) (string, error) {
	file, err := os.Open(filepath.Join(root, dirName, "SKILL.md"))
	if err != nil {
		return "", fmt.Errorf("failed to open SKILL.md for skill %q: %w", dirName, err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	inFrontmatter := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if !inFrontmatter {
			continue
		}
		if value, ok := strings.CutPrefix(line, "name:"); ok {
			name := strings.TrimSpace(value)
			name = strings.Trim(name, `"'`)
			if name != "" {
				return name, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read SKILL.md for skill %q: %w", dirName, err)
	}
	return dirName, nil
}

func (t *Transport) generateSkillRegistryPlugin(root, pluginName string, names []string) (string, error) {
	pluginDir, err := os.MkdirTemp("", "claude_skill_registry_*")
	if err != nil {
		return "", fmt.Errorf("failed to create skill registry plugin dir: %w", err)
	}
	t.skillRegistryDirs = append(t.skillRegistryDirs, pluginDir)

	if err := os.MkdirAll(filepath.Join(pluginDir, ".claude-plugin"), 0o755); err != nil {
		return "", fmt.Errorf("failed to create plugin manifest dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginDir, "skills"), 0o755); err != nil {
		return "", fmt.Errorf("failed to create plugin skills dir: %w", err)
	}

	manifest := map[string]any{
		"name":        pluginName,
		"version":     "0.0.0",
		"description": "Temporary skill registry wrapper generated by claude-agent-sdk-go",
		"skills":      []string{"./skills/"},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal skill registry plugin manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, ".claude-plugin", "plugin.json"), data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write skill registry plugin manifest: %w", err)
	}

	for _, name := range names {
		source := filepath.Join(root, name)
		target := filepath.Join(pluginDir, "skills", name)
		if err := linkSkillDir(source, target); err != nil {
			return "", err
		}
	}

	return pluginDir, nil
}

func linkSkillDir(source, target string) error {
	if runtime.GOOS == windowsOS {
		return copySkillDir(source, target)
	}
	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("failed to symlink skill %s to %s: %w", source, target, err)
	}
	return nil
}

func copySkillDir(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, info.Mode())
	})
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// prepareMcpConfig generates MCP config file if needed and returns modified options.
// Returns the original options unchanged if no MCP servers are configured.
func (t *Transport) prepareMcpConfig(options *shared.Options) (*shared.Options, error) {
	if options == nil || len(options.McpServers) == 0 {
		return options, nil
	}

	mcpConfigPath, err := t.generateMcpConfigFile(options)
	if err != nil {
		return nil, fmt.Errorf("failed to generate MCP config file: %w", err)
	}

	// Create shallow copy with mcp-config in ExtraArgs
	optsCopy := *options
	if optsCopy.ExtraArgs == nil {
		optsCopy.ExtraArgs = make(map[string]*string)
	} else {
		extraArgsCopy := make(map[string]*string, len(optsCopy.ExtraArgs)+1)
		for k, v := range optsCopy.ExtraArgs {
			extraArgsCopy[k] = v
		}
		optsCopy.ExtraArgs = extraArgsCopy
	}
	optsCopy.ExtraArgs["mcp-config"] = &mcpConfigPath
	return &optsCopy, nil
}

// emitCLIVersionWarning performs a non-blocking CLI version check and emits
// a warning via StderrCallback if the CLI version is outdated.
func (t *Transport) emitCLIVersionWarning(ctx context.Context) {
	if warning := cli.CheckCLIVersion(ctx, t.cliPath); warning != "" {
		if t.options != nil && t.options.StderrCallback != nil {
			t.options.StderrCallback(warning)
		}
	}
}
