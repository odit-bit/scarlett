package store

import (
	"net"
	"time"

	"github.com/hashicorp/raft"
)

var _ raft.StreamLayer = (*StreamLayer)(nil)

type StreamLayer struct {
	Listener net.Listener
}

// Accept implements raft.StreamLayer.
func (s *StreamLayer) Accept() (net.Conn, error) {
	return s.Listener.Accept()
}

// Addr implements raft.StreamLayer.
func (s *StreamLayer) Addr() net.Addr {
	return s.Listener.Addr()
}

// Close implements raft.StreamLayer.
func (s *StreamLayer) Close() error {
	return s.Listener.Close()
}

// Dial implements raft.StreamLayer.
func (s *StreamLayer) Dial(address raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	return net.Dial("tcp", string(address))
}
