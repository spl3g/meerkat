package infrastructure

import (
	"context"
	"net"

	"meerkat-v0/internal/monitoring/domain"
)

// TCPClientImpl implements the domain TCPClient interface
type TCPClientImpl struct{}

// NewTCPClient creates a new TCP client implementation
func NewTCPClient() domain.TCPClient {
	return &TCPClientImpl{}
}

// Dial performs a TCP connection check to the given hostname and port
func (c *TCPClientImpl) Dial(ctx context.Context, hostname, port string) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(hostname, port))
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}


