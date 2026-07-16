package claudecode

import "github.com/tea4go/claude-agent-sdk-go/internal/shared"

// DefaultSkillRegistryPluginName is the plugin name used by WithSkillRegistry
// and WithSkillRegistryAll when no explicit plugin name is configured.
const DefaultSkillRegistryPluginName = shared.DefaultSkillRegistryPluginName

// CanonicalSkillName returns the runtime name Claude Code assigns to a plugin
// Skill. Pass the declared frontmatter name, or the directory name when the
// Skill does not declare a different name.
func CanonicalSkillName(name string) string {
	return shared.CanonicalSkillName(name)
}

// SkillRegistryScopedName returns the scoped runtime name used to explicitly
// invoke a Skill exposed by WithSkillRegistry. Pass an empty pluginName to use
// DefaultSkillRegistryPluginName.
func SkillRegistryScopedName(pluginName, skillName string) string {
	return shared.SkillRegistryScopedName(pluginName, skillName)
}
