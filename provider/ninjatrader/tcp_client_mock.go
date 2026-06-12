// Package ninjatrader — in-process mock TCP client mirroring mock_nt.go.
//
// MockTCPClient simulates the C# AddOn's wire behaviour: dials the Go-side
// TCP server, reads signal frames, emits fill frames after a configurable
// delay, responds to heartbeats with ack frames. For tests and the
// `nq_smoke tcp` sub-command — never for production.
//
// Pattern parallels StartMockNT (mock_nt.go) for the CSV path: goroutine
// with start/stop, signal queue (received-signal channel), fill emission.
package ninjatrader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

// MockTCPClient is the in-process mock NT AddOn used by tests.
type MockTCPClient struct {
	addr      string
	fillDelay time.Duration
	logger    *slog.Logger

	connMu sync.Mutex
	conn   net.Conn

	// SignalsReceived is published to by the read loop for test assertions.
	signalsReceived chan SignalPayload

	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Test hook: if non-nil, called instead of default fill behaviour.
	// Allows tests to verify the connection is alive without auto-filling.
	OnSignal func(SignalPayload) *FillPayload
}

// NewMockTCPClient constructs a mock that dials addr and emits a synthetic
// fill after fillDelay for every signal received (default behaviour).
func NewMockTCPClient(addr string, fillDelay time.Duration) *MockTCPClient {
	return &MockTCPClient{
		addr:            addr,
		fillDelay:       fillDelay,
		logger:          slog.Default(),
		signalsReceived: make(chan SignalPayload, 16),
	}
}

// Start dials the server and spins up the read loop. Blocks until the
// initial connection succeeds (or ctx fires). Returns an error if the
// initial dial fails.
func (m *MockTCPClient) Start(ctx context.Context) error {
	cctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	c, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(cctx, "tcp", m.addr)
	if err != nil {
		return fmt.Errorf("mock_tcp_client: dial %s: %w", m.addr, err)
	}
	m.connMu.Lock()
	m.conn = c
	m.connMu.Unlock()
	m.wg.Add(1)
	go m.readLoop(cctx, c)
	return nil
}

// Stop cancels the read loop, closes the connection, and waits for the
// goroutine to exit. Safe to call once.
func (m *MockTCPClient) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.connMu.Lock()
	if m.conn != nil {
		_ = m.conn.Close()
		m.conn = nil
	}
	m.connMu.Unlock()
	m.wg.Wait()
}

// IsConnected reports whether the mock currently has an open socket.
func (m *MockTCPClient) IsConnected() bool {
	m.connMu.Lock()
	defer m.connMu.Unlock()
	return m.conn != nil
}

// SignalsReceived returns the channel that publishes every signal frame the
// mock has decoded. Tests can drain this to assert delivery semantics.
func (m *MockTCPClient) SignalsReceived() <-chan SignalPayload { return m.signalsReceived }

// SendFill manually emits a fill frame. Tests use this when they want to
// control fill timing explicitly (override the default fillDelay path).
func (m *MockTCPClient) SendFill(fill FillPayload) error {
	m.connMu.Lock()
	c := m.conn
	m.connMu.Unlock()
	if c == nil {
		return errors.New("mock_tcp_client: not connected")
	}
	return WriteFrame(c, FrameFill, fill)
}

// SendAck manually emits an ack frame (heartbeat or signal_id).
func (m *MockTCPClient) SendAck(target string) error {
	m.connMu.Lock()
	c := m.conn
	m.connMu.Unlock()
	if c == nil {
		return errors.New("mock_tcp_client: not connected")
	}
	return WriteFrame(c, FrameAck, AckPayload{Acks: target})
}

// sendFrameForTest emits an arbitrary frame from the mock client side.
// Used by Plan 4.4 Stage 2 integration tests
// (tcp_server_bars_test.go) to inject bars_historical / bar_update
// frames into a real TCPServer for end-to-end bar receive-path
// validation. Production callers should use the typed SendFill /
// SendAck helpers above.
func (m *MockTCPClient) sendFrameForTest(frameType FrameType, payload any) error {
	m.connMu.Lock()
	c := m.conn
	m.connMu.Unlock()
	if c == nil {
		return errors.New("mock_tcp_client: not connected")
	}
	return WriteFrame(c, frameType, payload)
}

func (m *MockTCPClient) readLoop(ctx context.Context, c net.Conn) {
	defer m.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		env, err := ReadFrame(c)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			m.logger.Warn("mock_tcp_client: read error", "err", err)
			return
		}

		switch env.Type {
		case FrameSignal:
			var sig SignalPayload
			if err := json.Unmarshal(env.Payload, &sig); err != nil {
				m.logger.Warn("mock_tcp_client: bad signal payload", "err", err)
				continue
			}
			select {
			case m.signalsReceived <- sig:
			default:
			}
			// Auto-ack the signal so the server's heartbeat ack timer
			// is happy (the spec treats ack as bidirectional, L4410).
			_ = WriteFrame(c, FrameAck, AckPayload{Acks: sig.SignalID})
			// Then emit a fill after fillDelay.
			m.scheduleFill(ctx, c, sig)

		case FrameHeartbeat:
			_ = WriteFrame(c, FrameAck, AckPayload{Acks: "heartbeat"})

		case FrameAck:
			// Server-side ack of our own heartbeats (if we sent any).

		case FrameFill:
			// Mock is the fill emitter, not the consumer.

		default:
			m.logger.Warn("mock_tcp_client: unknown frame type", "type", env.Type)
		}
	}
}

func (m *MockTCPClient) scheduleFill(ctx context.Context, c net.Conn, sig SignalPayload) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		// Allow tests to override fill behaviour entirely.
		if m.OnSignal != nil {
			fill := m.OnSignal(sig)
			if fill == nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(m.fillDelay):
			}
			_ = WriteFrame(c, FrameFill, *fill)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(m.fillDelay):
		}
		fill := FillPayload{
			SignalID:      sig.SignalID,
			FillPrice:     sig.Entry, // no slippage modelled in default mock
			FillTime:      time.Now().UTC().Format(time.RFC3339),
			Side:          sig.Side,
			Quantity:      sig.Quantity,
			SlippageTicks: 0,
			Status:        "filled",
		}
		if err := WriteFrame(c, FrameFill, fill); err != nil {
			m.logger.Warn("mock_tcp_client: write fill", "err", err)
		}
	}()
}
