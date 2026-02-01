package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

type YamuxConfig struct {
	MaxStreamWindowSize    uint32
	KeepAliveInterval      time.Duration
	ConnectionWriteTimeout time.Duration
}

func DefaultYamuxConfig() YamuxConfig {
	return YamuxConfig{
		MaxStreamWindowSize:    4 * 1024 * 1024,
		KeepAliveInterval:      10 * time.Second,
		ConnectionWriteTimeout: 30 * time.Second,
	}
}

type yamuxStream struct {
	net.Conn
}

type yamuxSession struct {
	session *yamux.Session
}

func NewYamuxSession(conn io.ReadWriteCloser, isServer bool) (Session, error) {
	return NewYamuxSessionWithConfig(conn, isServer, DefaultYamuxConfig())
}

func NewYamuxSessionWithConfig(conn io.ReadWriteCloser, isServer bool, cfg YamuxConfig) (Session, error) {
	yamuxCfg := yamux.DefaultConfig()
	yamuxCfg.EnableKeepAlive = true
	yamuxCfg.KeepAliveInterval = cfg.KeepAliveInterval
	yamuxCfg.MaxStreamWindowSize = cfg.MaxStreamWindowSize
	yamuxCfg.ConnectionWriteTimeout = cfg.ConnectionWriteTimeout

	var session *yamux.Session
	var err error
	if isServer {
		session, err = yamux.Server(conn, yamuxCfg)
	} else {
		session, err = yamux.Client(conn, yamuxCfg)
	}
	if err != nil {
		return nil, fmt.Errorf("create yamux session: %w", err)
	}
	return &yamuxSession{session: session}, nil
}

func (s *yamuxSession) OpenStream(ctx context.Context) (Stream, error) {
	stream, err := s.session.Open()
	if err != nil {
		return nil, err
	}
	return &yamuxStream{stream}, nil
}

func (s *yamuxSession) AcceptStream(ctx context.Context) (Stream, error) {
	stream, err := s.session.Accept()
	if err != nil {
		return nil, err
	}
	return &yamuxStream{stream}, nil
}

func (s *yamuxSession) Close() error {
	return s.session.Close()
}

func (s *yamuxSession) CloseWithError(code uint64, msg string) error {
	_ = s.session.GoAway()
	return s.session.Close()
}

func (s *yamuxSession) IsClosed() bool {
	return s.session.IsClosed()
}

type YamuxListener struct {
	listener net.Listener
	cfg      YamuxConfig
}

func NewYamuxListener(addr string, tlsCfg *tls.Config) (*YamuxListener, error) {
	return NewYamuxListenerWithConfig(addr, tlsCfg, DefaultYamuxConfig())
}

func NewYamuxListenerWithConfig(addr string, tlsCfg *tls.Config, cfg YamuxConfig) (*YamuxListener, error) {
	var ln net.Listener
	var err error
	if tlsCfg != nil {
		ln, err = tls.Listen("tcp", addr, tlsCfg)
	} else {
		ln, err = net.Listen("tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	return &YamuxListener{listener: ln, cfg: cfg}, nil
}

func (l *YamuxListener) Accept(ctx context.Context) (Session, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}
	return NewYamuxSessionWithConfig(conn, true, l.cfg)
}

func (l *YamuxListener) Close() error {
	return l.listener.Close()
}

func (l *YamuxListener) Addr() string {
	return l.listener.Addr().String()
}
