package shared

import "strings"

// DefaultSkillRegistryPluginName is the plugin name used for a generated
// external Skill registry when no explicit plugin name is configured.
const DefaultSkillRegistryPluginName = "sdk-skill-registry"

// CanonicalSkillName returns the runtime name Claude Code assigns to a plugin
// Skill. Claude Code trims the declared Skill name and replaces every
// character outside ASCII letters, digits, underscores, and hyphens with a
// hyphen.
func CanonicalSkillName(name string) string {
	name = strings.TrimSpace(name)

	var builder strings.Builder
	builder.Grow(len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '_' || r == '-':
			builder.WriteRune(r)
		default:
			builder.WriteByte('-')
		}
	}
	return builder.String()
}

// SkillRegistryScopedName returns the scoped runtime name used to invoke a
// Skill exposed through a generated registry plugin.
func SkillRegistryScopedName(pluginName, skillName string) string {
	if pluginName == "" {
		pluginName = DefaultSkillRegistryPluginName
	}
	return pluginName + ":" + CanonicalSkillName(skillName)
}
