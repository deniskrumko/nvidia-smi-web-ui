package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/gpuinfo"
)

func TestWriteDeviceTable(t *testing.T) {
	snapshot := gpuinfo.Snapshot{
		Devices: []gpuinfo.Device{
			{
				Index: ptr(0),
				Name:  ptr("NVIDIA Test GPU"),
				UUID:  ptr("GPU-0"),
				Memory: &gpuinfo.MemoryInfo{
					TotalBytes: 1024,
					UsedBytes:  512,
				},
				Utilization: &gpuinfo.UtilizationInfo{
					GPUPercent:    ptr(uint32(50)),
					MemoryPercent: ptr(uint32(25)),
				},
				Temperature: &gpuinfo.TemperatureInfo{
					GPUCelsius:      ptr(uint32(55)),
					FanSpeedPercent: ptr(uint32(40)),
				},
				Power: &gpuinfo.PowerInfo{
					UsageMilliwatts: ptr(uint32(100000)),
					LimitMilliwatts: ptr(uint32(250000)),
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteDeviceTable(&buf, snapshot); err != nil {
		t.Fatalf("write table: %v", err)
	}
	output := buf.String()
	for _, want := range []string{"ID", "NVIDIA Test GPU", "GPU-0", "50%", "100.0 W / 250.0 W"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in output:\n%s", want, output)
		}
	}
}

func TestWriteDeviceTableMissingFields(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteDeviceTable(&buf, gpuinfo.Snapshot{Devices: []gpuinfo.Device{{}}}); err != nil {
		t.Fatalf("write table: %v", err)
	}
	if !strings.Contains(buf.String(), "-") {
		t.Fatalf("expected placeholder in output:\n%s", buf.String())
	}
}

func TestWriteJSONStableFields(t *testing.T) {
	var buf bytes.Buffer
	err := WriteJSON(&buf, gpuinfo.Snapshot{
		System: gpuinfo.SystemInfo{DriverVersion: ptr("580.1")},
	})
	if err != nil {
		t.Fatalf("write json: %v", err)
	}
	output := buf.String()
	for _, want := range []string{`"system"`, `"driver_version"`, `"devices"`} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in output:\n%s", want, output)
		}
	}
}

func TestWriteProcessTable(t *testing.T) {
	var buf bytes.Buffer
	err := WriteProcessTable(&buf, []gpuinfo.Process{{
		DeviceIndex:  ptr(0),
		DeviceUUID:   ptr("GPU-0"),
		Type:         "compute",
		PID:          123,
		UsedGPUBytes: ptr(uint64(1024)),
	}})
	if err != nil {
		t.Fatalf("write process table: %v", err)
	}
	output := buf.String()
	for _, want := range []string{"compute", "123", "1.0 KiB"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in output:\n%s", want, output)
		}
	}
}

func ptr[T any](value T) *T {
	return &value
}
