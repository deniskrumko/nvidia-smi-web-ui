package gpu

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/gpuinfo"
)

const (
	debugGPUCountEnv     = "DEBUG_MODE_GPU_COUNT"
	defaultDebugGPUCount = 2
	gib                  = 1024 * 1024 * 1024
)

// DebugProvider returns synthetic GPU snapshots without initializing NVML.
type DebugProvider struct {
	now   func() time.Time
	count int
}

// NewDebugProvider creates a provider for local UI development without GPU access.
func NewDebugProvider() *DebugProvider {
	return newDebugProvider(time.Now, debugGPUCountFromEnv())
}

func newDebugProvider(now func() time.Time, count int) *DebugProvider {
	return &DebugProvider{now: now, count: count}
}

// List returns a synthetic GPU snapshot with values that change over time.
func (provider *DebugProvider) List(ctx context.Context, includeProcesses bool) (gpuinfo.Snapshot, error) {
	select {
	case <-ctx.Done():
		return gpuinfo.Snapshot{}, ctx.Err()
	default:
	}

	now := provider.now()
	driverVersion := "debug-driver"
	nvmlVersion := "debug-nvml-disabled"
	cudaVersion := "debug-cuda"

	devices := make([]gpuinfo.Device, 0, provider.count)
	for index := range provider.count {
		devices = append(devices, debugDevice(now, index, debugDeviceName(index), debugDeviceMemory(index), float64(index)*1.35, includeProcesses))
	}

	return gpuinfo.Snapshot{
		System: gpuinfo.SystemInfo{
			DriverVersion:     &driverVersion,
			NVMLVersion:       &nvmlVersion,
			CUDADriverVersion: &cudaVersion,
		},
		Devices: devices,
	}, nil
}

func debugGPUCountFromEnv() int {
	value := strings.TrimSpace(os.Getenv(debugGPUCountEnv))
	if value == "" {
		return defaultDebugGPUCount
	}

	count, err := strconv.Atoi(value)
	if err != nil || count < 0 {
		return defaultDebugGPUCount
	}
	return count
}

func debugDeviceName(index int) string {
	names := []string{"NVIDIA Debug RTX 4090", "NVIDIA Debug L40S"}
	return names[index%len(names)]
}

func debugDeviceMemory(index int) uint64 {
	totalMemory := []uint64{24 * gib, 48 * gib}
	return totalMemory[index%len(totalMemory)]
}

