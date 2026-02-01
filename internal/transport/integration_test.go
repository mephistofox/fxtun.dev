package transport

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQUICMultiStreamParallel(t *testing.T) {
	tlsCfg := generateTestTLSConfig()

	ln, err := NewQUICListener(":0", tlsCfg, DefaultQUICConfig())
	require.NoError(t, err)
	defer ln.Close()

	ctx := context.Background()
	const numStreams = 50

	var serverWg sync.WaitGroup
	serverWg.Add(1)
	received := make(chan string, numStreams)

	go func() {
		defer serverWg.Done()
		session, err := ln.Accept(ctx)
		if err != nil {
			return
		}
		defer session.Close()

		var streamWg sync.WaitGroup
		for i := 0; i < numStreams; i++ {
			stream, err := session.AcceptStream(ctx)
			if err != nil {
				return
			}
			streamWg.Add(1)
			go func() {
				defer streamWg.Done()
				defer stream.Close()
				buf, _ := io.ReadAll(stream)
				received <- string(buf)
			}()
		}
		streamWg.Wait()
	}()

	clientTLS := generateTestTLSConfig()
	clientTLS.InsecureSkipVerify = true
	clientSession, err := DialQUIC(ctx, ln.Addr(), clientTLS, DefaultQUICConfig())
	require.NoError(t, err)
	defer clientSession.Close()

	var clientWg sync.WaitGroup
	for i := 0; i < numStreams; i++ {
		clientWg.Add(1)
		go func(id int) {
			defer clientWg.Done()
			stream, err := clientSession.OpenStream(ctx)
			require.NoError(t, err)
			msg := fmt.Sprintf("stream-%d", id)
			_, err = stream.Write([]byte(msg))
			require.NoError(t, err)
			stream.Close()
		}(i)
	}
	clientWg.Wait()

	// Wait for server to finish
	serverWg.Wait()
	close(received)

	msgs := make(map[string]bool)
	for msg := range received {
		msgs[msg] = true
	}
	require.Equal(t, numStreams, len(msgs), "expected %d unique messages", numStreams)
}
