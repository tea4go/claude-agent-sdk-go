package shared

import "encoding/json"

const (
	StopReasonEndTurn      = "end_turn"
	StopReasonToolUse      = "tool_use"
	StopReasonStopSequence = "stop_sequence"
	StopReasonMaxTokens    = "max_tokens"
)

// RawMessage preserves a future CLI event that this SDK does not yet know.
type RawMessage struct {
	MessageType string         `json:"type"`
	Data        map[string]any `json:"-"`
}

func (m *RawMessage) Type() string { return m.MessageType }

func (m *RawMessage) MarshalJSON() ([]byte, error) {
	data := make(map[string]any, len(m.Data)+1)
	for key, value := range m.Data {
		data[key] = value
	}
	data["type"] = m.MessageType
	return json.Marshal(data)
}
