package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failWriter fails after N successful writes.
type failWriter struct {
	n       int
	written int
}

func (fw *failWriter) Write(p []byte) (int, error) {
	if fw.written >= fw.n {
		return 0, errors.New("write error")
	}
	fw.written++
	return len(p), nil
}

func cloneEmpty(msg any) any {
	switch msg.(type) {
	case *AuthMessage:
		return &AuthMessage{}
	case *AuthResultMessage:
		return &AuthResultMessage{}
	case *TunnelRequestMessage:
		return &TunnelRequestMessage{}
	case *TunnelCreatedMessage:
		return &TunnelCreatedMessage{}
	case *TunnelCloseMessage:
		return &TunnelCloseMessage{}
	case *TunnelClosedMessage:
		return &TunnelClosedMessage{}
	case *TunnelErrorMessage:
		return &TunnelErrorMessage{}
	case *NewConnectionMessage:
		return &NewConnectionMessage{}
	case *ConnectionAcceptMessage:
		return &ConnectionAcceptMessage{}
	case *ConnectionCloseMessage:
		return &ConnectionCloseMessage{}
	case *PingMessage:
		return &PingMessage{}
	case *PongMessage:
		return &PongMessage{}
	case *ErrorMessage:
		return &ErrorMessage{}
	default:
		return nil
	}
}

func TestNewMessage(t *testing.T) {
	msg := NewMessage(MsgAuth)
	assert.Equal(t, MsgAuth, msg.Type)
	assert.NotZero(t, msg.Timestamp)
}

func TestCodecEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  any
	}{
		{"Auth", &AuthMessage{Message: NewMessage(MsgAuth), Token: "tk_123", ClientID: "c1"}},
		{"AuthResult", &AuthResultMessage{Message: NewMessage(MsgAuthResult), Success: true, ClientID: "c1", MaxTunnels: 5}},
		{"TunnelRequest", &TunnelRequestMessage{Message: NewMessage(MsgTunnelRequest), TunnelType: TunnelHTTP, Subdomain: "test", LocalPort: 8080}},
		{"TunnelCreated", &TunnelCreatedMessage{Message: NewMessage(MsgTunnelCreated), TunnelID: "t1", TunnelType: TunnelTCP, RemotePort: 9000}},
		{"TunnelClose", &TunnelCloseMessage{Message: NewMessage(MsgTunnelClose), TunnelID: "t1"}},
		{"TunnelClosed", &TunnelClosedMessage{Message: NewMessage(MsgTunnelClosed), TunnelID: "t1"}},
		{"TunnelError", &TunnelErrorMessage{Message: NewMessage(MsgTunnelError), Error: "fail", Code: ErrCodeInternalError}},
		{"NewConnection", &NewConnectionMessage{Message: NewMessage(MsgNewConnection), TunnelID: "t1", ConnectionID: "cn1", RemoteAddr: "1.2.3.4:5678"}},
		{"ConnectionAccept", &ConnectionAcceptMessage{Message: NewMessage(MsgConnectionAccept), ConnectionID: "cn1"}},
		{"ConnectionClose", &ConnectionCloseMessage{Message: NewMessage(MsgConnectionClose), ConnectionID: "cn1"}},
		{"Ping", &PingMessage{Message: NewMessage(MsgPing)}},
		{"Pong", &PongMessage{Message: NewMessage(MsgPong)}},
		{"Error", &ErrorMessage{Message: NewMessage(MsgError), Error: "bad", Fatal: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			codec := NewCodec(&buf, &buf)

			err := codec.Encode(tt.msg)
			require.NoError(t, err)

			dst := cloneEmpty(tt.msg)
			require.NotNil(t, dst)

			err = codec.Decode(dst)
			require.NoError(t, err)

			origJSON, _ := json.Marshal(tt.msg)
			dstJSON, _ := json.Marshal(dst)
			assert.JSONEq(t, string(origJSON), string(dstJSON))
		})
	}
}

func TestCodecDecodeRaw(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(&buf, &buf)

	orig := &AuthMessage{Message: NewMessage(MsgAuth), Token: "secret"}
	require.NoError(t, codec.Encode(orig))

	raw, baseMsg, err := codec.DecodeRaw()
	require.NoError(t, err)
	assert.Equal(t, MsgAuth, baseMsg.Type)

	parsed, err := ParseMessage(raw, baseMsg.Type)
	require.NoError(t, err)
	am, ok := parsed.(*AuthMessage)
	require.True(t, ok)
	assert.Equal(t, "secret", am.Token)
}

func TestCodecEncodeBytesRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(&buf, &buf)

	orig := &PingMessage{Message: NewMessage(MsgPing)}
	data, err := json.Marshal(orig)
	require.NoError(t, err)

	require.NoError(t, codec.EncodeBytes(data))

	var decoded PingMessage
	require.NoError(t, codec.Decode(&decoded))
	assert.Equal(t, MsgPing, decoded.Type)
}

func TestCodecEncodeTooLarge(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(&buf, &buf)

	data := make([]byte, MaxMessageSize+1)
	err := codec.EncodeBytes(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

func TestCodecDecodeTooLargeHeader(t *testing.T) {
	var buf bytes.Buffer
	header := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(header, MaxMessageSize+1)
	buf.Write(header)

	codec := NewCodec(&buf, &buf)
	var msg Message
	err := codec.Decode(&msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

func TestCodecDecodeEmptyReader(t *testing.T) {
	codec := NewCodec(bytes.NewReader(nil), nil)
	var msg Message
	err := codec.Decode(&msg)
	assert.Error(t, err)
}

func TestCodecDecodePartialHeader(t *testing.T) {
	codec := NewCodec(bytes.NewReader([]byte{0x00, 0x00}), nil)
	var msg Message
	err := codec.Decode(&msg)
	assert.Error(t, err)
}

func TestCodecDecodeInvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	payload := []byte("{invalid json!!!")
	header := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(header, uint32(len(payload))) //nolint:gosec // test data, len() is small
	buf.Write(header)
	buf.Write(payload)

	codec := NewCodec(&buf, nil)
	var msg Message
	err := codec.Decode(&msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestCodecDecodePartialPayload(t *testing.T) {
	var buf bytes.Buffer
	header := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(header, 100)
	buf.Write(header)
	buf.Write([]byte("short"))

	codec := NewCodec(&buf, nil)
	var msg Message
	err := codec.Decode(&msg)
	assert.Error(t, err)
}

func TestCodecEncodeWriteError(t *testing.T) {
	// Encode uses a single combined write (header+payload), so fail on first write
	codec := NewCodec(nil, &failWriter{n: 0})
	err := codec.Encode(&PingMessage{Message: NewMessage(MsgPing)})
	assert.Error(t, err)
}

func TestParseMessageAllTypes(t *testing.T) {
	allTypes := []MessageType{
		MsgAuth, MsgAuthResult, MsgTunnelRequest, MsgTunnelCreated,
		MsgTunnelClose, MsgTunnelClosed, MsgTunnelError,
		MsgNewConnection, MsgConnectionAccept, MsgConnectionClose,
		MsgPing, MsgPong, MsgError,
	}
	for _, mt := range allTypes {
		t.Run(string(mt), func(t *testing.T) {
			data, _ := json.Marshal(Message{Type: mt, Timestamp: 1})
			parsed, err := ParseMessage(data, mt)
			require.NoError(t, err)
			assert.NotNil(t, parsed)
		})
	}
}

func TestParseMessageUnknownType(t *testing.T) {
	data, _ := json.Marshal(Message{Type: "unknown"})
	_, err := ParseMessage(data, "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown message type")
}

func TestParseMessageInvalidJSON(t *testing.T) {
	_, err := ParseMessage([]byte("{bad"), MsgAuth)
	assert.Error(t, err)
}

func TestDecodeRawTooLarge(t *testing.T) {
	var buf bytes.Buffer
	header := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(header, MaxMessageSize+1)
	buf.Write(header)

	codec := NewCodec(&buf, nil)
	_, _, err := codec.DecodeRaw()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

func TestDecodeRawEmptyReader(t *testing.T) {
	codec := NewCodec(bytes.NewReader(nil), nil)
	_, _, err := codec.DecodeRaw()
	assert.Error(t, err)
}

func TestDecodeRawInvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	payload := []byte("{not json!!")
	header := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(header, uint32(len(payload))) //nolint:gosec // test data, len() is small
	buf.Write(header)
	buf.Write(payload)

	codec := NewCodec(&buf, nil)
	_, _, err := codec.DecodeRaw()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

// Ensure io import is used
var _ io.Reader
