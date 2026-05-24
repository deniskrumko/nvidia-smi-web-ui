package nvmlclient

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/gpuinfo"
)

// Options controls the amount of data collected for a snapshot.
type Options struct {
	IncludeProcesses bool
}

// Client owns the NVML lifecycle and exposes hardware snapshots.
type Client struct {
	lib    library
	closed bool
}

type library interface {
	Init() nvml.Return
	Shutdown() nvml.Return
	ErrorString(nvml.Return) string
	SystemGetDriverVersion() (string, nvml.Return)
	SystemGetNVMLVersion() (string, nvml.Return)
	SystemGetCudaDriverVersion() (int, nvml.Return)
	DeviceGetCount() (int, nvml.Return)
	DeviceGetHandleByIndex(int) (device, nvml.Return)
	DeviceGetHandleByUUID(string) (device, nvml.Return)
}

type device interface {
	GetIndex() (int, nvml.Return)
	GetUUID() (string, nvml.Return)
	GetName() (string, nvml.Return)
	GetBrand() (nvml.BrandType, nvml.Return)
	GetSerial() (string, nvml.Return)
	GetBoardId() (uint32, nvml.Return)
	GetArchitecture() (nvml.DeviceArchitecture, nvml.Return)
	GetCudaComputeCapability() (int, int, nvml.Return)
	GetMemoryInfo() (nvml.Memory, nvml.Return)
	GetBAR1MemoryInfo() (nvml.BAR1Memory, nvml.Return)
	GetUtilizationRates() (nvml.Utilization, nvml.Return)
	GetEncoderUtilization() (uint32, uint32, nvml.Return)
	GetDecoderUtilization() (uint32, uint32, nvml.Return)
	GetJpgUtilization() (uint32, uint32, nvml.Return)
	GetOfaUtilization() (uint32, uint32, nvml.Return)
	GetTemperature(nvml.TemperatureSensors) (uint32, nvml.Return)
	GetTemperatureThreshold(nvml.TemperatureThresholds) (uint32, nvml.Return)
	GetFanSpeed() (uint32, nvml.Return)
	GetNumFans() (int, nvml.Return)
	GetPerformanceState() (nvml.Pstates, nvml.Return)
	GetPowerUsage() (uint32, nvml.Return)
	GetPowerManagementLimit() (uint32, nvml.Return)
	GetPowerManagementDefaultLimit() (uint32, nvml.Return)
	GetPowerManagementLimitConstraints() (uint32, uint32, nvml.Return)
	GetEnforcedPowerLimit() (uint32, nvml.Return)
	GetTotalEnergyConsumption() (uint64, nvml.Return)
	GetPowerState() (nvml.Pstates, nvml.Return)
	GetClockInfo(nvml.ClockType) (uint32, nvml.Return)
	GetMaxClockInfo(nvml.ClockType) (uint32, nvml.Return)
	GetCurrentClocksThrottleReasons() (uint64, nvml.Return)
	GetPciInfo() (nvml.PciInfo, nvml.Return)
	GetCurrPcieLinkGeneration() (int, nvml.Return)
	GetCurrPcieLinkWidth() (int, nvml.Return)
	GetMaxPcieLinkGeneration() (int, nvml.Return)
	GetMaxPcieLinkWidth() (int, nvml.Return)
	GetPcieLinkMaxSpeed() (uint32, nvml.Return)
	GetPcieReplayCounter() (int, nvml.Return)
	GetEccMode() (nvml.EnableState, nvml.EnableState, nvml.Return)
	GetTotalEccErrors(nvml.MemoryErrorType, nvml.EccCounterType) (uint64, nvml.Return)
	GetRetiredPages(nvml.PageRetirementCause) ([]uint64, nvml.Return)
	GetRetiredPagesPendingStatus() (nvml.EnableState, nvml.Return)
	GetComputeRunningProcesses() ([]nvml.ProcessInfo, nvml.Return)
	GetGraphicsRunningProcesses() ([]nvml.ProcessInfo, nvml.Return)
	GetMPSComputeRunningProcesses() ([]nvml.ProcessInfo, nvml.Return)
}

type realLibrary struct {
	lib nvml.Interface
}

// New initializes NVML and returns a client ready for data collection.
func New() (*Client, error) {
	return newWithLibrary(realLibrary{lib: nvml.New()})
}

