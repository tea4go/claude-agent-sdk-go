package control

// SlashCommand describes a CLI command available in a session.
type SlashCommand struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
