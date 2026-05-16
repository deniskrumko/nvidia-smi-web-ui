package web

import (
	"fmt"
	"os"
	"strings"

	"github.com/deniskrumko/nvidia-smi-web-ui/app/gpu"
	webapp "github.com/deniskrumko/nvidia-smi-web-ui/app/web"
	"github.com/spf13/cobra"
)

const debugEnv = "NVIDIA_SMI_WEB_UI_DEBUG"

// New creates the web command.
func New() *cobra.Command {
	var addr string
	command := &cobra.Command{
		Use:   "web",
		Short: "Run the local web UI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if debugEnabled() {
				return run(cmd, addr, gpu.NewDebugProvider(), " with debug GPU data and NVML disabled")
			}

			return gpu.WithService(func(service *gpu.Service) error {
				return run(cmd, addr, service, "")
			})
		},
	}
	command.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	return command
}

func run(cmd *cobra.Command, addr string, provider webapp.SnapshotProvider, suffix string) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "Serving web UI at http://%s%s\n", displayAddr(addr), suffix)
	if err != nil {
		return err
	}

	return webapp.Run(cmd.Context(), webapp.Config{
		Addr:             addr,
		SnapshotProvider: provider,
		Branding:         os.Getenv("WEB_PAGE_BRANDING"),
		Title:            pageTitle(),
	})
}

func debugEnabled() bool {
	value := strings.TrimSpace(os.Getenv(debugEnv))
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

func displayAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}

func pageTitle() string {
	if title := os.Getenv("WEB_PAGE_TITLE"); title != "" {
		return title
	}
	return os.Getenv("WEB_PAGE_BRANDING")
}
