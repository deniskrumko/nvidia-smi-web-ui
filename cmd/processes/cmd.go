package processes

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/deniskrumko/nvidia-smi-web-ui/app/gpu"
	"github.com/deniskrumko/nvidia-smi-web-ui/cmd/internal/flags"
	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/output"
)

// New creates the processes command.
func New() *cobra.Command {
	var (
		asJSON   bool
		warnings bool
	)
	command := &cobra.Command{
		Use:   "processes",
		Short: "List GPU processes reported by NVML",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return gpu.WithService(func(service *gpu.Service) error {
				snapshot, err := service.Processes(cmd.Context())
				if err != nil {
					return err
				}
				if asJSON {
					return output.WriteJSON(os.Stdout, snapshot)
				}
				if err := output.WriteProcessTable(os.Stdout, snapshot.Processes); err != nil {
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
	return command
}
