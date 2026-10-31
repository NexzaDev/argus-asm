package probe

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTCPConnectDetectsListeningPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("loopback listener unavailable: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	open, err := TCPConnect(context.Background(), "127.0.0.1", port, time.Second)
	if err != nil || !open {
		t.Fatalf("listening port: open=%t err=%v", open, err)
	}
	free, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closedPort := free.Addr().(*net.TCPAddr).Port
	free.Close()
	open, err = TCPConnect(context.Background(), "127.0.0.1", closedPort, time.Second)
	if err != nil || open {
		t.Fatalf("closed port: open=%t err=%v", open, err)
	}
}

func TestTCPConnectHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := TCPConnect(ctx, "127.0.0.1", 9, time.Second); err == nil {
		t.Fatal("cancelled context must return an error")
	}
}
