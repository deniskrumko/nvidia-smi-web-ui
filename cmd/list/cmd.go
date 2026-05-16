package list

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/deniskrumko/nvidia-smi-web-ui/app/gpu"
	"github.com/deniskrumko/nvidia-smi-web-ui/cmd/internal/flags"
	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/output"
)

// New creates the list command.
func New() *cobra.Command {
	var (
		asJSON      bool
		warnings    bool
		noProcesses bool
	)
	command := &cobra.Command{
		Use:   "list",
		Short: "List NVIDIA GPU devices",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return gpu.WithService(func(service *gpu.Service) error {
				snapshot, err := service.List(cmd.Context(), !noProcesses)
				if err != nil {
					return err
				}
				if asJSON {
					return output.WriteJSON(os.Stdout, snapshot)
				}
				if err := output.WriteDeviceTable(os.Stdout, snapshot); err != nil {
					return err
				}
				if warnings {
					return output.WriteWarnings(os.Stdout, snapshot.Warnings)
				}
				return nil
			})
		},
	}
	flags.AddOutput(command, &asJSON, &warnings)
	command.Flags().BoolVar(&noProcesses, "no-processes", false, "skip GPU process collection")
	return command
}