func newWithLibrary(lib library) (*Client, error) {
	if ret := lib.Init(); ret != nvml.SUCCESS {
		return nil, fmt.Errorf("initialize NVML: %s", lib.ErrorString(ret))
	}

	return &Client{lib: lib}, nil
}

func (l realLibrary) Init() nvml.Return                  { return l.lib.Init() }
func (l realLibrary) Shutdown() nvml.Return              { return l.lib.Shutdown() }
func (l realLibrary) ErrorString(ret nvml.Return) string { return l.lib.ErrorString(ret) }
func (l realLibrary) SystemGetDriverVersion() (string, nvml.Return) {
	return l.lib.SystemGetDriverVersion()
}
func (l realLibrary) SystemGetNVMLVersion() (string, nvml.Return) {
	return l.lib.SystemGetNVMLVersion()
}
func (l realLibrary) SystemGetCudaDriverVersion() (int, nvml.Return) {
	return l.lib.SystemGetCudaDriverVersion()
}
func (l realLibrary) DeviceGetCount() (int, nvml.Return) { return l.lib.DeviceGetCount() }
func (l realLibrary) DeviceGetHandleByIndex(index int) (device, nvml.Return) {
	return l.lib.DeviceGetHandleByIndex(index)
}
func (l realLibrary) DeviceGetHandleByUUID(uuid string) (device, nvml.Return) {
	return l.lib.DeviceGetHandleByUUID(uuid)
}

// Close releases the NVML library handle.
func (c *Client) Close() error {
	if c == nil || c.closed {
		return nil
	}
	c.closed = true
	if ret := c.lib.Shutdown(); ret != nvml.SUCCESS {
		return fmt.Errorf("shutdown NVML: %s", c.lib.ErrorString(ret))
	}
	return nil
}

// Snapshot collects all visible GPU devices.
func (c *Client) Snapshot(ctx context.Context, opts Options) (gpuinfo.Snapshot, error) {
	system, warnings := c.systemInfo()
	snapshot := gpuinfo.Snapshot{
		System:   system,
		Warnings: warnings,
	}
	count, ret := c.lib.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return snapshot, fmt.Errorf("get device count: %s", c.lib.ErrorString(ret))
	}

	snapshot.Devices = make([]gpuinfo.Device, 0, count)
	for i := 0; i < count; i++ {
		if err := ctx.Err(); err != nil {
			return snapshot, err
		}
		handle, ret := c.lib.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			return snapshot, fmt.Errorf("get device %d: %s", i, c.lib.ErrorString(ret))
		}
		device := c.collectDevice(handle, opts)
		snapshot.Warnings = append(snapshot.Warnings, device.Warnings...)
		snapshot.Devices = append(snapshot.Devices, device)
	}

	return snapshot, nil
}

// InspectByIndex collects one GPU by NVML index.
func (c *Client) InspectByIndex(_ context.Context, index int, opts Options) (gpuinfo.Device, error) {
	handle, ret := c.lib.DeviceGetHandleByIndex(index)
	if ret != nvml.SUCCESS {
		return gpuinfo.Device{}, fmt.Errorf("get device %d: %s", index, c.lib.ErrorString(ret))
	}
	return c.collectDevice(handle, opts), nil
}

// InspectByUUID collects one GPU by NVML UUID.
func (c *Client) InspectByUUID(_ context.Context, uuid string, opts Options) (gpuinfo.Device, error) {
	handle, ret := c.lib.DeviceGetHandleByUUID(uuid)
	if ret != nvml.SUCCESS {
		return gpuinfo.Device{}, fmt.Errorf("get device %q: %s", uuid, c.lib.ErrorString(ret))
	}
	return c.collectDevice(handle, opts), nil
}

// Processes collects running GPU processes across all visible devices.
func (c *Client) Processes(ctx context.Context) (gpuinfo.ProcessSnapshot, error) {
	snapshot, err := c.Snapshot(ctx, Options{IncludeProcesses: true})
	if err != nil {
		return gpuinfo.ProcessSnapshot{}, err
	}

	processSnapshot := gpuinfo.ProcessSnapshot{
		Warnings: snapshot.Warnings,
	}
	for _, device := range snapshot.Devices {
		processSnapshot.Processes = append(processSnapshot.Processes, device.Processes...)
	}

	return processSnapshot, nil
}

