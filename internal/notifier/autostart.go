package notifier

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	LinuxFileName   = "argus-asm-notifier.desktop"
	DarwinLabel     = "com.nexzadev.argus-asm.notifier"
	WindowsFileName = "argus-asm-notifier.cmd"
)

type Layout struct {
	GOOS       string
	Home       string
	AppData    string
	Executable string
	Args       []string
}

type Plan struct {
	Path    string
	Content string
}

func PlanInstall(layout Layout) (Plan, error) {
	if strings.TrimSpace(layout.Home) == "" && layout.GOOS != "windows" {
		return Plan{}, fmt.Errorf("home directory is unknown")
	}
	for _, value := range append([]string{layout.Executable}, layout.Args...) {
		if strings.ContainsAny(value, "\r\n") {
			return Plan{}, fmt.Errorf("autostart value contains a line break")
		}
	}
	argv := append([]string{layout.Executable}, layout.Args...)
	switch layout.GOOS {
	case "linux":
		return Plan{
			Path:    filepath.Join(layout.Home, ".config", "autostart", LinuxFileName),
			Content: linuxDesktop(argv),
		}, nil
	case "darwin":
		return Plan{
			Path:    filepath.Join(layout.Home, "Library", "LaunchAgents", DarwinLabel+".plist"),
			Content: darwinPlist(argv),
		}, nil
	case "windows":
		if strings.TrimSpace(layout.AppData) == "" {
			return Plan{}, fmt.Errorf("APPDATA is not set")
		}
		for _, value := range argv {
			if strings.ContainsAny(value, "\"%") {
				return Plan{}, fmt.Errorf("autostart on Windows cannot represent quotes or percent signs in %q", value)
			}
		}
		return Plan{
			Path:    filepath.Join(layout.AppData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", WindowsFileName),
			Content: windowsCmd(argv),
		}, nil
	default:
		return Plan{}, fmt.Errorf("autostart is not supported on %s", layout.GOOS)
	}
}

func linuxDesktop(argv []string) string {
	quoted := make([]string, len(argv))
	for i, value := range argv {
		escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", "$", `\$`).Replace(value)
		quoted[i] = `"` + escaped + `"`
	}
	return "[Desktop Entry]\nType=Application\nName=Argus ASM notifier\n" +
		"Comment=Shows Argus ASM notifications in the background\n" +
		"Exec=" + strings.Join(quoted, " ") + "\nTerminal=false\nHidden=false\nX-GNOME-Autostart-enabled=true\n"
}

func darwinPlist(argv []string) string {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	b.WriteString("<plist version=\"1.0\">\n<dict>\n")
	b.WriteString("\t<key>Label</key>\n\t<string>" + DarwinLabel + "</string>\n")
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	for _, value := range argv {
		b.WriteString("\t\t<string>" + xmlEscape(value) + "</string>\n")
	}
	b.WriteString("\t</array>\n\t<key>RunAtLoad</key>\n\t<true/>\n\t<key>KeepAlive</key>\n\t<false/>\n</dict>\n</plist>\n")
	return b.String()
}

func windowsCmd(argv []string) string {
	quoted := make([]string, len(argv))
	for i, value := range argv {
		quoted[i] = `"` + value + `"`
	}
	return "@echo off\r\nstart \"\" /min " + strings.Join(quoted, " ") + "\r\n"
}

func xmlEscape(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;").Replace(value)
}
