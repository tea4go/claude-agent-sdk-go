package claudecode

import "github.com/tea4go/claude-agent-sdk-go/internal/shared"

// DefaultSkillRegistryPluginName is the plugin name used by WithSkillRegistry
// and WithSkillRegistryAll when no explicit plugin name is configured.
//
// DefaultSkillRegistryPluginName 是 WithSkillRegistry 与 WithSkillRegistryAll
// 在未显式配置插件名时使用的插件名。
const DefaultSkillRegistryPluginName = shared.DefaultSkillRegistryPluginName

// CanonicalSkillName returns the runtime name Claude Code assigns to a plugin
// Skill. Pass the declared frontmatter name, or the directory name when the
// Skill does not declare a different name.
//
// CanonicalSkillName 返回 Claude Code 为插件技能分配的运行时名称。
// 传入 frontmatter 中声明的名称；若技能未声明其他名称则传入目录名。
func CanonicalSkillName(name string) string {
	return shared.CanonicalSkillName(name)
}

// SkillRegistryScopedName returns the scoped runtime name used to explicitly
// invoke a Skill exposed by WithSkillRegistry. Pass an empty pluginName to use
// DefaultSkillRegistryPluginName.
//
// SkillRegistryScopedName 返回用于显式调用 WithSkillRegistry 所暴露技能的带作用域运行时名称。
// pluginName 传空时使用 DefaultSkillRegistryPluginName。
func SkillRegistryScopedName(pluginName, skillName string) string {
	return shared.SkillRegistryScopedName(pluginName, skillName)
}
