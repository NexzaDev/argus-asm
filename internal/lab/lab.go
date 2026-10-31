package lab

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const (
	Host       = "127.0.0.1"
	MinPort    = 1024
	MaxPort    = 65535
	MaxPorts   = 8
	bannerText = "argus-asm local lab: synthetic service, no data is served\n"
	writeLimit = 2 * time.Second
)

func ValidatePorts(ports []int) error {
	if len(ports) == 0 {
		return fmt.Errorf("%w: at least one lab port is required", domain.ErrValidation)
	}
	if len(ports) > MaxPorts {
		return fmt.Errorf("%w: at most %d lab ports", domain.ErrValidation, MaxPorts)
	}
	seen := map[int]bool{}
	for _, p := range ports {
		if p < MinPort || p > MaxPort {
			return fmt.Errorf("%w: lab port %d must be between %d and %d (unprivileged)", domain.ErrValidation, p, MinPort, MaxPort)
		}
		if seen[p] {
			return fmt.Errorf("%w: lab port %d listed twice", domain.ErrValidation, p)
		}
		seen[p] = true
	}
	return nil
}

func Serve(ctx context.Context, ports []int, ready func(addrs []string)) error {
	if err := ValidatePorts(ports); err != nil {
		return err
	}
	listeners := make([]net.Listener, 0, len(ports))
	addrs := make([]string, 0, len(ports))
	closeAll := func() {
		for _, ln := range listeners {
			_ = ln.Close()
		}
	}
	for _, p := range ports {
		addr := net.JoinHostPort(Host, strconv.Itoa(p))
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			closeAll()
			return fmt.Errorf("listen on %s: %w", addr, err)
		}
		listeners = append(listeners, ln)
		addrs = append(addrs, addr)
	}
	if ready != nil {
		ready(addrs)
	}
	var wg sync.WaitGroup
	for _, ln := range listeners {
		wg.Add(1)
		go func(ln net.Listener) {
			defer wg.Done()
			acceptLoop(ln)
		}(ln)
	}
	<-ctx.Done()
	closeAll()
	wg.Wait()
	return nil
}

func acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go serveConn(conn)
	}
}

func serveConn(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(writeLimit))
	_, _ = io.WriteString(conn, bannerText)
}
