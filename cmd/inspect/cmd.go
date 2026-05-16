package inspect

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/deniskrumko/nvidia-smi-web-ui/app/gpu"
	"github.com/deniskrumko/nvidia-smi-web-ui/cmd/internal/flags"
	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/gpuinfo"
	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/output"
)

// New creates the inspect command.
func New() *cobra.Command {
	var (
		asJSON      bool
		warnings    bool
		noProcesses bool
		id          int
		uuid        string
	)
	command := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect one NVIDIA GPU device",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if uuid == "" && id < 0 {
				return errors.New("provide --id or --uuid")
			}
			if uuid != "" && id >= 0 {
				return errors.New("provide either --id or --uuid, not both")
			}
			return gpu.WithService(func(service *gpu.Service) error {
				var (
					device gpuinfo.Device
					err    error
				)
				if uuid != "" {
					device, err = service.InspectByUUID(cmd.Context(), uuid, !noProcesses)
				} else {
					device, err = service.InspectByIndex(cmd.Context(), id, !noProcesses)
				}
				if err != nil {
					return err
				}
				if asJSON {
					return output.WriteJSON(os.Stdout, device)
				}
				if err := output.WriteDeviceDetails(os.Stdout, device); err != nil {
					return err
				}
				if warnings {
					return output.WriteWarnings(os.Stdout, device.Warnings)
				}
				return nil
			})
		},
	}
	flags.AddOutput(command, &asJSON, &warnings)
	command.Flags().BoolVar(&noProcesses, "no-processes", false, "skip GPU process collection")
	command.Flags().IntVar(&id, "id", -1, "NVML GPU index to inspect")
	command.Flags().StringVar(&uuid, "uuid", "", "NVML GPU UUID to inspect")
	return command
}
