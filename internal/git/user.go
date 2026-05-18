package git

import (
	"os/exec"
	"strings"
)

// GetUserEmail returns the configured git user.email.
func GetUserEmail() (string, error) {
	cmd := exec.Command("git", "config", "user.email")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetUserName returns the configured git user.name.
func GetUserName() (string, error) {
	cmd := exec.Command("git", "config", "user.name")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
