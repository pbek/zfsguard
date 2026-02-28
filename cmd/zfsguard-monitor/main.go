// zfsguard-monitor is the background monitoring service for ZFS and SMART health.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pbek/zfsguard/internal/config"
	"github.com/pbek/zfsguard/internal/monitor"
	"github.com/pbek/zfsguard/internal/version"
)

func main() {
	configPath := flag.String("config", "", "Path to config file (default: auto-detect)")
	oneshot := flag.Bool("oneshot", false, "Run a single check and exit")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String("zfsguard-monitor"))
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	svc := monitor.New(cfg)

	if *oneshot {
		if err := svc.RunOnce(); err != nil {
			log.Fatalf("Check failed: %v", err)
		}
		return
	}

	if err := svc.Run(); err != nil {
		log.Fatalf("Monitor failed: %v", err)
	}
}