func debugDevice(now time.Time, index int, name string, totalMemory uint64, offset float64, includeProcesses bool) gpuinfo.Device {
	phase := float64(now.UnixMilli())/1000.0 + offset
	utilization := wave(phase, 8, 92)
	memoryUtilization := wave(phase*0.71+1.2, 18, 86)
	temperature := wave(phase*0.32+0.8, 38, 76)
	powerWatts := wave(phase*0.54+0.3, 82, 315)
	fanSpeed := wave(phase*0.44+1.6, 24, 78)
	usedMemory := uint64(float64(totalMemory) * float64(memoryUtilization) / 100)
	freeMemory := totalMemory - usedMemory
	uuid := fmt.Sprintf("GPU-debug-%d", index)
	brand := "NVIDIA"
	serial := fmt.Sprintf("DEBUG-%04d", index)
	boardID := uint32(1000 + index)
	architecture := "Ada Lovelace"
	computeCapability := "8.9"
	performanceState := fmt.Sprintf("P%d", int(wave(phase*0.21, 0, 4)))
	powerState := performanceState
	graphicsClock := uint32(900 + utilization*18)
	memoryClock := uint32(5000 + memoryUtilization*45)
	videoClock := uint32(1200 + utilization*8)
	busID := fmt.Sprintf("0000:%02d:00.0", 65+index)
	domain := uint32(0)
	bus := uint32(65 + index)
	pciDevice := uint32(0)
	pciDeviceID := uint32(0x2684 + index)
	pciSubsystemID := uint32(0x16F410DE + index)
	currentLinkGeneration := 4
	currentLinkWidth := 16
	maxLinkGeneration := 4
	maxLinkWidth := 16
	maxLinkSpeed := uint32(32000)
	replayCounter := int(wave(phase*0.08, 0, 12))
	eccMode := "Disabled"
	pendingStatus := "No"
	singleBitRetired := int(wave(phase*0.05, 0, 2))
	doubleBitRetired := 0

	device := gpuinfo.Device{
		Index:             &index,
		UUID:              &uuid,
		Name:              &name,
		Brand:             &brand,
		Serial:            &serial,
		BoardID:           &boardID,
		Architecture:      &architecture,
		ComputeCapability: &computeCapability,
		Memory: &gpuinfo.MemoryInfo{
			TotalBytes: totalMemory,
			FreeBytes:  freeMemory,
			UsedBytes:  usedMemory,
		},
		BAR1Memory: &gpuinfo.MemoryInfo{
			TotalBytes: totalMemory / 16,
			FreeBytes:  totalMemory / 24,
			UsedBytes:  totalMemory/16 - totalMemory/24,
		},
		Utilization: &gpuinfo.UtilizationInfo{
			GPUPercent:     ptr(utilization),
			MemoryPercent:  ptr(memoryUtilization),
			EncoderPercent: ptr(wave(phase*0.68+0.2, 0, 45)),
			DecoderPercent: ptr(wave(phase*0.61+0.7, 0, 52)),
			JPEGPercent:    ptr(wave(phase*0.47+0.5, 0, 28)),
			OFAPercent:     ptr(wave(phase*0.42+0.4, 0, 22)),
		},
		Temperature: &gpuinfo.TemperatureInfo{
			GPUCelsius:       ptr(temperature),
			ShutdownCelsius:  ptr(uint32(95)),
			SlowdownCelsius:  ptr(uint32(88)),
			MemMaxCelsius:    ptr(temperature + 4),
			GPUMaxCelsius:    ptr(uint32(83)),
			FanSpeedPercent:  ptr(fanSpeed),
			NumberOfFans:     ptr(2),
			PerformanceState: &performanceState,
		},
		Power: &gpuinfo.PowerInfo{
			UsageMilliwatts:         ptr(powerWatts * 1000),
			LimitMilliwatts:         ptr(uint32(350000)),
			DefaultLimitMilliwatts:  ptr(uint32(350000)),
			EnforcedLimitMilliwatts: ptr(uint32(350000)),
			MinLimitMilliwatts:      ptr(uint32(100000)),
			MaxLimitMilliwatts:      ptr(uint32(450000)),
			TotalEnergyMillijoules:  ptr(uint64(now.Unix()+int64(index*1000)) * 250000),
			PowerState:              &powerState,
		},
		Clocks: &gpuinfo.ClockInfo{
			GraphicsMHz:    &graphicsClock,
			SMMHz:          &graphicsClock,
			MemoryMHz:      &memoryClock,
			VideoMHz:       &videoClock,
			MaxGraphicsMHz: ptr(uint32(2520)),
			MaxSMMHz:       ptr(uint32(2520)),
			MaxMemoryMHz:   ptr(uint32(10501)),
			MaxVideoMHz:    ptr(uint32(1950)),
		},
		PCI: &gpuinfo.PCIInfo{
			BusID:                 &busID,
			Domain:                &domain,
			Bus:                   &bus,
			Device:                &pciDevice,
			PCIDeviceID:           &pciDeviceID,
			PCISubsystemID:        &pciSubsystemID,
			CurrentLinkGeneration: &currentLinkGeneration,
			CurrentLinkWidth:      &currentLinkWidth,
			MaxLinkGeneration:     &maxLinkGeneration,
			MaxLinkWidth:          &maxLinkWidth,
			MaxLinkSpeedMbps:      &maxLinkSpeed,
			ReplayCounter:         &replayCounter,
		},
		ECC: &gpuinfo.ECCInfo{
			CorrectedVolatile:    ptr(uint64(index)),
			UncorrectedVolatile:  ptr(uint64(0)),
			CorrectedAggregate:   ptr(uint64(index * 4)),
			UncorrectedAggregate: ptr(uint64(0)),
			CurrentMode:          &eccMode,
			PendingMode:          &eccMode,
		},
		RetiredPages: &gpuinfo.RetiredPagesInfo{
			SingleBitECCCount: &singleBitRetired,
			DoubleBitECCCount: &doubleBitRetired,
			PendingStatus:     &pendingStatus,
		},
	}

	if includeProcesses {
		device.Processes = []gpuinfo.Process{
			debugProcess(index, uuid, "compute", 4200+uint32(index), uint64(wave(phase, 384, 2048))*1024*1024),
			debugProcess(index, uuid, "graphics", 4300+uint32(index), uint64(wave(phase+1.1, 128, 1024))*1024*1024),
		}
	}

	return device
}

func debugProcess(index int, uuid string, processType string, pid uint32, usedMemory uint64) gpuinfo.Process {
	return gpuinfo.Process{
		DeviceIndex:  &index,
		DeviceUUID:   &uuid,
		Type:         processType,
		PID:          pid,
		UsedGPUBytes: &usedMemory,
	}
}

func wave(phase float64, minValue uint32, maxValue uint32) uint32 {
	normalized := (math.Sin(phase) + 1) / 2
	return minValue + uint32(math.Round(normalized*float64(maxValue-minValue)))
}

func ptr[T any](value T) *T {
	return &value
}
