package metrics

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var pseudoFilesystems = map[string]bool{
	"proc":      true,
	"sysfs":     true,
	"devtmpfs":  true,
	"devpts":    true,
	"tmpfs":     true,
	"cgroup":    true,
	"cgroup2":   true,
	"overlay":   true,
	"securityfs": true,
	"debugfs":   true,
	"tracefs":   true,
	"hugetlbfs": true,
	"mqueue":    true,
	"configfs":  true,
	"fusectl":   true,
	"pstore":    true,
	"bpf":       true,
	"efivarfs":  true,
	"autofs":    true,
	"rpc_pipefs": true,
}

type DiskDetail struct {
	MountPoint   string  `json:"mount_point"`
	Filesystem   string  `json:"filesystem"`
	TotalGB      float64 `json:"total_gb"`
	UsedGB       float64 `json:"used_gb"`
	UsagePercent float64 `json:"usage_percent"`
}

type Snapshot struct {
	CPUCores            int
	CPUUsagePercent     float64
	MemoryTotalGB       float64
	MemoryUsedGB        float64
	MemoryUsagePercent  float64
	SwapTotalGB         float64
	SwapUsedGB          float64
	SwapUsagePercent    float64
	DiskUsagePercent    float64
	DiskDetails         []DiskDetail
}

type SystemCollector struct {
	diskPath string
}

func NewSystemCollector(diskPath string) *SystemCollector {
	if strings.TrimSpace(diskPath) == "" {
		diskPath = "/"
	}
	return &SystemCollector{diskPath: diskPath}
}

func (c *SystemCollector) Collect() (Snapshot, error) {
	cpuCores := runtime.NumCPU()
	cpuUsage, err := collectCPUUsage()
	if err != nil {
		return Snapshot{}, err
	}
	memTotal, memUsed, memUsage, err := collectMemoryStats()
	if err != nil {
		return Snapshot{}, err
	}
	swapTotal, swapUsed, swapUsage, err := collectSwapStats()
	if err != nil {
		return Snapshot{}, err
	}
	diskDetails, diskUsage, err := collectAllDisks()
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		CPUCores:           cpuCores,
		CPUUsagePercent:    roundToOneDecimal(cpuUsage),
		MemoryTotalGB:      roundToTwoDecimals(memTotal),
		MemoryUsedGB:       roundToTwoDecimals(memUsed),
		MemoryUsagePercent: roundToOneDecimal(memUsage),
		SwapTotalGB:        roundToTwoDecimals(swapTotal),
		SwapUsedGB:         roundToTwoDecimals(swapUsed),
		SwapUsagePercent:   roundToOneDecimal(swapUsage),
		DiskUsagePercent:   roundToOneDecimal(diskUsage),
		DiskDetails:        diskDetails,
	}, nil
}

func collectCPUUsage() (float64, error) {
	idle1, total1, err := readCPUStat()
	if err != nil {
		return 0, err
	}
	time.Sleep(200 * time.Millisecond)
	idle2, total2, err := readCPUStat()
	if err != nil {
		return 0, err
	}

	totalDelta := float64(total2 - total1)
	idleDelta := float64(idle2 - idle1)
	if totalDelta <= 0 {
		return 0, nil
	}
	used := (1 - idleDelta/totalDelta) * 100
	if used < 0 {
		return 0, nil
	}
	if used > 100 {
		return 100, nil
	}
	return used, nil
}

func readCPUStat() (uint64, uint64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, fmt.Errorf("open /proc/stat: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, 0, fmt.Errorf("invalid cpu stat format")
		}

		var values []uint64
		for _, field := range fields[1:] {
			value, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("parse cpu stat: %w", err)
			}
			values = append(values, value)
		}

		var total uint64
		for _, value := range values {
			total += value
		}

		idle := values[3]
		if len(values) > 4 {
			idle += values[4]
		}
		return idle, total, nil
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("scan /proc/stat: %w", err)
	}
	return 0, 0, fmt.Errorf("cpu stat line not found")
}

