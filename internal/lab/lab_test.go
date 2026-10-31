package lab

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

func TestValidatePorts(t *testing.T) {
	cases := map[string][]int{
		"empty":      {},
		"privileged": {80},
		"too high":   {70000},
		"duplicate":  {18080, 18080},
		"too many":   {2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008},
	}
	for name, ports := range cases {
		if err := ValidatePorts(ports); !errors.Is(err, domain.ErrValidation) {
			t.Errorf("%s: err = %v, want ErrInvalidPorts", name, err)
		}
	}
	if err := ValidatePorts([]int{18080, 18022}); err != nil {
		t.Fatalf("valid ports rejected: %v", err)
	}
}

func TestServeAcceptsOnLoopbackAndStops(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("loopback unavailable: %v", err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	probe.Close()

	ctx, cancel := context.WithCancel(context.Background())
	addrCh := make(chan []string, 1)
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, []int{port}, func(addrs []string) { addrCh <- addrs })
	}()
	addrs := <-addrCh
	if len(addrs) != 1 || addrs[0] != net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) {
		t.Fatalf("addrs = %v", addrs)
	}
	conn, err := net.DialTimeout("tcp", addrs[0], 2*time.Second)
	if err != nil {
		t.Fatalf("dial lab: %v", err)
	}
	banner, _ := io.ReadAll(conn)
	conn.Close()
	if string(banner) == "" {
		t.Fatal("lab must send its synthetic banner")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if late, err := net.DialTimeout("tcp", addrs[0], 500*time.Millisecond); err == nil {
		late.Close()
		t.Fatal("lab must stop listening after cancellation")
	}
}
