package subprocess

import "github.com/tea4go/claude-agent-sdk-go/internal/shared"

func (t *Transport) prepareRuntimeOptions() (*shared.Options, error) {
	opts, err := t.prepareSkillRegistries(t.options)
	if err != nil {
		return nil, err
	}
	t.options = opts
	return t.prepareMcpConfig()
}
