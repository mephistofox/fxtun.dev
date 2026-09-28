//go:build linux

package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func notSentLowat(t *testing.T, conn net.Conn) int {
	t.Helper()
	raw, err := conn.(*net.TCPConn).SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var v int
	var gerr error
	_ = raw.Control(func(fd uintptr) {
		v, gerr = unix.GetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_NOTSENT_LOWAT)
	})
	if gerr != nil {
		t.Fatal(gerr)
	}
	return v
}

func TestTuneSessionConnSetsNotSentLowat(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	tuneSessionConn(conn)

	if got := notSentLowat(t, conn); got != 128<<10 {
		t.Fatalf("TCP_NOTSENT_LOWAT = %d, want %d", got, 128<<10)
	}
}

// The TLS control listener hands out *tls.Conn, so the socket underneath must
// be tuned before the TLS wrapper hides it.
func TestListenSessionTLSTunesRawSocket(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}

	l, err := listenSessionTLS("127.0.0.1:0", cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	client, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if got := notSentLowat(t, conn.(*tls.Conn).NetConn()); got != 128<<10 {
		t.Fatalf("TCP_NOTSENT_LOWAT = %d, want %d", got, 128<<10)
	}
}
