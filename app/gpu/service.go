package gpu

import (
	"context"

	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/gpuinfo"
	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/nvmlclient"
)

// Service coordinates GPU use cases for CLI and future API transports.
type Service struct {
	client *nvmlclient.Client
}

// NewService creates a GPU service.
func NewService(client *nvmlclient.Client) *Service {
	return &Service{client: client}
}

// List returns a full GPU snapshot.
func (s *Service) List(ctx context.Context, includeProcesses bool) (gpuinfo.Snapshot, error) {
	return s.client.Snapshot(ctx, nvmlclient.Options{IncludeProcesses: includeProcesses})
}

// InspectByIndex returns one GPU snapshot by NVML index.
func (s *Service) InspectByIndex(ctx context.Context, index int, includeProcesses bool) (gpuinfo.Device, error) {
	return s.client.InspectByIndex(ctx, index, nvmlclient.Options{IncludeProcesses: includeProcesses})
}

// InspectByUUID returns one GPU snapshot by NVML UUID.
func (s *Service) InspectByUUID(ctx context.Context, uuid string, includeProcesses bool) (gpuinfo.Device, error) {
	return s.client.InspectByUUID(ctx, uuid, nvmlclient.Options{IncludeProcesses: includeProcesses})
}

// Processes returns all GPU processes across visible devices.
func (s *Service) Processes(ctx context.Context) (gpuinfo.ProcessSnapshot, error) {
	return s.client.Processes(ctx)
}
