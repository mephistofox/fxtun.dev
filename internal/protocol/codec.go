package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const (
	// MaxMessageSize is the maximum allowed message size (1MB)
	MaxMessageSize = 1 << 20
	// HeaderSize is the size of the length prefix
	HeaderSize = 4
)

// Codec handles encoding and decoding of protocol messages
type Codec struct {
	reader io.Reader
	writer io.Writer
}

// NewCodec creates a new codec for the given reader/writer
func NewCodec(r io.Reader, w io.Writer) *Codec {
	return &Codec{
		reader: r,
		writer: w,
	}
}

// Encode writes a message to the writer with length prefix
func (c *Codec) Encode(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	if len(data) > MaxMessageSize {
		return fmt.Errorf("message too large: %d > %d", len(data), MaxMessageSize)
	}

	// Write length prefix + payload in single write
	buf := make([]byte, HeaderSize+len(data))
	binary.BigEndian.PutUint32(buf[:HeaderSize], uint32(len(data)))
	copy(buf[HeaderSize:], data)

	if _, err := c.writer.Write(buf); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// Decode reads a message from the reader
func (c *Codec) Decode(msg any) error {
	// Read length prefix
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	length := binary.BigEndian.Uint32(header)
	if length > MaxMessageSize {
		return fmt.Errorf("message too large: %d > %d", length, MaxMessageSize)
	}

	// Read payload
	data := make([]byte, length)
	if _, err := io.ReadFull(c.reader, data); err != nil {
		return fmt.Errorf("read payload: %w", err)
	}

	if err := json.Unmarshal(data, msg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	return nil
}

// DecodeRaw reads a message and returns raw JSON along with the base message
func (c *Codec) DecodeRaw() ([]byte, *Message, error) {
	// Read length prefix
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return nil, nil, fmt.Errorf("read header: %w", err)
	}

	length := binary.BigEndian.Uint32(header)
	if length > MaxMessageSize {
		return nil, nil, fmt.Errorf("message too large: %d > %d", length, MaxMessageSize)
	}

	// Read payload
	data := make([]byte, length)
	if _, err := io.ReadFull(c.reader, data); err != nil {
		return nil, nil, fmt.Errorf("read payload: %w", err)
	}

	// Decode base message to get type
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, nil, fmt.Errorf("unmarshal base message: %w", err)
	}

	return data, &msg, nil
}

// EncodeBytes writes raw bytes with length prefix
func (c *Codec) EncodeBytes(data []byte) error {
	if len(data) > MaxMessageSize {
		return fmt.Errorf("message too large: %d > %d", len(data), MaxMessageSize)
	}

	// Write length prefix + payload in single write
	buf := make([]byte, HeaderSize+len(data))
	binary.BigEndian.PutUint32(buf[:HeaderSize], uint32(len(data)))
	copy(buf[HeaderSize:], data)

	if _, err := c.writer.Write(buf); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// ParseMessage parses raw JSON into the appropriate message type
func ParseMessage(data []byte, msgType MessageType) (any, error) {
	var msg any

	switch msgType {
	case MsgAuth:
		msg = &AuthMessage{}
	case MsgAuthResult:
		msg = &AuthResultMessage{}
	case MsgTunnelRequest:
		msg = &TunnelRequestMessage{}
	case MsgTunnelCreated:
		msg = &TunnelCreatedMessage{}
	case MsgTunnelClose:
		msg = &TunnelCloseMessage{}
	case MsgTunnelClosed:
		msg = &TunnelClosedMessage{}
	case MsgTunnelError:
		msg = &TunnelErrorMessage{}
	case MsgNewConnection:
		msg = &NewConnectionMessage{}
	case MsgConnectionAccept:
		msg = &ConnectionAcceptMessage{}
	case MsgConnectionClose:
		msg = &ConnectionCloseMessage{}
	case MsgPing:
		msg = &PingMessage{}
	case MsgPong:
		msg = &PongMessage{}
	case MsgError:
		msg = &ErrorMessage{}
	default:
		return nil, fmt.Errorf("unknown message type: %s", msgType)
	}

	if err := json.Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", msgType, err)
	}

	return msg, nil
}
