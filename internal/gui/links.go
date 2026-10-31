//go:build !nogui

package gui

import (
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/buildinfo"
)

var errExternalLink = errors.New("external links are limited to the Argus ASM project on GitHub")

func projectLink(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errExternalLink
	}
	base, err := url.Parse(buildinfo.ProjectURL)
	if err != nil {
		return nil, errExternalLink
	}
	if u.Scheme != "https" || u.Host != base.Host || u.User != nil {
		return nil, errExternalLink
	}
	if u.Path != base.Path && !strings.HasPrefix(u.Path, base.Path+"/") {
		return nil, errExternalLink
	}
	return u, nil
}

func bugReportBody() string {
	return fmt.Sprintf(`Version: %s
Platform: %s/%s
Go runtime: %s

What happened:

Steps to reproduce:

No logs, evidence, or credentials are attached. Add only what is safe to share.
`, buildinfo.Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
}

func newIssueURL(body string) string {
	return buildinfo.ProjectURL + "/issues/new?title=" + url.QueryEscape("Bug report") + "&body=" + url.QueryEscape(body)
}
