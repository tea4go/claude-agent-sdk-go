package claudecode

import (
	"github.com/tea4go/claude-agent-sdk-go/internal/control"
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

type Usage = shared.Usage
type RawMessage = shared.RawMessage
type SlashCommand = control.SlashCommand

type AbortableTransport interface{ Abort() error }