func collectMemoryStats() (totalGB, usedGB, usagePercent float64, err error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("open /proc/meminfo: %w", err)
	}
	defer file.Close()

	var totalKB, availableKB float64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			totalKB, err = parseMeminfoValue(line)
			if err != nil {
				return 0, 0, 0, err
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			availableKB, err = parseMeminfoValue(line)
			if err != nil {
				return 0, 0, 0, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, 0, fmt.Errorf("scan /proc/meminfo: %w", err)
	}
	if totalKB <= 0 {
		return 0, 0, 0, fmt.Errorf("MemTotal not found")
	}
	if availableKB < 0 {
		availableKB = 0
	}

	usedKB := totalKB - availableKB
	if usedKB < 0 {
		usedKB = 0
	}
	totalGB = totalKB / (1024 * 1024)
	usedGB = usedKB / (1024 * 1024)
	if totalKB > 0 {
		usagePercent = (usedKB / totalKB) * 100
	}
	if usagePercent > 100 {
		usagePercent = 100
	}
	return totalGB, usedGB, usagePercent, nil
}

func collectSwapStats() (totalGB, usedGB, usagePercent float64, err error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("open /proc/meminfo: %w", err)
	}
	defer file.Close()

	var swapTotalKB, swapFreeKB float64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "SwapTotal:") {
			swapTotalKB, err = parseMeminfoValue(line)
			if err != nil {
				return 0, 0, 0, err
			}
		}
		if strings.HasPrefix(line, "SwapFree:") {
			swapFreeKB, err = parseMeminfoValue(line)
			if err != nil {
				return 0, 0, 0, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, 0, fmt.Errorf("scan /proc/meminfo: %w", err)
	}

	if swapTotalKB <= 0 {
		return 0, 0, 0, nil
	}
	swapUsedKB := swapTotalKB - swapFreeKB
	if swapUsedKB < 0 {
		swapUsedKB = 0
	}
	totalGB = swapTotalKB / (1024 * 1024)
	usedGB = swapUsedKB / (1024 * 1024)
	usagePercent = (swapUsedKB / swapTotalKB) * 100
	if usagePercent > 100 {
		usagePercent = 100
	}
	return totalGB, usedGB, usagePercent, nil
}

func parseMeminfoValue(line string) (float64, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, fmt.Errorf("invalid meminfo format")
	}
	value, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, fmt.Errorf("parse meminfo value: %w", err)
	}
	return value, nil
}

func collectAllDisks() ([]DiskDetail, float64, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, 0, fmt.Errorf("open /proc/mounts: %w", err)
	}
	defer file.Close()

	var details []DiskDetail
	var totalUsed, totalCapacity float64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		device := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]

		if pseudoFilesystems[fsType] {
			continue
		}
		if strings.HasPrefix(device, "/dev/loop") {
			continue
		}
		if !filepath.IsAbs(mountPoint) {
			continue
		}

		var fs syscall.Statfs_t
		if err := syscall.Statfs(mountPoint, &fs); err != nil {
			continue
		}

		total := float64(fs.Blocks) * float64(fs.Bsize)
		free := float64(fs.Bavail) * float64(fs.Bsize)
		if total <= 0 {
			continue
		}
		used := total - free
		if used < 0 {
			used = 0
		}
		usagePct := (used / total) * 100
		if usagePct > 100 {
			usagePct = 100
		}

		totalGB := total / (1024 * 1024 * 1024)
		usedGB := used / (1024 * 1024 * 1024)
		totalCapacity += total
		totalUsed += used

		details = append(details, DiskDetail{
			MountPoint:   mountPoint,
			Filesystem:   fsType,
			TotalGB:      roundToTwoDecimals(totalGB),
			UsedGB:       roundToTwoDecimals(usedGB),
			UsagePercent: roundToOneDecimal(usagePct),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("scan /proc/mounts: %w", err)
	}

	var diskUsage float64
	if totalCapacity > 0 {
		diskUsage = (totalUsed / totalCapacity) * 100
		if diskUsage > 100 {
			diskUsage = 100
		}
	}
	return details, diskUsage, nil
}

func roundToOneDecimal(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func roundToTwoDecimals(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
