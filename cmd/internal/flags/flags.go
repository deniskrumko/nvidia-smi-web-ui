package flags

import "github.com/spf13/cobra"

// AddOutput adds common output flags to a command.
func AddOutput(command *cobra.Command, json *bool, warnings *bool) {
	command.Flags().BoolVar(json, "json", false, "render output as JSON")
	command.Flags().BoolVar(warnings, "warnings", false, "show per-metric NVML warnings")
}
