package subprocess

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tea4go/claude-agent-sdk-go/internal/cli"
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

func TestPrepareSkillRegistriesCreatesPluginWrapper(t *testing.T) {
	registryRoot := t.TempDir()
	createTestSkill(t, registryRoot, "compress")
	createTestSkill(t, registryRoot, "validate-json")

	transport := New("echo", &shared.Options{
		SkillRegistries: []shared.SkillRegistryConfig{
			{
				Root:  registryRoot,
				Names: []string{"compress", "validate-json"},
			},
		},
	}, true, "sdk-go")

	opts, err := transport.prepareSkillRegistries(transport.options)
	if err != nil {
		t.Fatalf("prepareSkillRegistries error: %v", err)
	}

	if len(opts.Plugins) != 1 {
		t.Fatalf("expected 1 generated plugin, got %d", len(opts.Plugins))
	}
	pluginDir := opts.Plugins[0].Path
	assertGeneratedPluginManifest(t, pluginDir, "sdk-skill-registry")
	assertSkillLink(t, pluginDir, registryRoot, "compress")
	assertSkillLink(t, pluginDir, registryRoot, "validate-json")
	if len(opts.AllowedTools) != 0 {
		t.Fatalf("expected registry-only configuration not to restrict allowed tools, got %v", opts.AllowedTools)
	}

	if len(transport.skillRegistryDirs) != 1 {
		t.Fatalf("expected generated plugin dir to be tracked for cleanup, got %d", len(transport.skillRegistryDirs))
	}

	transport.cleanup()
	if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
		t.Fatalf("expected generated plugin dir to be removed, stat err=%v", err)
	}
}

func TestPrepareSkillRegistriesUsesFrontmatterSkillNameForAllowedTool(t *testing.T) {
	registryRoot := t.TempDir()
	createTestSkillWithFrontmatterName(t, registryRoot, "zym-skills", "find-skills")

	transport := New("echo", &shared.Options{
		AllowedTools: []string{"Read"},
		SkillRegistries: []shared.SkillRegistryConfig{
			{
				Root:  registryRoot,
				Names: []string{"zym-skills"},
			},
		},
	}, true, "sdk-go")

	opts, err := transport.prepareSkillRegistries(transport.options)
	if err != nil {
		t.Fatalf("prepareSkillRegistries error: %v", err)
	}

	pluginDir := opts.Plugins[0].Path
	assertSkillLink(t, pluginDir, registryRoot, "zym-skills")
	assertContainsString(t, opts.AllowedTools, "Skill(sdk-skill-registry:find-skills)")
	assertNotContainsString(t, opts.AllowedTools, "Skill(sdk-skill-registry:zym-skills)")
}

func TestPrepareSkillRegistriesPreservesExistingSkillAllowlist(t *testing.T) {
	registryRoot := t.TempDir()
	createTestSkill(t, registryRoot, "compress")

	transport := New("echo", &shared.Options{
		AllowedTools: []string{"Read", "Skill(project-review)", "Skill"},
		SkillRegistries: []shared.SkillRegistryConfig{
			{
				Root:  registryRoot,
				Names: []string{"compress"},
			},
		},
	}, true, "sdk-go")

	opts, err := transport.prepareSkillRegistries(transport.options)
	if err != nil {
		t.Fatalf("prepareSkillRegistries error: %v", err)
	}

	assertContainsString(t, opts.AllowedTools, "Read")
	assertContainsString(t, opts.AllowedTools, "Skill")
	assertContainsString(t, opts.AllowedTools, "Skill(project-review)")
	assertContainsString(t, opts.AllowedTools, "Skill(sdk-skill-registry:compress)")
}

func TestRuntimeOptionsCombineRegistryAndConfiguredSkills(t *testing.T) {
	registryRoot := t.TempDir()
	createTestSkill(t, registryRoot, "compress")

	transport := New("echo", &shared.Options{
		Skills: []string{"project-review"},
		SkillRegistries: []shared.SkillRegistryConfig{
			{
				Root:  registryRoot,
				Names: []string{"compress"},
			},
		},
	}, true, "sdk-go")

	opts, err := transport.prepareRuntimeOptions()
	if err != nil {
		t.Fatalf("prepareRuntimeOptions error: %v", err)
	}
	defer transport.cleanup()

	cmd := cli.BuildCommand("claude", opts, true)
	assertContainsArgs(t, cmd, "--allowed-tools", "Skill(sdk-skill-registry:compress),Skill(project-review)")
	assertContainsArgs(t, cmd, "--setting-sources", "user,project")
}

