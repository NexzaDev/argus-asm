package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var ErrForbidden = errors.New("permission denied")

type Role string

const (
	RoleViewer  Role = "viewer"
	RoleAnalyst Role = "analyst"
	RoleAdmin   Role = "admin"
)

type Permission string

const (
	PermRead       Permission = "read"
	PermOperate    Permission = "operate"
	PermAdminister Permission = "administer"
)

const BootstrapActorName = "local"

var userNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func ParseRole(raw string) (Role, error) {
	switch role := Role(strings.ToLower(strings.TrimSpace(raw))); role {
	case RoleViewer, RoleAnalyst, RoleAdmin:
		return role, nil
	}
	return "", validationErrorf("unknown role %q (expected viewer, analyst, or admin)", raw)
}

func ValidateUserName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if !userNamePattern.MatchString(name) {
		return "", validationErrorf("user name %q must be 1-64 letters, digits, '.', '_' or '-'", raw)
	}
	return name, nil
}

func (r Role) Grants(p Permission) bool {
	switch p {
	case PermRead:
		return r == RoleViewer || r == RoleAnalyst || r == RoleAdmin
	case PermOperate:
		return r == RoleAnalyst || r == RoleAdmin
	case PermAdminister:
		return r == RoleAdmin
	default:
		return false
	}
}

type Actor struct {
	Name string `json:"name"`
	Role Role   `json:"role"`
}

func (a Actor) Require(p Permission) error {
	if !a.Role.Grants(p) {
		return fmt.Errorf("%w: %s (role %s) lacks %s permission", ErrForbidden, a.Name, a.Role, p)
	}
	return nil
}

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
