package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// A target is either positional or supplied with -t, never both. The trailing
// arguments belong to the command (for example the replacement session name).
func targetArgs(target *string, trailing int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("target") {
			if *target == "" {
				return fmt.Errorf("-t requires a non-empty session name or ID")
			}
			if len(args) != trailing {
				return fmt.Errorf("use either -t SESSION or a positional session, not both")
			}
		} else if len(args) != trailing+1 || args[0] == "" {
			return fmt.Errorf("a session target is required; use -t SESSION")
		}
		return nil
	}
}
func targetValue(target string, args []string) string {
	if target != "" {
		return target
	}
	return args[0]
}
