package cli

import (
	"strings"

	"github.com/raywcm/taskdock/internal/config"
	"github.com/raywcm/taskdock/internal/i18n"
	"github.com/raywcm/taskdock/internal/store"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func configureLocalization(root *cobra.Command, stateDir *string) {
	shorts := map[*cobra.Command]string{}
	usages := map[*pflag.Flag]string{}
	englishUsage := root.UsageTemplate()
	localize := func() {
		settings := config.Defaults()
		if dir, err := config.StateDir(*stateDir); err == nil {
			if loaded, err := store.ReadSettings(dir); err == nil {
				settings = loaded
			}
		}
		locale := i18n.New(settings.Language)
		var walk func(*cobra.Command)
		walk = func(cmd *cobra.Command) {
			if _, ok := shorts[cmd]; !ok {
				shorts[cmd] = cmd.Short
			}
			cmd.Short = locale.CLI(shorts[cmd])
			translate := func(flag *pflag.Flag) {
				if _, ok := usages[flag]; !ok {
					usages[flag] = flag.Usage
				}
				flag.Usage = locale.CLI(usages[flag])
				if strings.HasPrefix(usages[flag], "help for ") {
					flag.Usage = locale.Text("cli_help_flag")
				}
				if strings.HasPrefix(usages[flag], "version for ") {
					flag.Usage = locale.Text("cli_version_flag")
				}
			}
			cmd.Flags().VisitAll(translate)
			cmd.PersistentFlags().VisitAll(translate)
			for _, child := range cmd.Commands() {
				walk(child)
			}
		}
		walk(root)
		root.Short = "TaskDock · " + locale.CLI("Independent terminal task manager")
		usage := englishUsage
		usage = strings.NewReplacer("Usage:", locale.Text("cli_usage"), "Aliases:", locale.Text("cli_aliases"), "Available Commands:", locale.Text("cli_commands"), "Additional Commands:", locale.Text("cli_additional_commands"), "Global Flags:", locale.Text("cli_global_flags"), "Flags:", locale.Text("cli_flags"), "Use ", locale.Text("cli_use"), " for more information about a command.", locale.Text("cli_more_info")).Replace(usage)
		root.SetUsageTemplate(usage)
	}
	originalHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) { localize(); originalHelp(cmd, args) })
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) { localize() }
}
