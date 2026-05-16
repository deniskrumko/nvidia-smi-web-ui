package nvmlclient

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/gpuinfo"
)

func TestNewWithLibraryInitFailure(t *testing.T) {
	_, err := newWithLibrary(&fakeLibrary{initRet: nvml.ERROR_UNINITIALIZED})
	if err == nil {
		t.Fatal("expected init error")
	}
	if !strings.Contains(err.Error(), "initialize NVML") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSnapshotSuccess(t *testing.T) {
	client := mustFakeClient(t, &fakeLibrary{
		driverVersion: "580.1",
		nvmlVersion:   "13.0",
		cudaVersion:   13000,
		devices: []device{
			newFakeDevice(0, "GPU-0"),
			newFakeDevice(1, "GPU-1"),
		},
	})

	snapshot, err := client.Snapshot(context.Background(), Options{IncludeProcesses: true})
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if len(snapshot.Devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(snapshot.Devices))
	}
	if got := *snapshot.System.CUDADriverVersion; got != "13.0" {
		t.Fatalf("unexpected CUDA version: %s", got)
	}
	if got := snapshot.Devices[0].Memory.UsedBytes; got == 0 {
		t.Fatal("expected memory usage to be collected")
	}
	if len(snapshot.Devices[0].Processes) != 3 {
		t.Fatalf("expected 3 process kinds, got %d", len(snapshot.Devices[0].Processes))
	}
}

func TestSnapshotEmpty(t *testing.T) {
	client := mustFakeClient(t, &fakeLibrary{})

	snapshot, err := client.Snapshot(context.Background(), Options{})
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if len(snapshot.Devices) != 0 {
		t.Fatalf("expected no devices, got %d", len(snapshot.Devices))
	}
}

func TestSnapshotHandleFailure(t *testing.T) {
	client := mustFakeClient(t, &fakeLibrary{
		devices:   []device{newFakeDevice(0, "GPU-0")},
		handleRet: nvml.ERROR_INVALID_ARGUMENT,
	})

	_, err := client.Snapshot(context.Background(), Options{})
	if err == nil {
		t.Fatal("expected handle error")
	}
	if !strings.Contains(err.Error(), "get device 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOptionalMetricFailureProducesWarning(t *testing.T) {
	fake := newFakeDevice(0, "GPU-0")
	fake.fail["fan_speed"] = nvml.ERROR_NOT_SUPPORTED
	client := mustFakeClient(t, &fakeLibrary{devices: []device{fake}})

	snapshot, err := client.Snapshot(context.Background(), Options{})
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if snapshot.Devices[0].Memory == nil {
		t.Fatal("expected other metrics to be preserved")
	}
	assertWarning(t, snapshot.Devices[0].Warnings, "fan_speed")
}

func TestSystemMetricFailureProducesWarning(t *testing.T) {
	client := mustFakeClient(t, &fakeLibrary{
		systemFailures: map[string]nvml.Return{
			"driver_version": nvml.ERROR_NOT_SUPPORTED,
		},
		devices: []device{newFakeDevice(0, "GPU-0")},
	})

	snapshot, err := client.Snapshot(context.Background(), Options{})
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	assertWarning(t, snapshot.Warnings, "driver_version")
}

func TestProcessesCollectsAllKinds(t *testing.T) {
	client := mustFakeClient(t, &fakeLibrary{devices: []device{newFakeDevice(0, "GPU-0")}})

	snapshot, err := client.Processes(context.Background())
	if err != nil {
		t.Fatalf("processes failed: %v", err)
	}
	if len(snapshot.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %+v", snapshot.Warnings)
	}
	if len(snapshot.Processes) != 3 {
		t.Fatalf("expected 3 processes, got %d", len(snapshot.Processes))
	}
	gotTypes := map[string]bool{}
	for _, process := range snapshot.Processes {
		gotTypes[process.Type] = true
	}
	for _, want := range []string{"compute", "graphics", "mps_compute"} {
		if !gotTypes[want] {
			t.Fatalf("missing process type %q in %+v", want, snapshot.Processes)
		}
	}
}

func TestIntegrationNVML(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped in short mode")
	}
	if value := os.Getenv("NVML_INTEGRATION"); value != "1" {
		t.Skip("set NVML_INTEGRATION=1 on a Linux GPU host to run this test")
	}
	client, err := New()
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	}()
	if _, err := client.Snapshot(context.Background(), Options{}); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
}

func assertWarning(t *testing.T, warnings []gpuinfo.Warning, field string) {
	t.Helper()
	for _, warning := range warnings {
		if warning.Field == field {
			return
		}
	}
	t.Fatalf("expected warning for %q, got %+v", field, warnings)
}

func mustFakeClient(t *testing.T, lib *fakeLibrary) *Client {
	t.Helper()
	client, err := newWithLibrary(lib)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client
}

type fakeLibrary struct {
	initRet        nvml.Return
	shutdownRet    nvml.Return
	driverVersion  string
	nvmlVersion    string
	cudaVersion    int
	systemFailures map[string]nvml.Return
	devices        []device
	handleRet      nvml.Return
}