func (c *Client) systemInfo() (gpuinfo.SystemInfo, []gpuinfo.Warning) {
	var info gpuinfo.SystemInfo
	var warnings []gpuinfo.Warning
	warn := func(field string, ret nvml.Return) {
		warnings = append(warnings, gpuinfo.Warning{
			Scope:   "system",
			Field:   field,
			Message: c.lib.ErrorString(ret),
		})
	}
	if value, err := os.Hostname(); err == nil && strings.TrimSpace(value) != "" {
		info.HostName = ptr(value)
	}
	if value, ret := c.lib.SystemGetDriverVersion(); ret == nvml.SUCCESS {
		info.DriverVersion = ptr(value)
	} else {
		warn("driver_version", ret)
	}
	if value, ret := c.lib.SystemGetNVMLVersion(); ret == nvml.SUCCESS {
		info.NVMLVersion = ptr(value)
	} else {
		warn("nvml_version", ret)
	}
	if value, ret := c.lib.SystemGetCudaDriverVersion(); ret == nvml.SUCCESS {
		info.CUDADriverVersion = ptr(formatCUDADriverVersion(value))
	} else {
		warn("cuda_driver_version", ret)
	}
	return info, warnings
}

func (c *Client) collectDevice(d device, opts Options) gpuinfo.Device {
	var result gpuinfo.Device

	warn := func(field string, ret nvml.Return) {
		result.Warnings = append(result.Warnings, gpuinfo.Warning{
			Scope:   deviceScope(result),
			Field:   field,
			Message: c.lib.ErrorString(ret),
		})
	}

	if value, ret := d.GetIndex(); ret == nvml.SUCCESS {
		result.Index = ptr(value)
	} else {
		warn("index", ret)
	}
	if value, ret := d.GetUUID(); ret == nvml.SUCCESS {
		result.UUID = ptr(value)
	} else {
		warn("uuid", ret)
	}
	if value, ret := d.GetName(); ret == nvml.SUCCESS {
		result.Name = ptr(value)
	} else {
		warn("name", ret)
	}
	if value, ret := d.GetBrand(); ret == nvml.SUCCESS {
		result.Brand = ptr(formatBrand(value))
	} else {
		warn("brand", ret)
	}
	if value, ret := d.GetSerial(); ret == nvml.SUCCESS {
		result.Serial = ptr(value)
	} else {
		warn("serial", ret)
	}
	if value, ret := d.GetBoardId(); ret == nvml.SUCCESS {
		result.BoardID = ptr(value)
	} else {
		warn("board_id", ret)
	}
	if value, ret := d.GetArchitecture(); ret == nvml.SUCCESS {
		result.Architecture = ptr(formatArchitecture(value))
	} else {
		warn("architecture", ret)
	}
	if major, minor, ret := d.GetCudaComputeCapability(); ret == nvml.SUCCESS {
		result.ComputeCapability = ptr(fmt.Sprintf("%d.%d", major, minor))
	} else {
		warn("compute_capability", ret)
	}

	result.Memory = c.collectMemory(d, warn)
	result.BAR1Memory = c.collectBAR1Memory(d, warn)
	result.Utilization = c.collectUtilization(d, warn)
	result.Temperature = c.collectTemperature(d, warn)
	result.Power = c.collectPower(d, warn)
	result.Clocks = c.collectClocks(d, warn)
	result.PCI = c.collectPCI(d, warn)
	result.ECC = c.collectECC(d, warn)
	result.RetiredPages = c.collectRetiredPages(d, warn)

	if opts.IncludeProcesses {
		result.Processes = c.collectProcesses(d, result, warn)
	}

	return result
}

func (c *Client) collectMemory(d device, warn func(string, nvml.Return)) *gpuinfo.MemoryInfo {
	value, ret := d.GetMemoryInfo()
	if ret != nvml.SUCCESS {
		warn("memory", ret)
		return nil
	}
	return &gpuinfo.MemoryInfo{TotalBytes: value.Total, FreeBytes: value.Free, UsedBytes: value.Used}
}

func (c *Client) collectBAR1Memory(d device, warn func(string, nvml.Return)) *gpuinfo.MemoryInfo {
	value, ret := d.GetBAR1MemoryInfo()
	if ret != nvml.SUCCESS {
		warn("bar1_memory", ret)
		return nil
	}
	return &gpuinfo.MemoryInfo{TotalBytes: value.Bar1Total, FreeBytes: value.Bar1Free, UsedBytes: value.Bar1Used}
}

