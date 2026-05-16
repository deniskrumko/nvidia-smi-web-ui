package web

import (
	"fmt"
	"strings"

	webapp "github.com/deniskrumko/nvidia-smi-web-ui/app/web"
	"github.com/spf13/cobra"
)

// New creates the web command.
func New() *cobra.Command {
	var addr string
	command := &cobra.Command{
		Use:   "web",
		Short: "Run the local web UI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Serving web UI at http://%s\n", displayAddr(addr))
			if err != nil {
				return err
			}

			return webapp.Run(cmd.Context(), webapp.Config{Addr: addr})
		},
	}
	command.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	return command
}

func displayAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}
