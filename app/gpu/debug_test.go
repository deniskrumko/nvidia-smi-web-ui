package gpu

import (
	"context"
	"testing"
	"time"
)

func TestDebugProviderReturnsSyntheticSnapshotWithoutProcesses(t *testing.T) {
	provider := newDebugProvider(func() time.Time {
		return time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	}, 2)

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
	}, 2)

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

func TestDebugProviderUsesGPUCount(t *testing.T) {
	provider := newDebugProvider(func() time.Time {
		return time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	}, 8)

	snapshot, err := provider.List(context.Background(), false)
	if err != nil {
		t.Fatalf("list debug snapshot: %v", err)
	}
	if got := len(snapshot.Devices); got != 8 {
		t.Fatalf("expected 8 debug devices, got %d", got)
	}
}

func TestDebugGPUCountFromEnv(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "empty", value: "", want: defaultDebugGPUCount},
		{name: "custom", value: "8", want: 8},
		{name: "zero", value: "0", want: 0},
		{name: "invalid", value: "many", want: defaultDebugGPUCount},
		{name: "negative", value: "-1", want: defaultDebugGPUCount},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(debugGPUCountEnv, test.value)

			if got := debugGPUCountFromEnv(); got != test.want {
				t.Fatalf("expected %d, got %d", test.want, got)
			}
		})
	}
}
