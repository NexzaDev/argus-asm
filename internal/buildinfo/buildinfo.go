package buildinfo

import "strings"

const (
	Name        = "argus-asm"
	Description = "Local, defensive attack surface inventory and findings tracker"
	License     = "MIT"
)

var (
	Version    = "0.1.0-dev"
	Commit     = "unknown"
	BuildDate  = "unknown"
	ProjectURL = "https://github.com/NexzaDev/argus-asm"
)

func IssuesURL() string {
	if strings.TrimSpace(ProjectURL) == "" {
		return ""
	}
	return ProjectURL + "/issues/new/choose"
}
