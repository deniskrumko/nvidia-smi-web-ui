package gpuinfo

// Snapshot contains a complete point-in-time GPU inventory.
type Snapshot struct {
	System   SystemInfo `json:"system"`
	Devices  []Device   `json:"devices"`
	Warnings []Warning  `json:"warnings,omitempty"`
}

// ProcessSnapshot contains GPU processes and warnings from process collection.
type ProcessSnapshot struct {
	Processes []Process `json:"processes"`
	Warnings  []Warning `json:"warnings,omitempty"`
}

// SystemInfo describes the NVIDIA software stack available through NVML.
type SystemInfo struct {
	DriverVersion     *string `json:"driver_version,omitempty"`
	NVMLVersion       *string `json:"nvml_version,omitempty"`
	CUDADriverVersion *string `json:"cuda_driver_version,omitempty"`
}

// Device contains all GPU fields collected by the first snapshot implementation.
type Device struct {
	Index             *int              `json:"index,omitempty"`
	UUID              *string           `json:"uuid,omitempty"`
	Name              *string           `json:"name,omitempty"`
	Brand             *string           `json:"brand,omitempty"`
	Serial            *string           `json:"serial,omitempty"`
	BoardID           *uint32           `json:"board_id,omitempty"`
	Architecture      *string           `json:"architecture,omitempty"`
	ComputeCapability *string           `json:"compute_capability,omitempty"`
	Memory            *MemoryInfo       `json:"memory,omitempty"`
	BAR1Memory        *MemoryInfo       `json:"bar1_memory,omitempty"`
	Utilization       *UtilizationInfo  `json:"utilization,omitempty"`
	Temperature       *TemperatureInfo  `json:"temperature,omitempty"`
	Power             *PowerInfo        `json:"power,omitempty"`
	Clocks            *ClockInfo        `json:"clocks,omitempty"`
	PCI               *PCIInfo          `json:"pci,omitempty"`
	ECC               *ECCInfo          `json:"ecc,omitempty"`
	RetiredPages      *RetiredPagesInfo `json:"retired_pages,omitempty"`
	Processes         []Process         `json:"processes,omitempty"`
	Warnings          []Warning         `json:"warnings,omitempty"`
}

// MemoryInfo stores byte-level memory counters.
type MemoryInfo struct {
	TotalBytes uint64 `json:"total_bytes"`
	FreeBytes  uint64 `json:"free_bytes"`
	UsedBytes  uint64 `json:"used_bytes"`
}

// UtilizationInfo stores percentage utilization metrics.
type UtilizationInfo struct {
	GPUPercent     *uint32 `json:"gpu_percent,omitempty"`
	MemoryPercent  *uint32 `json:"memory_percent,omitempty"`
	EncoderPercent *uint32 `json:"encoder_percent,omitempty"`
	DecoderPercent *uint32 `json:"decoder_percent,omitempty"`
	JPEGPercent    *uint32 `json:"jpeg_percent,omitempty"`
	OFAPercent     *uint32 `json:"ofa_percent,omitempty"`
}

// TemperatureInfo stores thermal readings in Celsius.
type TemperatureInfo struct {
	GPUCelsius       *uint32 `json:"gpu_celsius,omitempty"`
	ShutdownCelsius  *uint32 `json:"shutdown_celsius,omitempty"`
	SlowdownCelsius  *uint32 `json:"slowdown_celsius,omitempty"`
	MemMaxCelsius    *uint32 `json:"memory_max_celsius,omitempty"`
	GPUMaxCelsius    *uint32 `json:"gpu_max_celsius,omitempty"`
	FanSpeedPercent  *uint32 `json:"fan_speed_percent,omitempty"`
	NumberOfFans     *int    `json:"number_of_fans,omitempty"`
	PerformanceState *string `json:"performance_state,omitempty"`
}