func (c *Client) collectUtilization(d device, warn func(string, nvml.Return)) *gpuinfo.UtilizationInfo {
	var info gpuinfo.UtilizationInfo
	if value, ret := d.GetUtilizationRates(); ret == nvml.SUCCESS {
		info.GPUPercent = ptr(value.Gpu)
		info.MemoryPercent = ptr(value.Memory)
	} else {
		warn("utilization", ret)
	}
	if value, _, ret := d.GetEncoderUtilization(); ret == nvml.SUCCESS {
		info.EncoderPercent = ptr(value)
	} else {
		warn("encoder_utilization", ret)
	}
	if value, _, ret := d.GetDecoderUtilization(); ret == nvml.SUCCESS {
		info.DecoderPercent = ptr(value)
	} else {
		warn("decoder_utilization", ret)
	}
	if value, _, ret := d.GetJpgUtilization(); ret == nvml.SUCCESS {
		info.JPEGPercent = ptr(value)
	} else {
		warn("jpeg_utilization", ret)
	}
	if value, _, ret := d.GetOfaUtilization(); ret == nvml.SUCCESS {
		info.OFAPercent = ptr(value)
	} else {
		warn("ofa_utilization", ret)
	}
	return &info
}

func (c *Client) collectTemperature(d device, warn func(string, nvml.Return)) *gpuinfo.TemperatureInfo {
	var info gpuinfo.TemperatureInfo
	if value, ret := d.GetTemperature(nvml.TEMPERATURE_GPU); ret == nvml.SUCCESS {
		info.GPUCelsius = ptr(value)
	} else {
		warn("temperature_gpu", ret)
	}
	collectTempThreshold(d, nvml.TEMPERATURE_THRESHOLD_SHUTDOWN, "temperature_shutdown", &info.ShutdownCelsius, warn)
	collectTempThreshold(d, nvml.TEMPERATURE_THRESHOLD_SLOWDOWN, "temperature_slowdown", &info.SlowdownCelsius, warn)
	collectTempThreshold(d, nvml.TEMPERATURE_THRESHOLD_MEM_MAX, "temperature_memory_max", &info.MemMaxCelsius, warn)
	collectTempThreshold(d, nvml.TEMPERATURE_THRESHOLD_GPU_MAX, "temperature_gpu_max", &info.GPUMaxCelsius, warn)
	if value, ret := d.GetFanSpeed(); ret == nvml.SUCCESS {
		info.FanSpeedPercent = ptr(value)
	} else {
		warn("fan_speed", ret)
	}
	if value, ret := d.GetNumFans(); ret == nvml.SUCCESS {
		info.NumberOfFans = ptr(value)
	} else {
		warn("number_of_fans", ret)
	}
	if value, ret := d.GetPerformanceState(); ret == nvml.SUCCESS {
		info.PerformanceState = ptr(formatPState(value))
	} else {
		warn("performance_state", ret)
	}
	return &info
}

func collectTempThreshold(d device, threshold nvml.TemperatureThresholds, field string, dst **uint32, warn func(string, nvml.Return)) {
	if value, ret := d.GetTemperatureThreshold(threshold); ret == nvml.SUCCESS {
		*dst = ptr(value)
	} else {
		warn(field, ret)
	}
}

func (c *Client) collectPower(d device, warn func(string, nvml.Return)) *gpuinfo.PowerInfo {
	var info gpuinfo.PowerInfo
	if value, ret := d.GetPowerUsage(); ret == nvml.SUCCESS {
		info.UsageMilliwatts = ptr(value)
	} else {
		warn("power_usage", ret)
	}
	if value, ret := d.GetPowerManagementLimit(); ret == nvml.SUCCESS {
		info.LimitMilliwatts = ptr(value)
	} else {
		warn("power_limit", ret)
	}
	if value, ret := d.GetPowerManagementDefaultLimit(); ret == nvml.SUCCESS {
		info.DefaultLimitMilliwatts = ptr(value)
	} else {
		warn("power_default_limit", ret)
	}
	if value, ret := d.GetEnforcedPowerLimit(); ret == nvml.SUCCESS {
		info.EnforcedLimitMilliwatts = ptr(value)
	} else {
		warn("power_enforced_limit", ret)
	}
	if minValue, maxValue, ret := d.GetPowerManagementLimitConstraints(); ret == nvml.SUCCESS {
		info.MinLimitMilliwatts = ptr(minValue)
		info.MaxLimitMilliwatts = ptr(maxValue)
	} else {
		warn("power_limit_constraints", ret)
	}
	if value, ret := d.GetTotalEnergyConsumption(); ret == nvml.SUCCESS {
		info.TotalEnergyMillijoules = ptr(value)
	} else {
		warn("total_energy", ret)
	}
	if value, ret := d.GetPowerState(); ret == nvml.SUCCESS {
		info.PowerState = ptr(formatPState(value))
	} else {
		warn("power_state", ret)
	}
	return &info
}

