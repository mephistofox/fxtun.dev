package protocol

import "time"

// MessageType defines the type of control message
type MessageType string

const (
	// Authentication
	MsgAuth       MessageType = "auth"
	MsgAuthResult MessageType = "auth_result"

	// Tunnel management
	MsgTunnelRequest MessageType = "tunnel_request"
	MsgTunnelCreated MessageType = "tunnel_created"
	MsgTunnelClose   MessageType = "tunnel_close"
	MsgTunnelClosed  MessageType = "tunnel_closed"
	MsgTunnelError   MessageType = "tunnel_error"

	// Connection notifications
	MsgNewConnection    MessageType = "new_connection"
	MsgConnectionAccept MessageType = "connection_accept"
	MsgConnectionClose  MessageType = "connection_close"

	// Keepalive
	MsgPing MessageType = "ping"
	MsgPong MessageType = "pong"

	// Server lifecycle
	MsgServerShutdown MessageType = "server_shutdown"

	// Session pooling
	MsgJoinSession       MessageType = "join_session"
	MsgJoinSessionResult MessageType = "join_session_result"

	// Errors
	MsgError MessageType = "error"
)

// TunnelType defines the type of tunnel
type TunnelType string

const (
	TunnelHTTP TunnelType = "http"
	TunnelTCP  TunnelType = "tcp"
	TunnelUDP  TunnelType = "udp"
)

// Message is the base structure for all control messages
type Message struct {
	Type      MessageType `json:"type"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// NewMessage creates a new message with the given type
func NewMessage(msgType MessageType) Message {
	return Message{
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
	}
}

// AuthMessage is sent by client to authenticate
type AuthMessage struct {
	Message
	Token     string `json:"token"`
	ClientID  string `json:"client_id,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// ClientCapabilities describes features available based on the user's plan.
type ClientCapabilities struct {
	InspectorEnabled bool `json:"inspector_enabled"`
	MaxBodySize      int  `json:"max_body_size,omitempty"`
	MaxBufferEntries int  `json:"max_buffer_entries,omitempty"`
}

// AuthResultMessage is the server response to authentication
type AuthResultMessage struct {
	Message
	Success       bool                `json:"success"`
	ClientID      string              `json:"client_id,omitempty"`
	Error         string              `json:"error,omitempty"`
	Code          string              `json:"code,omitempty"`
	MaxTunnels    int                 `json:"max_tunnels,omitempty"`
	ServerName    string              `json:"server_name,omitempty"`
	SessionID     string              `json:"session_id,omitempty"`
	SessionSecret string              `json:"session_secret,omitempty"`
	MinVersion    string              `json:"min_version,omitempty"`
	Capabilities  *ClientCapabilities `json:"capabilities,omitempty"`
}

// TunnelRequestMessage is sent by client to create a tunnel
type TunnelRequestMessage struct {
	Message
	TunnelType TunnelType `json:"tunnel_type"`
	Name       string     `json:"name,omitempty"`

	// For HTTP tunnels
	Subdomain string `json:"subdomain,omitempty"`

	// For TCP/UDP tunnels
	LocalPort  int `json:"local_port"`
	RemotePort int `json:"remote_port,omitempty"` // 0 = auto-assign
}

// TunnelCreatedMessage is the server response when tunnel is created
type TunnelCreatedMessage struct {
	Message
	TunnelID   string     `json:"tunnel_id"`
	TunnelType TunnelType `json:"tunnel_type"`
	Name       string     `json:"name,omitempty"`

	// For HTTP tunnels
	URL       string `json:"url,omitempty"`
	Subdomain string `json:"subdomain,omitempty"`

	// For TCP/UDP tunnels
	RemotePort int    `json:"remote_port,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
}

// TunnelCloseMessage is sent to close a tunnel
type TunnelCloseMessage struct {
	Message
	TunnelID string `json:"tunnel_id"`
}

// TunnelClosedMessage confirms tunnel closure
type TunnelClosedMessage struct {
	Message
	TunnelID string `json:"tunnel_id"`
}

// TunnelErrorMessage indicates an error with a tunnel operation
type TunnelErrorMessage struct {
	Message
	TunnelID string `json:"tunnel_id,omitempty"`
	Error    string `json:"error"`
	Code     string `json:"code,omitempty"`
}

// NewConnectionMessage notifies client of incoming connection
type NewConnectionMessage struct {
	Message
	TunnelID     string `json:"tunnel_id"`
	ConnectionID string `json:"connection_id"`
	RemoteAddr   string `json:"remote_addr"`

	// For HTTP connections
	Host   string `json:"host,omitempty"`
	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
}

// ConnectionAcceptMessage tells server client is ready for data
type ConnectionAcceptMessage struct {
	Message
	ConnectionID string `json:"connection_id"`
}

// ConnectionCloseMessage notifies about connection closure
type ConnectionCloseMessage struct {
	Message
	ConnectionID string `json:"connection_id"`
	Error        string `json:"error,omitempty"`
}

// PingMessage for keepalive
type PingMessage struct {
	Message
}

// PongMessage for keepalive response
type PongMessage struct {
	Message
}

// ErrorMessage for general errors
type ErrorMessage struct {
	Message
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
	Fatal bool   `json:"fatal,omitempty"`
}

// ServerShutdownMessage is sent by server before shutting down
type ServerShutdownMessage struct {
	Message
	Reason string `json:"reason,omitempty"`
}

// JoinSessionMessage is sent by client to join an existing session with additional data connections
type JoinSessionMessage struct {
	Message
	ClientID string `json:"client_id"`
	Secret   string `json:"secret"`
}

// JoinSessionResult is the server response to a join session request
type JoinSessionResult struct {
	Message
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Error codes
const (
	ErrCodeAuthFailed       = "AUTH_FAILED"
	ErrCodeInvalidToken     = "INVALID_TOKEN"
	ErrCodeTokenExpired     = "TOKEN_EXPIRED"
	ErrCodeTunnelLimit      = "TUNNEL_LIMIT"
	ErrCodeSubdomainTaken   = "SUBDOMAIN_TAKEN"
	ErrCodeSubdomainInvalid = "SUBDOMAIN_INVALID"
	ErrCodePortUnavailable  = "PORT_UNAVAILABLE"
	ErrCodePermissionDenied = "PERMISSION_DENIED"
	ErrCodeInternalError    = "INTERNAL_ERROR"
	ErrCodeProtocolError    = "PROTOCOL_ERROR"
)
