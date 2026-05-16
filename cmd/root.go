package cmd

import (
	"context"

	inspectcmd "github.com/deniskrumko/nvidia-smi-web-ui/cmd/inspect"
	listcmd "github.com/deniskrumko/nvidia-smi-web-ui/cmd/list"
	processescmd "github.com/deniskrumko/nvidia-smi-web-ui/cmd/processes"
	webcmd "github.com/deniskrumko/nvidia-smi-web-ui/cmd/web"
	"github.com/spf13/cobra"
)

// Execute runs the root command.
func Execute(ctx context.Context) error {
	root := &cobra.Command{
		Use:           "nvidia-smi-web-ui",
		Short:         "Inspect NVIDIA GPUs through NVML",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.CompletionOptions.HiddenDefaultCmd = true
	root.AddCommand(listcmd.New(), inspectcmd.New(), processescmd.New(), webcmd.New())
	return root.ExecuteContext(ctx)
}
