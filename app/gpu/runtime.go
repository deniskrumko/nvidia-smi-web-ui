package gpu

import "github.com/deniskrumko/nvidia-smi-web-ui/pkg/nvmlclient"

// WithService initializes NVML, runs fn, and always attempts to close NVML.
func WithService(fn func(*Service) error) error {
	client, err := nvmlclient.New()
	if err != nil {
		return err
	}

	runErr := fn(NewService(client))
	closeErr := client.Close()
	if runErr != nil {
		return runErr
	}
	return closeErr
}
