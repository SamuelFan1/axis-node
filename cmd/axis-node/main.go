package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/SamuelFan1/axis-node/internal/axisclient"
	"github.com/SamuelFan1/axis-node/internal/config"
	"github.com/SamuelFan1/axis-node/internal/ippublic"
	"github.com/SamuelFan1/axis-node/internal/metrics"
	"github.com/SamuelFan1/axis-node/internal/monitoring"
	monitorproviders "github.com/SamuelFan1/axis-node/internal/monitoring/providers"
	"github.com/SamuelFan1/axis-node/internal/nodeid"
)

func buildMonitoringCollector(cfg *config.Config) *monitoring.Collector {
	if cfg == nil || !cfg.MonitoringEnabled {
		return nil
	}

	providers := make([]monitoring.Provider, 0, 2)
	if cfg.MonitoringGoSidecarEnabled {
		providers = append(providers, monitorproviders.NewGoSidecarProvider(
			cfg.SidecarStatsURL,
			time.Duration(cfg.SidecarStatsTimeoutSec)*time.Second,
		))
	}
	if cfg.MonitoringCFTunnelEnabled {
		providers = append(providers, monitorproviders.NewCloudflaredProvider(
			cfg.CFTunnelServiceName,
			cfg.CFTunnelMonitorServiceName,
			cfg.CFTunnelHealthURL,
			time.Duration(cfg.CFTunnelTimeoutSec)*time.Second,
		))
	}

	if len(providers) == 0 {
		return nil
	}

	return monitoring.NewCollector(providers...)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "register":
		if err := runRegister(); err != nil {
			log.Fatalf("register: %v", err)
		}
	case "agent":
		if err := runAgent(); err != nil {
			log.Fatalf("agent: %v", err)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func runRegister() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	uuidValue, err := nodeid.LoadOrCreate(cfg.UUIDFile)
	if err != nil {
		return fmt.Errorf("load or create uuid: %w", err)
	}

	client := axisclient.New(cfg.ServerURL, cfg.SharedToken)
	resp, err := client.RegisterNode(axisclient.RegisterNodeRequest{
		UUID:              uuidValue,
		Hostname:          cfg.Hostname,
		ManagementAddress: cfg.ManagementAddress,
		Region:            cfg.Region,
		Zone:              cfg.Zone,
		Status:            cfg.Status,
	})
	if err != nil {
		return fmt.Errorf("register node to axis: %w", err)
	}

	if resp.Node.UUID != "" && resp.Node.UUID != uuidValue {
		if err := nodeid.Save(cfg.UUIDFile, resp.Node.UUID); err != nil {
			return fmt.Errorf("save returned uuid: %w", err)
		}
		uuidValue = resp.Node.UUID
	}

	fmt.Printf("registered node: uuid=%s hostname=%s management_address=%s status=%s region=%s\n",
		uuidValue,
		resp.Node.Hostname,
		resp.Node.ManagementAddress,
		resp.Node.Status,
		resp.Node.Region,
	)
	return nil
}

func runAgent() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	uuidValue, err := nodeid.LoadOrCreate(cfg.UUIDFile)
	if err != nil {
		return fmt.Errorf("load or create uuid: %w", err)
	}

	client := axisclient.New(cfg.ServerURL, cfg.SharedToken)
	registerResp, err := client.RegisterNode(axisclient.RegisterNodeRequest{
		UUID:              uuidValue,
		Hostname:          cfg.Hostname,
		ManagementAddress: cfg.ManagementAddress,
		Region:            cfg.Region,
		Zone:              cfg.Zone,
		Status:            cfg.Status,
	})
	if err != nil {
		return fmt.Errorf("register node to axis: %w", err)
	}
	if registerResp.Node.UUID != "" && registerResp.Node.UUID != uuidValue {
		if err := nodeid.Save(cfg.UUIDFile, registerResp.Node.UUID); err != nil {
			return fmt.Errorf("save returned uuid: %w", err)
		}
		uuidValue = registerResp.Node.UUID
	}

	collector := metrics.NewSystemCollector(cfg.DiskPath)
	monitorCollector := buildMonitoringCollector(cfg)
	publicIP := ippublic.Detect()
	reportOnce := func() error {
		snapshot, err := collector.Collect()
		if err != nil {
			return err
		}
		internalIP := extractInternalIP(cfg.ManagementAddress)
		diskDetails := make([]axisclient.DiskDetail, len(snapshot.DiskDetails))
		for i, d := range snapshot.DiskDetails {
			diskDetails[i] = axisclient.DiskDetail{
				MountPoint:   d.MountPoint,
				Filesystem:   d.Filesystem,
				TotalGB:      d.TotalGB,
				UsedGB:       d.UsedGB,
				UsagePercent: d.UsagePercent,
			}
		}
		var monitoringRaw json.RawMessage
		if monitorCollector != nil {
			monitoringSnapshot := monitorCollector.Collect(context.Background())
			monitoringRaw, err = json.Marshal(monitoringSnapshot)
			if err != nil {
				return fmt.Errorf("marshal monitoring snapshot: %w", err)
			}
		}
		_, err = client.ReportNode(axisclient.ReportNodeRequest{
			UUID:               uuidValue,
			Hostname:           cfg.Hostname,
			ManagementAddress:  cfg.ManagementAddress,
			InternalIP:         internalIP,
			PublicIP:           publicIP,
			Region:             cfg.Region,
			Zone:               cfg.Zone,
			Status:             cfg.Status,
			CPUCores:           snapshot.CPUCores,
			CPUUsagePercent:    snapshot.CPUUsagePercent,
			MemoryTotalGB:      snapshot.MemoryTotalGB,
			MemoryUsedGB:       snapshot.MemoryUsedGB,
			MemoryUsagePercent: snapshot.MemoryUsagePercent,
			SwapTotalGB:        snapshot.SwapTotalGB,
			SwapUsedGB:         snapshot.SwapUsedGB,
			SwapUsagePercent:   snapshot.SwapUsagePercent,
			DiskUsagePercent:   snapshot.DiskUsagePercent,
			DiskDetails:        diskDetails,
			MonitoringSnapshot: monitoringRaw,
		})
		if err != nil {
			return err
		}
		fmt.Printf("reported node: uuid=%s cpu=%d cores %.1f%% mem=%.1f%% disk=%.1f%%\n",
			uuidValue,
			snapshot.CPUCores,
			snapshot.CPUUsagePercent,
			snapshot.MemoryUsagePercent,
			snapshot.DiskUsagePercent,
		)
		return nil
	}

	if err := reportOnce(); err != nil {
		return fmt.Errorf("initial report: %w", err)
	}

	ticker := time.NewTicker(time.Duration(cfg.ReportIntervalSec) * time.Second)
	defer ticker.Stop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	for {
		select {
		case <-ticker.C:
			if err := reportOnce(); err != nil {
				log.Printf("report failed: %v", err)
			}
		case sig := <-stop:
			fmt.Printf("axis-node agent stopping: signal=%s\n", sig.String())
			return nil
		}
	}
}

func extractInternalIP(managementAddress string) string {
	host, _, err := net.SplitHostPort(managementAddress)
	if err == nil && strings.TrimSpace(host) != "" {
		return host
	}
	return managementAddress
}

func printUsage() {
	fmt.Println("axis-node")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  axis-node register")
	fmt.Println("  axis-node agent")
}
