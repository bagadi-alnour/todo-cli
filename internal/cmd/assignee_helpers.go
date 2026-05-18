package cmd

import (
	"fmt"
	"strings"

	"github.com/bagadi-alnour/todo-cli/internal/contributors"
	"github.com/bagadi-alnour/todo-cli/internal/terminal"
)

func resolveAssignee(projectRoot, query string) (email string, err error) {
	email, _, err = contributors.Resolve(projectRoot, query)
	return email, err
}

func formatAssigneeLabel(projectRoot, email string) string {
	if email == "" {
		return ""
	}
	return contributors.LookupName(projectRoot, email)
}

func printAssigneeHint(projectRoot string, paths []string) {
	if len(paths) == 0 {
		return
	}
	suggested, err := contributors.SuggestFromBlame(projectRoot, paths)
	if err != nil || len(suggested) == 0 {
		return
	}
	top := suggested[0]
	label := contributors.DisplayName(top)
	fmt.Printf("  %s💡 Suggested assignee: %s — todo edit <id> --assign %s%s\n",
		terminal.Dim, label, strings.Split(top.Email, "@")[0], terminal.Reset)
}