// PowerInfo stores power values in milliwatts unless otherwise noted.
type PowerInfo struct {
	UsageMilliwatts         *uint32 `json:"usage_milliwatts,omitempty"`
	LimitMilliwatts         *uint32 `json:"limit_milliwatts,omitempty"`
	DefaultLimitMilliwatts  *uint32 `json:"default_limit_milliwatts,omitempty"`
	EnforcedLimitMilliwatts *uint32 `json:"enforced_limit_milliwatts,omitempty"`
	MinLimitMilliwatts      *uint32 `json:"min_limit_milliwatts,omitempty"`
	MaxLimitMilliwatts      *uint32 `json:"max_limit_milliwatts,omitempty"`
	TotalEnergyMillijoules  *uint64 `json:"total_energy_millijoules,omitempty"`
	PowerState              *string `json:"power_state,omitempty"`
}

// ClockInfo stores current and maximum clocks in MHz.
type ClockInfo struct {
	GraphicsMHz         *uint32 `json:"graphics_mhz,omitempty"`
	SMMHz               *uint32 `json:"sm_mhz,omitempty"`
	MemoryMHz           *uint32 `json:"memory_mhz,omitempty"`
	VideoMHz            *uint32 `json:"video_mhz,omitempty"`
	MaxGraphicsMHz      *uint32 `json:"max_graphics_mhz,omitempty"`
	MaxSMMHz            *uint32 `json:"max_sm_mhz,omitempty"`
	MaxMemoryMHz        *uint32 `json:"max_memory_mhz,omitempty"`
	MaxVideoMHz         *uint32 `json:"max_video_mhz,omitempty"`
	ThrottleReasonsMask *uint64 `json:"throttle_reasons_mask,omitempty"`
}

// PCIInfo stores PCI bus and link information.
type PCIInfo struct {
	BusID                 *string `json:"bus_id,omitempty"`
	Domain                *uint32 `json:"domain,omitempty"`
	Bus                   *uint32 `json:"bus,omitempty"`
	Device                *uint32 `json:"device,omitempty"`
	PCIDeviceID           *uint32 `json:"pci_device_id,omitempty"`
	PCISubsystemID        *uint32 `json:"pci_subsystem_id,omitempty"`
	CurrentLinkGeneration *int    `json:"current_link_generation,omitempty"`
	CurrentLinkWidth      *int    `json:"current_link_width,omitempty"`
	MaxLinkGeneration     *int    `json:"max_link_generation,omitempty"`
	MaxLinkWidth          *int    `json:"max_link_width,omitempty"`
	MaxLinkSpeedMbps      *uint32 `json:"max_link_speed_mbps,omitempty"`
	ReplayCounter         *int    `json:"replay_counter,omitempty"`
}

// ECCInfo stores high-level ECC counters when a device supports them.
type ECCInfo struct {
	CorrectedVolatile    *uint64 `json:"corrected_volatile,omitempty"`
	UncorrectedVolatile  *uint64 `json:"uncorrected_volatile,omitempty"`
	CorrectedAggregate   *uint64 `json:"corrected_aggregate,omitempty"`
	UncorrectedAggregate *uint64 `json:"uncorrected_aggregate,omitempty"`
	CurrentMode          *string `json:"current_mode,omitempty"`
	PendingMode          *string `json:"pending_mode,omitempty"`
}

// RetiredPagesInfo stores retired page counts by cause.
type RetiredPagesInfo struct {
	SingleBitECCCount *int    `json:"single_bit_ecc_count,omitempty"`
	DoubleBitECCCount *int    `json:"double_bit_ecc_count,omitempty"`
	PendingStatus     *string `json:"pending_status,omitempty"`
}

// Process describes a GPU process reported by NVML.
type Process struct {
	DeviceIndex       *int    `json:"device_index,omitempty"`
	DeviceUUID        *string `json:"device_uuid,omitempty"`
	Type              string  `json:"type"`
	PID               uint32  `json:"pid"`
	UsedGPUBytes      *uint64 `json:"used_gpu_bytes,omitempty"`
	GPUInstanceID     *uint32 `json:"gpu_instance_id,omitempty"`
	ComputeInstanceID *uint32 `json:"compute_instance_id,omitempty"`
}

// Warning describes a metric that could not be collected.
type Warning struct {
	Scope   string `json:"scope"`
	Field   string `json:"field"`
	Message string `json:"message"`
}
