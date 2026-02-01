package transport

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func generateTestTLSConfig() *tls.Config {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)

	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(&x509.Certificate{Raw: certDER})

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, //nolint:gosec // test-only self-signed cert
		RootCAs:            certPool,
		NextProtos:         []string{"fxtunnel"},
	}
}

func TestQUICTransport(t *testing.T) {
	tlsCfg := generateTestTLSConfig()

	ln, err := NewQUICListener(":0", tlsCfg, DefaultQUICConfig())
	require.NoError(t, err)
	defer ln.Close()

	ctx := context.Background()

	sessionCh := make(chan Session, 1)
	go func() {
		s, err := ln.Accept(ctx)
		require.NoError(t, err)
		sessionCh <- s
	}()

	clientTLS := &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // test-only self-signed cert
		NextProtos:         []string{"fxtunnel"},
	}
	clientSession, err := DialQUIC(ctx, ln.Addr(), clientTLS, DefaultQUICConfig())
	require.NoError(t, err)
	defer clientSession.Close()

	serverSession := <-sessionCh
	defer serverSession.Close()

	go func() {
		stream, err := clientSession.OpenStream(ctx)
		require.NoError(t, err)
		_, err = stream.Write([]byte("quic-hello"))
		require.NoError(t, err)
		stream.Close()
	}()

	stream, err := serverSession.AcceptStream(ctx)
	require.NoError(t, err)

	buf, err := io.ReadAll(stream)
	require.NoError(t, err)
	require.Equal(t, "quic-hello", string(buf))
}