func (c *Client) collectClocks(d device, warn func(string, nvml.Return)) *gpuinfo.ClockInfo {
	var info gpuinfo.ClockInfo
	collectClock(d, nvml.CLOCK_GRAPHICS, "clock_graphics", &info.GraphicsMHz, warn)
	collectClock(d, nvml.CLOCK_SM, "clock_sm", &info.SMMHz, warn)
	collectClock(d, nvml.CLOCK_MEM, "clock_memory", &info.MemoryMHz, warn)
	collectClock(d, nvml.CLOCK_VIDEO, "clock_video", &info.VideoMHz, warn)
	collectMaxClock(d, nvml.CLOCK_GRAPHICS, "max_clock_graphics", &info.MaxGraphicsMHz, warn)
	collectMaxClock(d, nvml.CLOCK_SM, "max_clock_sm", &info.MaxSMMHz, warn)
	collectMaxClock(d, nvml.CLOCK_MEM, "max_clock_memory", &info.MaxMemoryMHz, warn)
	collectMaxClock(d, nvml.CLOCK_VIDEO, "max_clock_video", &info.MaxVideoMHz, warn)
	if value, ret := d.GetCurrentClocksThrottleReasons(); ret == nvml.SUCCESS {
		info.ThrottleReasonsMask = ptr(value)
	} else {
		warn("clock_throttle_reasons", ret)
	}
	return &info
}

func collectClock(d device, clock nvml.ClockType, field string, dst **uint32, warn func(string, nvml.Return)) {
	if value, ret := d.GetClockInfo(clock); ret == nvml.SUCCESS {
		*dst = ptr(value)
	} else {
		warn(field, ret)
	}
}

func collectMaxClock(d device, clock nvml.ClockType, field string, dst **uint32, warn func(string, nvml.Return)) {
	if value, ret := d.GetMaxClockInfo(clock); ret == nvml.SUCCESS {
		*dst = ptr(value)
	} else {
		warn(field, ret)
	}
}

func (c *Client) collectPCI(d device, warn func(string, nvml.Return)) *gpuinfo.PCIInfo {
	var info gpuinfo.PCIInfo
	if value, ret := d.GetPciInfo(); ret == nvml.SUCCESS {
		info.BusID = ptr(cString(value.BusId[:]))
		info.Domain = ptr(value.Domain)
		info.Bus = ptr(value.Bus)
		info.Device = ptr(value.Device)
		info.PCIDeviceID = ptr(value.PciDeviceId)
		info.PCISubsystemID = ptr(value.PciSubSystemId)
	} else {
		warn("pci_info", ret)
	}
	if value, ret := d.GetCurrPcieLinkGeneration(); ret == nvml.SUCCESS {
		info.CurrentLinkGeneration = ptr(value)
	} else {
		warn("pcie_current_link_generation", ret)
	}
	if value, ret := d.GetCurrPcieLinkWidth(); ret == nvml.SUCCESS {
		info.CurrentLinkWidth = ptr(value)
	} else {
		warn("pcie_current_link_width", ret)
	}
	if value, ret := d.GetMaxPcieLinkGeneration(); ret == nvml.SUCCESS {
		info.MaxLinkGeneration = ptr(value)
	} else {
		warn("pcie_max_link_generation", ret)
	}
	if value, ret := d.GetMaxPcieLinkWidth(); ret == nvml.SUCCESS {
		info.MaxLinkWidth = ptr(value)
	} else {
		warn("pcie_max_link_width", ret)
	}
	if value, ret := d.GetPcieLinkMaxSpeed(); ret == nvml.SUCCESS {
		info.MaxLinkSpeedMbps = ptr(value)
	} else {
		warn("pcie_max_link_speed", ret)
	}
	if value, ret := d.GetPcieReplayCounter(); ret == nvml.SUCCESS {
		info.ReplayCounter = ptr(value)
	} else {
		warn("pcie_replay_counter", ret)
	}
	return &info
}

