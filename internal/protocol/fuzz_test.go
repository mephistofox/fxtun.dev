package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"testing"
	"time"
	"unicode/utf8"
)

// FuzzDecode feeds arbitrary bytes into the length-prefixed decoder.
func FuzzDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x00, 0x00, 0x02, '{', '}'})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Add([]byte{0x00, 0x10, 0x00, 0x00, '{'})
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		c := NewCodec(bytes.NewReader(data), io.Discard)
		var msg map[string]any
		if err := c.Decode(&msg); err != nil {
			return
		}
		// A successful decode must have consumed a well-formed frame.
		if len(data) < HeaderSize {
			t.Fatalf("decoded from a short buffer: %q", data)
		}
		if l := binary.BigEndian.Uint32(data[:HeaderSize]); l > MaxMessageSize {
			t.Fatalf("decoded an oversized frame: %d", l)
		}
	})
}

// FuzzDecodeRaw exercises the raw-frame path used by the control loop.
func FuzzDecodeRaw(f *testing.F) {
	f.Add([]byte{0x00, 0x00, 0x00, 0x0f, '{', '"', 't', 'y', 'p', 'e', '"', ':', '"', 'p', 'i', 'n', 'g', '"', '}'})
	f.Add([]byte{0x7f, 0xff, 0xff, 0xff, 'x'})

	f.Fuzz(func(t *testing.T, data []byte) {
		c := NewCodec(bytes.NewReader(data), io.Discard)
		raw, msg, err := c.DecodeRaw()
		if err != nil {
			return
		}
		if msg == nil {
			t.Fatal("nil message with nil error")
		}
		if len(raw) > MaxMessageSize {
			t.Fatalf("raw payload above the cap: %d", len(raw))
		}
		// Whatever type came off the wire must not crash the dispatcher.
		_, _ = ParseMessage(raw, msg.Type)
	})
}

// FuzzParseMessage drives the type dispatcher with arbitrary JSON and types.
func FuzzParseMessage(f *testing.F) {
	f.Add("auth", []byte(`{"type":"auth","token":"x"}`))
	f.Add("tunnel_request", []byte(`{"tunnel_type":"http","local_port":-1,"remote_port":99999999999}`))
	f.Add("new_connection", []byte(`{"remote_addr":"\ud800"}`))
	f.Add("", []byte(`null`))
	f.Add("auth_result", []byte(`{"capabilities":{"max_body_size":-9223372036854775808}}`))

	f.Fuzz(func(t *testing.T, typ string, data []byte) {
		if len(data) > MaxMessageSize {
			t.Skip()
		}
		msg, err := ParseMessage(data, MessageType(typ))
		if err != nil {
			return
		}
		// Anything the dispatcher accepts must survive a re-encode: the server
		// echoes parsed messages back to peers.
		if _, err := json.Marshal(msg); err != nil {
			t.Fatalf("parsed %q is not re-encodable: %v", typ, err)
		}
	})
}

// FuzzReadStreamHeader feeds arbitrary bytes into the per-stream binary header.
func FuzzReadStreamHeader(f *testing.F) {
	f.Add([]byte{0x00, 0x00})
	f.Add([]byte{0xff})
	f.Add([]byte{0x02, 'i', 'd', 0x03, '1', '.', '2'})

	f.Fuzz(func(t *testing.T, data []byte) {
		h, err := ReadStreamHeader(bytes.NewReader(data))
		if err != nil {
			return
		}
		if len(h.TunnelID) > 255 || len(h.RemoteAddr) > 255 {
			t.Fatalf("header field above the wire limit: %d/%d", len(h.TunnelID), len(h.RemoteAddr))
		}
		// Round-trip: what we read must re-encode to the same bytes.
		var buf bytes.Buffer
		if err := WriteStreamHeader(&buf, h.TunnelID, h.RemoteAddr); err != nil {
			t.Fatalf("re-encode failed: %v", err)
		}
		h2, err := ReadStreamHeader(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatalf("re-decode failed: %v", err)
		}
		if h2.TunnelID != h.TunnelID || h2.RemoteAddr != h.RemoteAddr {
			t.Fatalf("round-trip mismatch: %q/%q vs %q/%q", h.TunnelID, h.RemoteAddr, h2.TunnelID, h2.RemoteAddr)
		}
	})
}

// FuzzEncodeDecodeRoundTrip checks that anything we emit we can read back.
func FuzzEncodeDecodeRoundTrip(f *testing.F) {
	f.Add("ping", "req-1", "")
	f.Add("auth", "", "\x00\xff")

	f.Fuzz(func(t *testing.T, typ, reqID, token string) {
		// encoding/json replaces invalid UTF-8 with U+FFFD by design, so only
		// well-formed strings can be expected to survive a round-trip.
		if !utf8.ValidString(typ) || !utf8.ValidString(reqID) || !utf8.ValidString(token) {
			t.Skip()
		}
		var buf bytes.Buffer
		c := NewCodec(&buf, &buf)
		in := &AuthMessage{
			Message: Message{Type: MessageType(typ), RequestID: reqID},
			Token:   token,
		}
		if err := c.Encode(in); err != nil {
			return
		}
		var out AuthMessage
		if err := c.Decode(&out); err != nil {
			t.Fatalf("decode of our own frame failed: %v", err)
		}
		if out.Token != in.Token || out.Type != in.Type || out.RequestID != in.RequestID {
			t.Fatalf("round-trip mismatch: %+v vs %+v", in, out)
		}
	})
}

// FuzzNegotiateCompression drives the 1-byte handshake plus the zstd stream
// that follows it with bytes chosen by the peer.
func FuzzNegotiateCompression(f *testing.F) {
	f.Add([]byte{0x01}, true)
	f.Add([]byte{0x00}, false)
	f.Add([]byte{0x01, 0x28, 0xb5, 0x2f, 0xfd}, true)

	f.Fuzz(func(t *testing.T, peerBytes []byte, want bool) {
		local, remote := net.Pipe()
		defer local.Close()
		defer remote.Close()

		// net.Pipe is unbuffered, so the peer has to drain concurrently with
		// writing or the two sides deadlock on the handshake byte.
		done := make(chan struct{})
		drained := make(chan struct{})
		go func() {
			defer close(drained)
			_, _ = io.Copy(io.Discard, remote)
		}()
		go func() {
			defer close(done)
			_ = remote.SetDeadline(time.Now().Add(2 * time.Second))
			_, _ = remote.Write(peerBytes)
			// Hang up once the peer has said everything it has to say, so the
			// reads below end in EOF instead of sitting on the deadline.
			remote.Close()
		}()

		conn, _, err := NegotiateCompression(local, want, true)
		if err == nil {
			_ = local.SetDeadline(time.Now().Add(2 * time.Second))
			_, _ = io.Copy(io.Discard, io.LimitReader(conn, 1<<20))
		}
		local.Close()
		remote.Close()
		<-done
		<-drained
	})
}