func TestRuntimeOptionsDoNotRestrictSkillsWhenOnlyRegistryConfigured(t *testing.T) {
	registryRoot := t.TempDir()
	createTestSkill(t, registryRoot, "compress")

	transport := New("echo", &shared.Options{
		SkillRegistries: []shared.SkillRegistryConfig{
			{
				Root:  registryRoot,
				Names: []string{"compress"},
			},
		},
	}, true, "sdk-go")

	opts, err := transport.prepareRuntimeOptions()
	if err != nil {
		t.Fatalf("prepareRuntimeOptions error: %v", err)
	}
	defer transport.cleanup()

	cmd := cli.BuildCommand("claude", opts, true)
	assertNotContainsArg(t, cmd, "--allowed-tools")
}

func TestPrepareSkillRegistriesAllDiscoversSkillDirs(t *testing.T) {
	registryRoot := t.TempDir()
	createTestSkill(t, registryRoot, "alpha")
	createTestSkill(t, registryRoot, "beta")
	if err := os.Mkdir(filepath.Join(registryRoot, "notes"), 0o755); err != nil {
		t.Fatalf("mkdir notes: %v", err)
	}

	transport := New("echo", &shared.Options{
		SkillRegistries: []shared.SkillRegistryConfig{
			{
				Root:  registryRoot,
				Names: []string{},
			},
		},
	}, true, "sdk-go")

	opts, err := transport.prepareSkillRegistries(transport.options)
	if err != nil {
		t.Fatalf("prepareSkillRegistries error: %v", err)
	}

	pluginDir := opts.Plugins[0].Path
	assertSkillLink(t, pluginDir, registryRoot, "alpha")
	assertSkillLink(t, pluginDir, registryRoot, "beta")
	if _, err := os.Lstat(filepath.Join(pluginDir, "skills", "notes")); !os.IsNotExist(err) {
		t.Fatalf("expected non-skill directory to be ignored, stat err=%v", err)
	}
	if len(opts.AllowedTools) != 0 {
		t.Fatalf("expected registry-all configuration not to restrict allowed tools, got %v", opts.AllowedTools)
	}
}

func TestPrepareSkillRegistriesRejectsMissingSkill(t *testing.T) {
	registryRoot := t.TempDir()

	transport := New("echo", &shared.Options{
		SkillRegistries: []shared.SkillRegistryConfig{
			{
				Root:  registryRoot,
				Names: []string{"missing"},
			},
		},
	}, true, "sdk-go")

	_, err := transport.prepareSkillRegistries(transport.options)
	if err == nil {
		t.Fatal("expected missing skill error")
	}
}

func createTestSkill(t *testing.T, root, name string) {
	t.Helper()
	createTestSkillWithFrontmatterName(t, root, name, name)
}

func createTestSkillWithFrontmatterName(t *testing.T, root, dirName, skillName string) {
	t.Helper()
	dir := filepath.Join(root, dirName)
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+skillName+"\ndescription: test\n---\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scripts", "main.ts"), []byte("console.log('ok')\n"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}
}

func assertGeneratedPluginManifest(t *testing.T, pluginDir, name string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatalf("read plugin manifest: %v", err)
	}

	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("unmarshal plugin manifest: %v", err)
	}
	if manifest["name"] != name {
		t.Fatalf("expected plugin name %q, got %v", name, manifest["name"])
	}
}

func assertSkillLink(t *testing.T, pluginDir, registryRoot, name string) {
	t.Helper()
	linkPath := filepath.Join(pluginDir, "skills", name)
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat skill link %s: %v", name, err)
	}

	target := filepath.Join(registryRoot, name)
	if runtime.GOOS == windowsOS {
		if !info.IsDir() {
			t.Fatalf("expected copied skill directory on Windows")
		}
		return
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink", linkPath)
	}
	gotTarget, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if gotTarget != target {
		t.Fatalf("expected link target %q, got %q", target, gotTarget)
	}
}

func assertContainsString(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %v to contain %q", values, want)
}

func assertContainsArgs(t *testing.T, args []string, flag, value string) {
	t.Helper()
	for i, arg := range args {
		if arg == flag && i+1 < len(args) && args[i+1] == value {
			return
		}
	}
	t.Fatalf("expected command to contain %s %s, got %v", flag, value, args)
}

func assertNotContainsArg(t *testing.T, args []string, unwanted string) {
	t.Helper()
	for _, arg := range args {
		if arg == unwanted {
			t.Fatalf("expected command not to contain %s, got %v", unwanted, args)
		}
	}
}

func assertNotContainsString(t *testing.T, values []string, unwanted string) {
	t.Helper()
	for _, value := range values {
		if value == unwanted {
			t.Fatalf("expected %v not to contain %q", values, unwanted)
		}
	}
}