func (c *Client) collectECC(d device, warn func(string, nvml.Return)) *gpuinfo.ECCInfo {
	var info gpuinfo.ECCInfo
	if current, pending, ret := d.GetEccMode(); ret == nvml.SUCCESS {
		info.CurrentMode = ptr(formatEnableState(current))
		info.PendingMode = ptr(formatEnableState(pending))
	} else {
		warn("ecc_mode", ret)
	}
	collectECCCounter(d, nvml.MEMORY_ERROR_TYPE_CORRECTED, nvml.VOLATILE_ECC, "ecc_corrected_volatile", &info.CorrectedVolatile, warn)
	collectECCCounter(d, nvml.MEMORY_ERROR_TYPE_UNCORRECTED, nvml.VOLATILE_ECC, "ecc_uncorrected_volatile", &info.UncorrectedVolatile, warn)
	collectECCCounter(d, nvml.MEMORY_ERROR_TYPE_CORRECTED, nvml.AGGREGATE_ECC, "ecc_corrected_aggregate", &info.CorrectedAggregate, warn)
	collectECCCounter(d, nvml.MEMORY_ERROR_TYPE_UNCORRECTED, nvml.AGGREGATE_ECC, "ecc_uncorrected_aggregate", &info.UncorrectedAggregate, warn)
	return &info
}

func collectECCCounter(d device, errorType nvml.MemoryErrorType, counterType nvml.EccCounterType, field string, dst **uint64, warn func(string, nvml.Return)) {
	if value, ret := d.GetTotalEccErrors(errorType, counterType); ret == nvml.SUCCESS {
		*dst = ptr(value)
	} else {
		warn(field, ret)
	}
}

func (c *Client) collectRetiredPages(d device, warn func(string, nvml.Return)) *gpuinfo.RetiredPagesInfo {
	var info gpuinfo.RetiredPagesInfo
	if pages, ret := d.GetRetiredPages(nvml.PAGE_RETIREMENT_CAUSE_MULTIPLE_SINGLE_BIT_ECC_ERRORS); ret == nvml.SUCCESS {
		info.SingleBitECCCount = ptr(len(pages))
	} else {
		warn("retired_pages_single_bit_ecc", ret)
	}
	if pages, ret := d.GetRetiredPages(nvml.PAGE_RETIREMENT_CAUSE_DOUBLE_BIT_ECC_ERROR); ret == nvml.SUCCESS {
		info.DoubleBitECCCount = ptr(len(pages))
	} else {
		warn("retired_pages_double_bit_ecc", ret)
	}
	if value, ret := d.GetRetiredPagesPendingStatus(); ret == nvml.SUCCESS {
		info.PendingStatus = ptr(formatEnableState(value))
	} else {
		warn("retired_pages_pending_status", ret)
	}
	return &info
}

func (c *Client) collectProcesses(d device, gpu gpuinfo.Device, warn func(string, nvml.Return)) []gpuinfo.Process {
	var processes []gpuinfo.Process
	processes = append(processes, c.collectProcessKind(d, gpu, "compute", warn)...)
	processes = append(processes, c.collectProcessKind(d, gpu, "graphics", warn)...)
	processes = append(processes, c.collectProcessKind(d, gpu, "mps_compute", warn)...)
	return processes
}

func (c *Client) collectProcessKind(d device, gpu gpuinfo.Device, processType string, warn func(string, nvml.Return)) []gpuinfo.Process {
	var (
		raw []nvml.ProcessInfo
		ret nvml.Return
	)
	switch processType {
	case "compute":
		raw, ret = d.GetComputeRunningProcesses()
	case "graphics":
		raw, ret = d.GetGraphicsRunningProcesses()
	case "mps_compute":
		raw, ret = d.GetMPSComputeRunningProcesses()
	default:
		return nil
	}
	if ret != nvml.SUCCESS {
		warn("processes_"+processType, ret)
		return nil
	}

	processes := make([]gpuinfo.Process, 0, len(raw))
	for _, p := range raw {
		processes = append(processes, gpuinfo.Process{
			DeviceIndex:       gpu.Index,
			DeviceUUID:        gpu.UUID,
			Type:              processType,
			PID:               p.Pid,
			UsedGPUBytes:      processMemoryPointer(p.UsedGpuMemory),
			GPUInstanceID:     invalidUint32Pointer(p.GpuInstanceId),
			ComputeInstanceID: invalidUint32Pointer(p.ComputeInstanceId),
		})
	}
	return processes
}

