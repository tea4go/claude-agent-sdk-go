package subprocess

import (
	"context"
	"io"
	"sync"
)

// ProtocolAdapter adapts subprocess stdin/stdout for use with control.Protocol.
// It implements the control.Transport interface to enable the control protocol
// to send requests via subprocess stdin.
//
// Note: The Read() method returns a closed channel because we don't use the
// protocol's built-in readLoop. Instead, subprocess.Transport routes control
// messages directly to protocol.HandleIncomingMessage() from handleStdout().
//
// ProtocolAdapter 将子进程的 stdin/stdout 适配给 control.Protocol 使用。
// 它实现 control.Transport 接口，使控制协议能通过子进程 stdin 发送请求。
//
// 注意：Read() 方法返回一个已关闭的通道，因为我们不使用协议内置的
// readLoop；而是由 subprocess.Transport 在 handleStdout() 中将控制消息
// 直接路由给 protocol.HandleIncomingMessage()。
type ProtocolAdapter struct {
	stdin    io.Writer
	mu       sync.Mutex
	closed   bool
	readChan chan []byte
}

// NewProtocolAdapter creates a new adapter that wraps subprocess stdin for
// the control protocol.
//
// NewProtocolAdapter 创建一个将子进程 stdin 包装给控制协议使用的新适配器。
func NewProtocolAdapter(stdin io.Writer) *ProtocolAdapter {
	// Create a closed channel for Read() - we handle message routing externally
	readChan := make(chan []byte)
	close(readChan)

	return &ProtocolAdapter{
		stdin:    stdin,
		readChan: readChan,
	}
}

// Write sends data to the subprocess stdin.
// This is called by Protocol.SendControlRequest() to send control requests.
//
// Write 将数据写入子进程 stdin，由 Protocol.SendControlRequest() 调用以发送控制请求。
func (pa *ProtocolAdapter) Write(ctx context.Context, data []byte) error {
	// Check context before proceeding
	if ctx.Err() != nil {
		return ctx.Err()
	}

	pa.mu.Lock()
	defer pa.mu.Unlock()

	if pa.closed {
		return io.ErrClosedPipe
	}

	if pa.stdin == nil {
		return io.ErrClosedPipe
	}

	_, err := pa.stdin.Write(data)
	return err
}

// Read returns a channel for reading data from the subprocess.
// This channel is pre-closed because we don't use the protocol's built-in
// readLoop - instead, subprocess.Transport routes control messages directly
// to protocol.HandleIncomingMessage() from handleStdout().
//
// Read 返回用于从子进程读取数据的通道。该通道预先已关闭，因为我们
// 不使用协议内置的 readLoop——而是由 subprocess.Transport 在 handleStdout()
// 中将控制消息直接路由给 protocol.HandleIncomingMessage()。
func (pa *ProtocolAdapter) Read(_ context.Context) <-chan []byte {
	return pa.readChan
}

// Close closes the adapter.
// Note: This does NOT close the underlying stdin - that's managed by Transport.
//
// Close 关闭适配器。注意：此操作不会关闭底层 stdin——那由 Transport 管理。
func (pa *ProtocolAdapter) Close() error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.closed = true
	// Don't close stdin here - Transport manages that
	return nil
}
