package claudecode

import (
	"context"
	"fmt"
)

func (c *ClientImpl) dispatchSkill(ctx context.Context, prompt, sessionID string) bool {
	name, _, ok := parseSkillCommand(prompt)
	if !ok {
		return false
	}
	c.mu.RLock()
	_, exists := getSkillHandler(c.options, name)
	inject, done := c.injectChan, c.mergeDone
	c.mu.RUnlock()
	if !exists || inject == nil {
		return false
	}
	go func() {
		skillCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() {
			select {
			case <-done:
				cancel()
			case <-skillCtx.Done():
			}
		}()
		iter, _ := tryRunSkill(skillCtx, prompt, c.options, sessionID)
		if iter == nil {
			return
		}
		defer iter.Close()
		for {
			msg, err := iter.Next(skillCtx)
			if err != nil {
				return
			}
			select {
			case inject <- msg:
			case <-done:
				return
			case <-skillCtx.Done():
				return
			}
		}
	}()
	return true
}

type orderedTransportEvent struct {
	msg Message
	err error
}

func mergeMessageChannels(ctx context.Context, out chan<- Message, errOut chan<- error, transport *streamReader, injected <-chan Message, done chan<- struct{}) {
	defer close(done)
	defer close(out)
	defer close(errOut)
	events := make(chan orderedTransportEvent, 1)
	go func() {
		defer close(events)
		for {
			msg, err := transport.next(ctx, nil)
			select {
			case events <- orderedTransportEvent{msg: msg, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.err != nil {
				for {
					select {
					case msg := <-injected:
						select {
						case out <- msg:
						case <-ctx.Done():
							return
						}
					default:
						if event.err != ErrNoMoreMessages {
							select {
							case errOut <- event.err:
							case <-ctx.Done():
							}
						}
						return
					}
				}
			}
			select {
			case out <- event.msg:
			case <-ctx.Done():
				return
			}
		case msg := <-injected:
			select {
			case out <- msg:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// Abort force-stops the active CLI process tree and releases the client.
func (c *ClientImpl) Abort() error {
	c.mu.RLock()
	transport := c.transport
	connected := c.connected
	c.mu.RUnlock()
	if !connected {
		return nil
	}
	var abortErr error
	if abortable, ok := transport.(AbortableTransport); ok {
		abortErr = abortable.Abort()
	}
	closeErr := c.Disconnect()
	if abortErr != nil {
		return fmt.Errorf("abort transport: %w", abortErr)
	}
	return closeErr
}

// GetSlashCommands returns commands from the CLI's system/init message.
func (c *ClientImpl) GetSlashCommands(ctx context.Context) ([]SlashCommand, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.RLock()
	transport := c.transport
	connected := c.connected
	c.mu.RUnlock()
	if !connected || transport == nil {
		return nil, fmt.Errorf("client not connected")
	}
	provider, ok := transport.(interface {
		GetSlashCommands(context.Context) ([]SlashCommand, error)
	})
	if !ok {
		return nil, fmt.Errorf("transport does not support slash command discovery")
	}
	return provider.GetSlashCommands(ctx)
}
