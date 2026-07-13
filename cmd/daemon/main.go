package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/benisong/bitchat/farp/identity"
	"github.com/benisong/bitchat/farp/ledger"
	"github.com/benisong/bitchat/farp/ratelimit"
	"github.com/benisong/bitchat/internal/api"
	"github.com/benisong/bitchat/internal/config"
	"github.com/benisong/bitchat/internal/db"
	"github.com/spf13/cobra"
)

var (
	dataDir     string
	isRelay     bool
	isDHTServer bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "bitchat daemon",
		Short: "Core daemon",
		RunE:  runDaemon,
	}
	rootCmd.Flags().StringVar(&dataDir, "data-dir", filepath.Join(os.Getenv("HOME"), ".bitchat"), "data directory")
	rootCmd.Flags().BoolVar(&isRelay, "relay", false, "enable relay service")
	rootCmd.Flags().BoolVar(&isDHTServer, "dht-server", false, "enable DHT server mode (needs public IP)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runDaemon(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 FARP Core Daemon v0.1 starting...")
	fmt.Printf("data-dir: %s\n", dataDir)

	// 1. open database
	sqlDB, err := db.Open(dataDir)
	if err != nil {
		return fmt.Errorf("db open: %w", err)
	}
	defer sqlDB.Close()
	if err := sqlDB.InitSchema(); err != nil {
		return fmt.Errorf("db init: %w", err)
	}
	fmt.Println("✅ database ready")

	// 2. load or create identity
	// MVP: generate fresh each run; persistence later
	node, err := identity.Generate()
	if err != nil {
		return fmt.Errorf("identity: %w", err)
	}
	fmt.Printf("✅ identity: %s...\n", node.ID[:16])

	// 3. init credits & rate limit
	credits := &ledger.CreditRecord{
		Pubkey:            node.ID,
		Balance:           0,
		Frozen:            false,
		ContributionRatio: 0,
	}
	quota := ratelimit.NewManager()
	if isRelay {
		quota.SetCapacity(config.TrustedQuotaPerWindow)
	}

	// 4. start HTTP API
	apiServer := api.NewServer(node, quota, credits)
	addr, err := apiServer.Start()
	if err != nil {
		return fmt.Errorf("api start: %w", err)
	}
	fmt.Printf("✅ local API: http://%s\n", addr)

	// 5. block forever
	fmt.Println("💤 daemon running... press Ctrl+C to stop")
	select {}
}
