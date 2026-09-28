package protocol

import (
	"fmt"
	"io"
)

// StreamHeader is the binary header sent at the start of each data stream
// to identify the tunnel and remote address.
//
// Wire format: [1 byte: tunnel_id_len][tunnel_id bytes][1 byte: remote_addr_len][remote_addr bytes]
type StreamHeader struct {
	TunnelID   string
	RemoteAddr string
}

// WriteStreamHeader writes a compact binary header to w.
func WriteStreamHeader(w io.Writer, tunnelID, remoteAddr string) error {
	tidLen := len(tunnelID)
	raLen := len(remoteAddr)
	if tidLen > 255 {
		return fmt.Errorf("tunnel_id too long: %d", tidLen)
	}
	if raLen > 255 {
		return fmt.Errorf("remote_addr too long: %d", raLen)
	}

	buf := make([]byte, 1+tidLen+1+raLen)
	buf[0] = byte(tidLen) //nolint:gosec // bounded above
	copy(buf[1:], tunnelID)
	buf[1+tidLen] = byte(raLen) //nolint:gosec // bounded above
	copy(buf[2+tidLen:], remoteAddr)

	_, err := w.Write(buf)
	return err
}

// ReadStreamHeader reads the binary stream header from r.
func ReadStreamHeader(r io.Reader) (*StreamHeader, error) {
	var lenBuf [1]byte

	// Read tunnel ID
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read tunnel_id length: %w", err)
	}
	tidLen := int(lenBuf[0])
	tid := make([]byte, tidLen)
	if tidLen > 0 {
		if _, err := io.ReadFull(r, tid); err != nil {
			return nil, fmt.Errorf("read tunnel_id: %w", err)
		}
	}

	// Read remote addr
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read remote_addr length: %w", err)
	}
	raLen := int(lenBuf[0])
	ra := make([]byte, raLen)
	if raLen > 0 {
		if _, err := io.ReadFull(r, ra); err != nil {
			return nil, fmt.Errorf("read remote_addr: %w", err)
		}
	}

	return &StreamHeader{
		TunnelID:   string(tid),
		RemoteAddr: string(ra),
	}, nil
}