func (l *fakeLibrary) Init() nvml.Return {
	if l.initRet != nvml.SUCCESS {
		return l.initRet
	}
	return nvml.SUCCESS
}

func (l *fakeLibrary) Shutdown() nvml.Return {
	if l.shutdownRet != nvml.SUCCESS {
		return l.shutdownRet
	}
	return nvml.SUCCESS
}

func (l *fakeLibrary) ErrorString(ret nvml.Return) string { return ret.String() }
func (l *fakeLibrary) SystemGetDriverVersion() (string, nvml.Return) {
	if ret, ok := l.systemFailures["driver_version"]; ok {
		return "", ret
	}
	return l.driverVersion, nvml.SUCCESS
}
func (l *fakeLibrary) SystemGetNVMLVersion() (string, nvml.Return) {
	if ret, ok := l.systemFailures["nvml_version"]; ok {
		return "", ret
	}
	return l.nvmlVersion, nvml.SUCCESS
}
func (l *fakeLibrary) SystemGetCudaDriverVersion() (int, nvml.Return) {
	if ret, ok := l.systemFailures["cuda_driver_version"]; ok {
		return 0, ret
	}
	return l.cudaVersion, nvml.SUCCESS
}
func (l *fakeLibrary) DeviceGetCount() (int, nvml.Return) {
	return len(l.devices), nvml.SUCCESS
}
func (l *fakeLibrary) DeviceGetHandleByIndex(index int) (device, nvml.Return) {
	if l.handleRet != nvml.SUCCESS {
		return nil, l.handleRet
	}
	if index < 0 || index >= len(l.devices) {
		return nil, nvml.ERROR_INVALID_ARGUMENT
	}
	return l.devices[index], nvml.SUCCESS
}
func (l *fakeLibrary) DeviceGetHandleByUUID(uuid string) (device, nvml.Return) {
	for _, dev := range l.devices {
		value, _ := dev.GetUUID()
		if value == uuid {
			return dev, nvml.SUCCESS
		}
	}
	return nil, nvml.ERROR_INVALID_ARGUMENT
}

type fakeDevice struct {
	index int
	uuid  string
	fail  map[string]nvml.Return
}

func newFakeDevice(index int, uuid string) *fakeDevice {
	return &fakeDevice{index: index, uuid: uuid, fail: map[string]nvml.Return{}}
}

func (d *fakeDevice) ret(field string) nvml.Return {
	if ret, ok := d.fail[field]; ok {
		return ret
	}
	return nvml.SUCCESS
}