func processMemoryPointer(value uint64) *uint64 {
	if value == ^uint64(0) {
		return nil
	}
	return ptr(value)
}

func invalidUint32Pointer(value uint32) *uint32 {
	if value == ^uint32(0) {
		return nil
	}
	return ptr(value)
}

func formatCUDADriverVersion(version int) string {
	major := version / 1000
	minor := (version % 1000) / 10
	return fmt.Sprintf("%d.%d", major, minor)
}

func formatEnableState(state nvml.EnableState) string {
	switch state {
	case nvml.FEATURE_ENABLED:
		return "enabled"
	case nvml.FEATURE_DISABLED:
		return "disabled"
	default:
		return fmt.Sprint(state)
	}
}

func formatPState(state nvml.Pstates) string {
	if state == nvml.PSTATE_UNKNOWN {
		return "unknown"
	}
	if state >= nvml.PSTATE_0 && state <= nvml.PSTATE_15 {
		return fmt.Sprintf("P%d", int(state))
	}
	return fmt.Sprint(state)
}

func formatBrand(brand nvml.BrandType) string {
	switch brand {
	case nvml.BRAND_QUADRO:
		return "Quadro"
	case nvml.BRAND_TESLA:
		return "Tesla"
	case nvml.BRAND_NVS:
		return "NVS"
	case nvml.BRAND_GRID:
		return "GRID"
	case nvml.BRAND_GEFORCE:
		return "GeForce"
	case nvml.BRAND_TITAN:
		return "Titan"
	case nvml.BRAND_NVIDIA_VAPPS:
		return "NVIDIA Virtual Applications"
	case nvml.BRAND_NVIDIA_VPC:
		return "NVIDIA Virtual PC"
	case nvml.BRAND_NVIDIA_VCS:
		return "NVIDIA Virtual Compute Server"
	case nvml.BRAND_NVIDIA_VWS:
		return "NVIDIA RTX Virtual Workstation"
	case nvml.BRAND_NVIDIA_CLOUD_GAMING:
		return "NVIDIA Cloud Gaming"
	case nvml.BRAND_QUADRO_RTX:
		return "Quadro RTX"
	case nvml.BRAND_NVIDIA_RTX:
		return "NVIDIA RTX"
	case nvml.BRAND_NVIDIA:
		return "NVIDIA"
	case nvml.BRAND_GEFORCE_RTX:
		return "GeForce RTX"
	case nvml.BRAND_TITAN_RTX:
		return "Titan RTX"
	case nvml.BRAND_UNKNOWN:
		return "unknown"
	default:
		return fmt.Sprint(brand)
	}
}

func formatArchitecture(architecture nvml.DeviceArchitecture) string {
	switch architecture {
	case nvml.DEVICE_ARCH_KEPLER:
		return "Kepler"
	case nvml.DEVICE_ARCH_MAXWELL:
		return "Maxwell"
	case nvml.DEVICE_ARCH_PASCAL:
		return "Pascal"
	case nvml.DEVICE_ARCH_VOLTA:
		return "Volta"
	case nvml.DEVICE_ARCH_TURING:
		return "Turing"
	case nvml.DEVICE_ARCH_AMPERE:
		return "Ampere"
	case nvml.DEVICE_ARCH_ADA:
		return "Ada"
	case nvml.DEVICE_ARCH_HOPPER:
		return "Hopper"
	case nvml.DEVICE_ARCH_BLACKWELL:
		return "Blackwell"
	case nvml.DEVICE_ARCH_UNKNOWN:
		return "unknown"
	default:
		return fmt.Sprint(architecture)
	}
}

func deviceScope(device gpuinfo.Device) string {
	if device.UUID != nil && *device.UUID != "" {
		return "device:" + *device.UUID
	}
	if device.Index != nil {
		return fmt.Sprintf("device:%d", *device.Index)
	}
	return "device"
}

func cString(bytes []uint8) string {
	end := 0
	for end < len(bytes) && bytes[end] != 0 {
		end++
	}
	return strings.TrimSpace(string(bytes[:end]))
}

func ptr[T any](value T) *T {
	return &value
}
