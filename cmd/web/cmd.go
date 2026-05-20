package web

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/deniskrumko/nvidia-smi-web-ui/app/gpu"
	webapp "github.com/deniskrumko/nvidia-smi-web-ui/app/web"
	"github.com/spf13/cobra"
)

const debugModeEnv = "DEBUG_MODE_ENABLED"
const accessLogLevelEnv = "LOG_ACCESS_LOG_LEVEL"
const accessLogEnv = "LOG_ACCESS_LOG_ENABLED"
const uiBrandingEnv = "UI_BRANDING"
const uiTitleEnv = "UI_TITLE"

// New creates the web command.
func New() *cobra.Command {
	var addr string
	command := &cobra.Command{
		Use:   "web",
		Short: "Run the local web UI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configureLogger()
			accessLogLevel, err := accessLogLevelFromEnv(os.Getenv(accessLogLevelEnv))
			if err != nil {
				return err
			}

			remoteHosts, err := remoteHostsFromEnv()
			if err != nil {
				return err
			}
			if len(remoteHosts) > 0 {
				return run(cmd, addr, nil, remoteHosts, accessLogLevel, " with remote GPU hosts")
			}

			if debugEnabled() {
				return run(cmd, addr, gpu.NewDebugProvider(), nil, accessLogLevel, " with debug GPU data and NVML disabled")
			}

			return gpu.WithService(func(service *gpu.Service) error {
				return run(cmd, addr, service, nil, accessLogLevel, "")
			})
		},
	}
	command.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	return command
}

func run(cmd *cobra.Command, addr string, provider webapp.SnapshotProvider, remoteHosts []webapp.RemoteHost, accessLogLevel slog.Level, suffix string) error {
	slog.InfoContext(cmd.Context(), "Web server started",
		"url", "http://"+displayAddr(addr),
		"mode", servingMode(suffix),
	)

	return webapp.Run(cmd.Context(), webapp.Config{
		Addr:             addr,
		SnapshotProvider: provider,
		RemoteHosts:      remoteHosts,
		DisableAccessLog: !accessLogEnabled(),
		AccessLogLevel:   accessLogLevel,
		Branding:         os.Getenv(uiBrandingEnv),
		Title:            pageTitle(),
	})
}

func debugEnabled() bool {
	return truthy(os.Getenv(debugModeEnv))
}

func accessLogEnabled() bool {
	value := strings.TrimSpace(os.Getenv(accessLogEnv))
	if value == "" {
		return false
	}
	return truthy(value)
}

func configureLogger() {
	slog.SetDefault(slog.New(newLogHandler(os.Stderr)))
}

func newLogHandler(writer io.Writer) slog.Handler {
	return slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug})
}

func accessLogLevelFromEnv(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("%s must be one of debug, info, warn, or error", accessLogLevelEnv)
	}
}

func servingMode(suffix string) string {
	mode := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(suffix), "with "))
	if mode == "" {
		return "local GPU data"
	}
	return mode
}

func displayAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}

func pageTitle() string {
	if title := os.Getenv(uiTitleEnv); title != "" {
		return title
	}
	return os.Getenv(uiBrandingEnv)
}
