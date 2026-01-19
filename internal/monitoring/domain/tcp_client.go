package domain

import (
	"context"
)

// TCPClient defines the interface for TCP client operations
// This interface abstracts TCP concerns from the domain layer
type TCPClient interface {
	// Dial performs a TCP connection check to the given hostname and port
	// Returns an error if the connection fails
	Dial(ctx context.Context, hostname, port string) error
}


