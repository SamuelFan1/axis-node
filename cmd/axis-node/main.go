package main

import (
	"fmt"
	"log"
	"os"

	"github.com/SamuelFan1/axis-node/internal/axisclient"
	"github.com/SamuelFan1/axis-node/internal/config"
	"github.com/SamuelFan1/axis-node/internal/nodeid"
)

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

	client := axisclient.New(cfg.ServerURL)
	resp, err := client.RegisterNode(axisclient.RegisterNodeRequest{
		UUID:              uuidValue,
		Hostname:          cfg.Hostname,
		ManagementAddress: cfg.ManagementAddress,
		Region:            cfg.Region,
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

func printUsage() {
	fmt.Println("axis-node")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  axis-node register")
}
