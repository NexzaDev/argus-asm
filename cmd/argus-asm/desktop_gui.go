//go:build !nogui

package main

import (
	"github.com/NexzaDev/argus-asm/internal/desktop"
	"github.com/NexzaDev/argus-asm/internal/gui"
)

func newDesktop() desktop.Runner {
	return gui.Runner{}
}