func (d *fakeDevice) GetIndex() (int, nvml.Return)   { return d.index, d.ret("index") }
func (d *fakeDevice) GetUUID() (string, nvml.Return) { return d.uuid, d.ret("uuid") }
func (d *fakeDevice) GetName() (string, nvml.Return) {
	return "NVIDIA Test GPU", d.ret("name")
}
func (d *fakeDevice) GetBrand() (nvml.BrandType, nvml.Return) {
	return nvml.BRAND_NVIDIA, d.ret("brand")
}
func (d *fakeDevice) GetSerial() (string, nvml.Return) { return "serial", d.ret("serial") }
func (d *fakeDevice) GetBoardId() (uint32, nvml.Return) {
	return 42, d.ret("board_id")
}
func (d *fakeDevice) GetArchitecture() (nvml.DeviceArchitecture, nvml.Return) {
	return nvml.DEVICE_ARCH_AMPERE, d.ret("architecture")
}
func (d *fakeDevice) GetCudaComputeCapability() (int, int, nvml.Return) {
	return 8, 0, d.ret("compute_capability")
}
func (d *fakeDevice) GetMemoryInfo() (nvml.Memory, nvml.Return) {
	return nvml.Memory{Total: 1024, Free: 256, Used: 768}, d.ret("memory")
}
func (d *fakeDevice) GetBAR1MemoryInfo() (nvml.BAR1Memory, nvml.Return) {
	return nvml.BAR1Memory{Bar1Total: 512, Bar1Free: 128, Bar1Used: 384}, d.ret("bar1_memory")
}
func (d *fakeDevice) GetUtilizationRates() (nvml.Utilization, nvml.Return) {
	return nvml.Utilization{Gpu: 50, Memory: 20}, d.ret("utilization")
}
func (d *fakeDevice) GetEncoderUtilization() (uint32, uint32, nvml.Return) {
	return 10, 1000, d.ret("encoder_utilization")
}
func (d *fakeDevice) GetDecoderUtilization() (uint32, uint32, nvml.Return) {
	return 11, 1000, d.ret("decoder_utilization")
}
func (d *fakeDevice) GetJpgUtilization() (uint32, uint32, nvml.Return) {
	return 12, 1000, d.ret("jpeg_utilization")
}
func (d *fakeDevice) GetOfaUtilization() (uint32, uint32, nvml.Return) {
	return 13, 1000, d.ret("ofa_utilization")
}
func (d *fakeDevice) GetTemperature(nvml.TemperatureSensors) (uint32, nvml.Return) {
	return 55, d.ret("temperature_gpu")
}
func (d *fakeDevice) GetTemperatureThreshold(nvml.TemperatureThresholds) (uint32, nvml.Return) {
	return 90, d.ret("temperature_threshold")
}
func (d *fakeDevice) GetFanSpeed() (uint32, nvml.Return) { return 40, d.ret("fan_speed") }
func (d *fakeDevice) GetNumFans() (int, nvml.Return)     { return 2, d.ret("number_of_fans") }
func (d *fakeDevice) GetPerformanceState() (nvml.Pstates, nvml.Return) {
	return nvml.PSTATE_2, d.ret("performance_state")
}
func (d *fakeDevice) GetPowerUsage() (uint32, nvml.Return) {
	return 100000, d.ret("power_usage")
}
func (d *fakeDevice) GetPowerManagementLimit() (uint32, nvml.Return) {
	return 250000, d.ret("power_limit")
}
func (d *fakeDevice) GetPowerManagementDefaultLimit() (uint32, nvml.Return) {
	return 250000, d.ret("power_default_limit")
}
func (d *fakeDevice) GetPowerManagementLimitConstraints() (uint32, uint32, nvml.Return) {
	return 100000, 300000, d.ret("power_limit_constraints")
}
func (d *fakeDevice) GetEnforcedPowerLimit() (uint32, nvml.Return) {
	return 240000, d.ret("power_enforced_limit")
}
func (d *fakeDevice) GetTotalEnergyConsumption() (uint64, nvml.Return) {
	return 1234, d.ret("total_energy")
}
func (d *fakeDevice) GetPowerState() (nvml.Pstates, nvml.Return) {
	return nvml.PSTATE_2, d.ret("power_state")
}
func (d *fakeDevice) GetClockInfo(nvml.ClockType) (uint32, nvml.Return) {
	return 1200, d.ret("clock")
}
func (d *fakeDevice) GetMaxClockInfo(nvml.ClockType) (uint32, nvml.Return) {
	return 1800, d.ret("max_clock")
}
func (d *fakeDevice) GetCurrentClocksThrottleReasons() (uint64, nvml.Return) {
	return 0, d.ret("clock_throttle_reasons")
}
func (d *fakeDevice) GetPciInfo() (nvml.PciInfo, nvml.Return) {
	var pci nvml.PciInfo
	copy(pci.BusId[:], "0000:01:00.0")
	pci.Domain = 0
	pci.Bus = 1
	pci.Device = 0
	pci.PciDeviceId = 1
	pci.PciSubSystemId = 2
	return pci, d.ret("pci_info")
}
func (d *fakeDevice) GetCurrPcieLinkGeneration() (int, nvml.Return) {
	return 4, d.ret("pcie_current_link_generation")
}
func (d *fakeDevice) GetCurrPcieLinkWidth() (int, nvml.Return) {
	return 16, d.ret("pcie_current_link_width")
}
func (d *fakeDevice) GetMaxPcieLinkGeneration() (int, nvml.Return) {
	return 5, d.ret("pcie_max_link_generation")
}
func (d *fakeDevice) GetMaxPcieLinkWidth() (int, nvml.Return) {
	return 16, d.ret("pcie_max_link_width")
}
func (d *fakeDevice) GetPcieLinkMaxSpeed() (uint32, nvml.Return) {
	return 32000, d.ret("pcie_max_link_speed")
}
func (d *fakeDevice) GetPcieReplayCounter() (int, nvml.Return) {
	return 0, d.ret("pcie_replay_counter")
}
func (d *fakeDevice) GetEccMode() (nvml.EnableState, nvml.EnableState, nvml.Return) {
	return nvml.FEATURE_ENABLED, nvml.FEATURE_ENABLED, d.ret("ecc_mode")
}
func (d *fakeDevice) GetTotalEccErrors(nvml.MemoryErrorType, nvml.EccCounterType) (uint64, nvml.Return) {
	return 0, d.ret("ecc_counter")
}
func (d *fakeDevice) GetRetiredPages(nvml.PageRetirementCause) ([]uint64, nvml.Return) {
	return []uint64{1, 2}, d.ret("retired_pages")
}
func (d *fakeDevice) GetRetiredPagesPendingStatus() (nvml.EnableState, nvml.Return) {
	return nvml.FEATURE_DISABLED, d.ret("retired_pages_pending_status")
}
func (d *fakeDevice) GetComputeRunningProcesses() ([]nvml.ProcessInfo, nvml.Return) {
	return []nvml.ProcessInfo{{Pid: 100, UsedGpuMemory: 10}}, d.ret("processes_compute")
}
func (d *fakeDevice) GetGraphicsRunningProcesses() ([]nvml.ProcessInfo, nvml.Return) {
	return []nvml.ProcessInfo{{Pid: 101, UsedGpuMemory: 11}}, d.ret("processes_graphics")
}
func (d *fakeDevice) GetMPSComputeRunningProcesses() ([]nvml.ProcessInfo, nvml.Return) {
	return []nvml.ProcessInfo{{Pid: 102, UsedGpuMemory: 12}}, d.ret("processes_mps_compute")
}
