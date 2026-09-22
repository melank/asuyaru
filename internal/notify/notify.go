// Package notify は macOS の通知センターへの通知を持つ。
package notify

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Display は osascript の display notification で通知センターへ出す。
func Display(ctx context.Context, title, body string) error {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script(title, body))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("通知を出す: %w: %s", err, out)
	}
	return nil
}

func script(title, body string) string {
	return "display notification " + appleString(body) + " with title " + appleString(title)
}

func appleString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
