package gpu

import (
	"context"
	"testing"
	"time"
)

func TestDebugProviderReturnsSyntheticSnapshotWithoutProcesses(t *testing.T) {
	provider := newDebugProvider(func() time.Time {
		return time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	})

	snapshot, err := provider.List(context.Background(), false)
	if err != nil {
		t.Fatalf("list debug snapshot: %v", err)
	}
	if got := len(snapshot.Devices); got != 2 {
		t.Fatalf("expected 2 debug devices, got %d", got)
	}
	if snapshot.System.NVMLVersion == nil || *snapshot.System.NVMLVersion != "debug-nvml-disabled" {
		t.Fatalf("expected debug NVML system value, got %#v", snapshot.System.NVMLVersion)
	}

	device := snapshot.Devices[0]
	if device.Name == nil || *device.Name != "NVIDIA Debug RTX 4090" {
		t.Fatalf("unexpected device name: %#v", device.Name)
	}
	if device.Memory == nil || device.Memory.TotalBytes != 24*gib {
		t.Fatalf("unexpected device memory: %#v", device.Memory)
	}
	if device.Utilization == nil || device.Utilization.GPUPercent == nil {
		t.Fatalf("expected utilization in debug snapshot: %#v", device.Utilization)
	}
	if len(device.Processes) != 0 {
		t.Fatalf("expected no processes, got %#v", device.Processes)
	}
}

func TestDebugProviderIncludesProcessesWhenRequested(t *testing.T) {
	provider := newDebugProvider(func() time.Time {
		return time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	})

	snapshot, err := provider.List(context.Background(), true)
	if err != nil {
		t.Fatalf("list debug snapshot: %v", err)
	}
	if got := len(snapshot.Devices[0].Processes); got != 2 {
		t.Fatalf("expected 2 debug processes, got %d", got)
	}
}

func TestDebugProviderReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewDebugProvider().List(ctx, false)
	if err != context.Canceled {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
