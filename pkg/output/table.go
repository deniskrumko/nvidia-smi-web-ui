package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/gpuinfo"
)

// WriteDeviceTable renders an nvidia-smi-like GPU overview table.
func WriteDeviceTable(w io.Writer, snapshot gpuinfo.Snapshot) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	var err error
	writeLine(tw, &err, "ID\tNAME\tUUID\tMEMORY\tGPU UTIL\tMEM UTIL\tTEMP\tPOWER\tFAN")
	for _, device := range snapshot.Devices {
		writef(
			tw,
			&err,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			intValue(device.Index),
			stringValue(device.Name),
			stringValue(device.UUID),
			memoryValue(device.Memory),
			percentValue(device.Utilization, func(u *gpuinfo.UtilizationInfo) *uint32 { return u.GPUPercent }),
			percentValue(device.Utilization, func(u *gpuinfo.UtilizationInfo) *uint32 { return u.MemoryPercent }),
			tempValue(device.Temperature),
			powerValue(device.Power),
			fanValue(device.Temperature),
		)
	}
	if err != nil {
		return err
	}
	return tw.Flush()
}

// WriteDeviceDetails renders one device with grouped details.
func WriteDeviceDetails(w io.Writer, device gpuinfo.Device) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	var err error
	writef(tw, &err, "Index:\t%s\n", intValue(device.Index))
	writef(tw, &err, "Name:\t%s\n", stringValue(device.Name))
	writef(tw, &err, "UUID:\t%s\n", stringValue(device.UUID))
	writef(tw, &err, "Brand:\t%s\n", stringValue(device.Brand))
	writef(tw, &err, "Architecture:\t%s\n", stringValue(device.Architecture))
	writef(tw, &err, "Compute Capability:\t%s\n", stringValue(device.ComputeCapability))
	writef(tw, &err, "Memory:\t%s\n", memoryValue(device.Memory))
	writef(tw, &err, "BAR1 Memory:\t%s\n", memoryValue(device.BAR1Memory))
	writef(tw, &err, "GPU Utilization:\t%s\n", percentValue(device.Utilization, func(u *gpuinfo.UtilizationInfo) *uint32 { return u.GPUPercent }))
	writef(tw, &err, "Memory Utilization:\t%s\n", percentValue(device.Utilization, func(u *gpuinfo.UtilizationInfo) *uint32 { return u.MemoryPercent }))
	writef(tw, &err, "Temperature:\t%s\n", tempValue(device.Temperature))
	writef(tw, &err, "Power:\t%s\n", powerValue(device.Power))
	if device.PCI != nil {
		writef(tw, &err, "PCI Bus ID:\t%s\n", stringValue(device.PCI.BusID))
		writef(tw, &err, "PCI Link:\tgen %s x%s\n", intValue(device.PCI.CurrentLinkGeneration), intValue(device.PCI.CurrentLinkWidth))
	}
	writef(tw, &err, "Warnings:\t%d\n", len(device.Warnings))
	if len(device.Processes) > 0 {
		writeLine(tw, &err, "")
		writeLine(tw, &err, "PROCESS TYPE\tPID\tGPU MEMORY")
		for _, process := range device.Processes {
			writef(tw, &err, "%s\t%d\t%s\n", process.Type, process.PID, bytesValue(process.UsedGPUBytes))
		}
	}
	if err != nil {
		return err
	}
	return tw.Flush()
}

// WriteProcessTable renders GPU process information.
func WriteProcessTable(w io.Writer, processes []gpuinfo.Process) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	var err error
	writeLine(tw, &err, "GPU\tUUID\tTYPE\tPID\tGPU MEMORY\tGPU INSTANCE\tCOMPUTE INSTANCE")
	for _, process := range processes {
		writef(
			tw,
			&err,
			"%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			intValue(process.DeviceIndex),
			stringValue(process.DeviceUUID),
			process.Type,
			process.PID,
			bytesValue(process.UsedGPUBytes),
			uint32Value(process.GPUInstanceID),
			uint32Value(process.ComputeInstanceID),
		)
	}
	if err != nil {
		return err
	}
	return tw.Flush()
}

// WriteWarnings renders collection warnings when requested by the user.
func WriteWarnings(w io.Writer, warnings []gpuinfo.Warning) error {
	if len(warnings) == 0 {
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	var err error
	writeLine(tw, &err, "\nWARNINGS")
	writeLine(tw, &err, "SCOPE\tFIELD\tMESSAGE")
	for _, warning := range warnings {
		writef(tw, &err, "%s\t%s\t%s\n", warning.Scope, warning.Field, warning.Message)
	}
	if err != nil {
		return err
	}
	return tw.Flush()
}

func writeLine(w io.Writer, err *error, value string) {
	if *err != nil {
		return
	}
	_, *err = fmt.Fprintln(w, value)
}

func writef(w io.Writer, err *error, format string, args ...any) {
	if *err != nil {
		return
	}
	_, *err = fmt.Fprintf(w, format, args...)
}

func stringValue(value *string) string {
	if value == nil || *value == "" {
		return "-"
	}
	return *value
}

func intValue(value *int) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *value)
}

func uint32Value(value *uint32) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *value)
}

func percentValue(info *gpuinfo.UtilizationInfo, get func(*gpuinfo.UtilizationInfo) *uint32) string {
	if info == nil {
		return "-"
	}
	value := get(info)
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%d%%", *value)
}

func tempValue(info *gpuinfo.TemperatureInfo) string {
	if info == nil || info.GPUCelsius == nil {
		return "-"
	}
	return fmt.Sprintf("%d C", *info.GPUCelsius)
}

func fanValue(info *gpuinfo.TemperatureInfo) string {
	if info == nil || info.FanSpeedPercent == nil {
		return "-"
	}
	return fmt.Sprintf("%d%%", *info.FanSpeedPercent)
}

func memoryValue(info *gpuinfo.MemoryInfo) string {
	if info == nil {
		return "-"
	}
	return fmt.Sprintf("%s / %s", formatBytes(info.UsedBytes), formatBytes(info.TotalBytes))
}

func powerValue(info *gpuinfo.PowerInfo) string {
	if info == nil || info.UsageMilliwatts == nil {
		return "-"
	}
	if info.LimitMilliwatts == nil {
		return fmt.Sprintf("%.1f W", float64(*info.UsageMilliwatts)/1000)
	}
	return fmt.Sprintf("%.1f W / %.1f W", float64(*info.UsageMilliwatts)/1000, float64(*info.LimitMilliwatts)/1000)
}

func bytesValue(value *uint64) string {
	if value == nil {
		return "-"
	}
	return formatBytes(*value)
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
