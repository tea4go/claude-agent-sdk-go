package subprocess

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func (t *Transport) prepareSkillRegistries(options *shared.Options) (*shared.Options, error) {
	if options == nil || len(options.SkillRegistries) == 0 {
		return options, nil
	}

	optsCopy := *options
	optsCopy.Plugins = append([]shared.SdkPluginConfig(nil), options.Plugins...)
	optsCopy.AllowedTools = append([]string(nil), options.AllowedTools...)
	shouldExtendAllowedTools := shouldExtendSkillAllowedTools(options)

	for i, registry := range options.SkillRegistries {
		pluginName := registry.PluginName
		if pluginName == "" {
			pluginName = shared.DefaultSkillRegistryPluginName
			if len(options.SkillRegistries) > 1 {
				pluginName = fmt.Sprintf("%s-%d", shared.DefaultSkillRegistryPluginName, i+1)
			}
		}

		names, err := resolveSkillRegistryNames(registry)
		if err != nil {
			t.cleanupSkillRegistryDirs()
			return nil, err
		}

		skillNames := make(map[string]string, len(names))
		runtimeNames := make(map[string]string, len(names))
		for _, name := range names {
			skillName, err := readSkillName(registry.Root, name)
			if err != nil {
				t.cleanupSkillRegistryDirs()
				return nil, err
			}
			skillNames[name] = skillName

			runtimeName := shared.CanonicalSkillName(skillName)
			if previousDir, ok := runtimeNames[runtimeName]; ok && previousDir != name {
				t.cleanupSkillRegistryDirs()
				return nil, fmt.Errorf(
					"skills %q and %q in registry %s resolve to the same Claude Code runtime name %q",
					previousDir,
					name,
					registry.Root,
					runtimeName,
				)
			}
			runtimeNames[runtimeName] = name
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
		if shouldExtendAllowedTools {
			for _, name := range names {
				tool := fmt.Sprintf("Skill(%s)", shared.SkillRegistryScopedName(pluginName, skillNames[name]))
				if !containsString(optsCopy.AllowedTools, tool) {
					optsCopy.AllowedTools = append(optsCopy.AllowedTools, tool)
				}
			}
		}
	}

	return &optsCopy, nil
}

func shouldExtendSkillAllowedTools(options *shared.Options) bool {
	return len(options.AllowedTools) > 0 || options.Skills != nil
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

func (t *Transport) cleanupSkillRegistryDirs() {
	for _, dir := range t.skillRegistryDirs {
		_ = os.RemoveAll(dir)
	}
	t.skillRegistryDirs = nil
}
