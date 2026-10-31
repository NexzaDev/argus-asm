package desktop

import (
	"context"
	"time"
)

type Request struct {
	DBPath      string
	Actor       string
	Now         func() time.Time
	ShowChooser bool
	Remember    func(mode string) error
	Forget      func() error
}

type Result struct {
	Choice string
}

type Runner interface {
	Unavailable() string
	Run(ctx context.Context, req Request) (Result, error)
}
