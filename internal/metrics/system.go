package metrics

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Snapshot struct {
	CPUUsagePercent    float64
	MemoryUsagePercent float64
	DiskUsagePercent   float64
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
	cpuUsage, err := collectCPUUsage()
	if err != nil {
		return Snapshot{}, err
	}
	memUsage, err := collectMemoryUsage()
	if err != nil {
		return Snapshot{}, err
	}
	diskUsage, err := collectDiskUsage(c.diskPath)
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		CPUUsagePercent:    roundToOneDecimal(cpuUsage),
		MemoryUsagePercent: roundToOneDecimal(memUsage),
		DiskUsagePercent:   roundToOneDecimal(diskUsage),
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

func collectMemoryUsage() (float64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, fmt.Errorf("open /proc/meminfo: %w", err)
	}
	defer file.Close()

	var totalKB float64
	var availableKB float64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			totalKB, err = parseMeminfoValue(line)
			if err != nil {
				return 0, err
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			availableKB, err = parseMeminfoValue(line)
			if err != nil {
				return 0, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("scan /proc/meminfo: %w", err)
	}
	if totalKB <= 0 {
		return 0, fmt.Errorf("MemTotal not found")
	}
	if availableKB < 0 {
		availableKB = 0
	}

	used := (1 - availableKB/totalKB) * 100
	if used < 0 {
		return 0, nil
	}
	if used > 100 {
		return 100, nil
	}
	return used, nil
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

func collectDiskUsage(path string) (float64, error) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, fmt.Errorf("statfs %s: %w", path, err)
	}

	total := float64(fs.Blocks) * float64(fs.Bsize)
	free := float64(fs.Bavail) * float64(fs.Bsize)
	if total <= 0 {
		return 0, nil
	}

	used := (1 - free/total) * 100
	if used < 0 {
		return 0, nil
	}
	if used > 100 {
		return 100, nil
	}
	return used, nil
}

func roundToOneDecimal(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}
