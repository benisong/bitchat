package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/benisong/bitchat/farp/contribution"
	"github.com/benisong/bitchat/farp/ratelimit"
	"github.com/benisong/bitchat/internal/api"
	"github.com/benisong/bitchat/internal/db"
	"github.com/benisong/bitchat/internal/identitystore"
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
	rootCmd.Flags().StringVar(&dataDir, "data-dir", defaultDataDir(), "data directory")
	rootCmd.Flags().BoolVar(&isRelay, "relay", false, "enable relay service")
	rootCmd.Flags().BoolVar(&isDHTServer, "dht-server", false, "enable DHT server mode (needs public IP)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runDaemon(cmd *cobra.Command, args []string) error {
	if isRelay || isDHTServer {
		return fmt.Errorf("relay and DHT modes are not implemented in the v0.1 security core")
	}
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

	// 2. load or create the encrypted, device-bound identity
	node, err := identitystore.LoadOrCreate(sqlDB, dataDir)
	if err != nil {
		return fmt.Errorf("identity: %w", err)
	}
	fmt.Printf("✅ identity: %s...\n", node.ID[:16])

	// 3. init credits & rate limit
	credits := contribution.NewState(node.ID)
	quota := ratelimit.NewManager()

	// 4. start HTTP API
	apiServer := api.NewServer(node, quota, credits)
	addr, err := apiServer.Start()
	if err != nil {
		return fmt.Errorf("api start: %w", err)
	}
	fmt.Printf("✅ local API: http://%s\n", addr)

	// 5. wait for an explicit shutdown so SQLite and the API close cleanly
	fmt.Println("💤 daemon running... press Ctrl+C to stop")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return apiServer.Close(shutdownCtx)
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".bitchat")
	}
	return filepath.Join(home, ".bitchat")
}
