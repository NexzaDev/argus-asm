package probe

import (
	"context"
	"net"
	"strconv"
	"time"
)

type Dialer func(ctx context.Context, host string, port int, timeout time.Duration) (bool, error)

func TCPConnect(ctx context.Context, host string, port int, timeout time.Duration) (bool, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, nil
	}
	_ = conn.Close()
	return true, nil
}
