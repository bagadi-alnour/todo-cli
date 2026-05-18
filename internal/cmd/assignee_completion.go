package cmd

import (
	"sort"
	"strings"

	"github.com/bagadi-alnour/todo-cli/internal/contributors"
	"github.com/spf13/cobra"
)

func registerAssigneeFlagCompletion(command *cobra.Command, flagName string) {
	_ = command.RegisterFlagCompletionFunc(flagName, completeAssignee)
}

func completeAssignee(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	projectRoot := findProjectRootOrWD()
	f, err := contributors.EnsureLoaded(projectRoot)
	if err != nil {
		return nil, cobra.ShellCompDirectiveDefault
	}

	prefix := strings.ToLower(toComplete)
	var out []string
	seen := map[string]struct{}{}

	add := func(value string) {
		if _, ok := seen[value]; ok {
			return
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(value), prefix) {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	if prefix == "" || strings.HasPrefix("me", prefix) {
		add("me")
	}

	for _, c := range f.Contributors {
		local := strings.Split(c.Email, "@")[0]
		add(local)
		if c.Name != "" {
			add(c.Name)
		}
	}

	sort.Strings(out)
	return out, cobra.ShellCompDirectiveNoFileComp
}
