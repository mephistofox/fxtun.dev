package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/quic-go/quic-go"
)

// QUICConfig holds configuration parameters for QUIC transport.
type QUICConfig struct {
	MaxIdleTimeout             time.Duration
	KeepAlivePeriod            time.Duration
	MaxStreamReceiveWindow     uint64
	MaxConnectionReceiveWindow uint64
	HandshakeIdleTimeout       time.Duration
	Enable0RTT                 bool
}

// DefaultQUICConfig returns sensible defaults for tunnel workloads.
func DefaultQUICConfig() QUICConfig {
	return QUICConfig{
		MaxIdleTimeout:             30 * time.Second,
		KeepAlivePeriod:            10 * time.Second,
		MaxStreamReceiveWindow:     4 * 1024 * 1024,
		MaxConnectionReceiveWindow: 16 * 1024 * 1024,
		HandshakeIdleTimeout:       10 * time.Second,
		Enable0RTT:                 true,
	}
}

func toQuicConfig(cfg QUICConfig) *quic.Config {
	return &quic.Config{
		MaxIdleTimeout:             cfg.MaxIdleTimeout,
		KeepAlivePeriod:            cfg.KeepAlivePeriod,
		MaxStreamReceiveWindow:     cfg.MaxStreamReceiveWindow,
		MaxConnectionReceiveWindow: cfg.MaxConnectionReceiveWindow,
		HandshakeIdleTimeout:       cfg.HandshakeIdleTimeout,
		Allow0RTT:                  cfg.Enable0RTT,
	}
}

// quicStream wraps a quic.Stream to satisfy the transport.Stream interface.
type quicStream struct {
	*quic.Stream
}

func (s *quicStream) Close() error {
	s.Stream.CancelRead(0)
	return s.Stream.Close()
}

// quicSession wraps a quic.Conn to satisfy the transport.Session interface.
type quicSession struct {
	conn *quic.Conn
}

func (s *quicSession) OpenStream(ctx context.Context) (Stream, error) {
	stream, err := s.conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return &quicStream{stream}, nil
}

func (s *quicSession) AcceptStream(ctx context.Context) (Stream, error) {
	stream, err := s.conn.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}
	return &quicStream{stream}, nil
}

func (s *quicSession) Close() error {
	return s.conn.CloseWithError(0, "")
}

func (s *quicSession) CloseWithError(code uint64, msg string) error {
	return s.conn.CloseWithError(quic.ApplicationErrorCode(code), msg)
}

func (s *quicSession) IsClosed() bool {
	return s.conn.Context().Err() != nil
}

// QUICListener accepts incoming QUIC connections as transport.Session.
type QUICListener struct {
	listener *quic.Listener
	addr     string
}

// NewQUICListener creates a QUIC listener on the given address.
// A non-nil TLS config is required.
func NewQUICListener(addr string, tlsCfg *tls.Config, cfg QUICConfig) (*QUICListener, error) {
	if tlsCfg == nil {
		return nil, fmt.Errorf("TLS config is required for QUIC")
	}
	ln, err := quic.ListenAddr(addr, tlsCfg, toQuicConfig(cfg))
	if err != nil {
		return nil, err
	}
	return &QUICListener{
		listener: ln,
		addr:     ln.Addr().String(),
	}, nil
}

func (l *QUICListener) Accept(ctx context.Context) (Session, error) {
	conn, err := l.listener.Accept(ctx)
	if err != nil {
		return nil, err
	}
	return &quicSession{conn: conn}, nil
}

func (l *QUICListener) Close() error {
	return l.listener.Close()
}

func (l *QUICListener) Addr() string {
	return l.addr
}

// DialQUIC establishes a QUIC connection and returns it as a transport.Session.
func DialQUIC(ctx context.Context, addr string, tlsCfg *tls.Config, cfg QUICConfig) (Session, error) {
	conn, err := quic.DialAddr(ctx, addr, tlsCfg, toQuicConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("quic dial: %w", err)
	}
	return &quicSession{conn: conn}, nil
}
