package shared

import "strings"

// DefaultSkillRegistryPluginName is the plugin name used for a generated
// external Skill registry when no explicit plugin name is configured.
//
// DefaultSkillRegistryPluginName 是在未显式配置插件名时，为自动生成的
// 外部 Skill 注册表使用的默认插件名。
const DefaultSkillRegistryPluginName = "sdk-skill-registry"

// CanonicalSkillName returns the runtime name Claude Code assigns to a plugin
// Skill. Claude Code trims the declared Skill name and replaces every
// character outside ASCII letters, digits, underscores, and hyphens with a
// hyphen.
//
// CanonicalSkillName 返回 Claude Code 为插件 Skill 分配的运行时名称。
// Claude Code 会去除声明名称两端的空白，并把所有非 ASCII 字母、数字、
// 下划线和连字符的字符替换为连字符，以保证名称合法。
func CanonicalSkillName(name string) string {
	name = strings.TrimSpace(name)

	var builder strings.Builder
	builder.Grow(len(name))
	// 逐字符扫描：保留合法字符，其余一律替换为连字符。
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
//
// SkillRegistryScopedName 返回用于调用某个通过生成的注册表插件暴露的
// Skill 时所用的带作用域前缀的运行时名称，格式为 "插件名:规范化Skill名"。
// 当 pluginName 为空时回退到默认插件名。
func SkillRegistryScopedName(pluginName, skillName string) string {
	if pluginName == "" {
		pluginName = DefaultSkillRegistryPluginName
	}
	return pluginName + ":" + CanonicalSkillName(skillName)
}
