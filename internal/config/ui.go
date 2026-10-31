package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	EnvUI        = "ARGUS_UI"
	UIGUI        = "gui"
	UIConsole    = "console"
	UICLI        = "cli"
	SourceStored = "stored"
	SourceNone   = "unset"
	uiFileName   = "ui.conf"
	uiKeyPrefix  = "ui="
)

type UILocation struct {
	Mode   string
	Source string
}

func ValidUIMode(mode string) bool {
	return mode == UIGUI || mode == UIConsole || mode == UICLI
}

func uiModesText() string {
	return UIGUI + ", " + UIConsole + ", or " + UICLI
}

func UIFilePath(userConfigDir func() (string, error)) (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine user config directory: %w", err)
	}
	return filepath.Join(dir, appDirName, uiFileName), nil
}

func ParseStoredUI(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", nil
	}
	if !strings.HasPrefix(trimmed, uiKeyPrefix) {
		return "", fmt.Errorf("stored UI preference is malformed")
	}
	mode := strings.TrimSpace(strings.TrimPrefix(trimmed, uiKeyPrefix))
	if !ValidUIMode(mode) {
		return "", fmt.Errorf("stored UI preference %q is not one of %s", mode, uiModesText())
	}
	return mode, nil
}

func FormatStoredUI(mode string) (string, error) {
	if !ValidUIMode(mode) {
		return "", fmt.Errorf("UI mode %q is not one of %s", mode, uiModesText())
	}
	return uiKeyPrefix + mode + "\n", nil
}

func ResolveUI(flagValue string, getenv func(string) string, stored string) (UILocation, error) {
	if value := strings.TrimSpace(flagValue); value != "" {
		if !ValidUIMode(value) {
			return UILocation{}, fmt.Errorf("--ui %q is not one of %s", value, uiModesText())
		}
		return UILocation{Mode: value, Source: SourceFlag}, nil
	}
	if value := strings.TrimSpace(getenv(EnvUI)); value != "" {
		if !ValidUIMode(value) {
			return UILocation{}, fmt.Errorf("%s=%q is not one of %s", EnvUI, value, uiModesText())
		}
		return UILocation{Mode: value, Source: SourceEnv}, nil
	}
	if stored != "" {
		if !ValidUIMode(stored) {
			return UILocation{}, fmt.Errorf("stored UI preference %q is invalid; run 'argus-asm config reset-ui'", stored)
		}
		return UILocation{Mode: stored, Source: SourceStored}, nil
	}
	return UILocation{Source: SourceNone}, nil
}
