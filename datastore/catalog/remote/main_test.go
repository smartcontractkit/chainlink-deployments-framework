package remote

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

var (
	// globalTestSetup is the shared testcontainer setup for all remote tests
	globalTestSetup *TestContainerSetup
	// catalogGRPCAddress is the address of the catalog service for tests
	catalogGRPCAddress string
)

// TestMain is the entry point for all tests in this package
// It sets up testcontainers once for all tests and tears them down at the end
func TestMain(m *testing.M) {
	// Check if we should use an existing catalog service instead of testcontainers
	existingAddr := os.Getenv("CATALOG_GRPC_ADDRESS")
	if existingAddr != "" {
		log.Printf("Using existing catalog service at: %s", existingAddr)
		catalogGRPCAddress = existingAddr
		// Run tests and exit
		os.Exit(m.Run())
	}

	// Setup context with timeout for initialization
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("========================================")
	log.Println("Setting up testcontainers for catalog remote tests...")
	log.Println("========================================")

	// Setup testcontainers
	setup, err := SetupCatalogTestContainers(ctx)
	if err != nil {
		log.Fatalf("Failed to setup testcontainers: %v", err)
	}
	globalTestSetup = setup
	catalogGRPCAddress = setup.GetCatalogGRPCAddress()

	log.Println("========================================")
	log.Println("Testcontainers setup complete!")
	log.Printf("PostgreSQL DSN: %s", setup.PostgresDSN)
	log.Printf("Catalog gRPC Address: %s", catalogGRPCAddress)
	log.Println("========================================")

	// Set the environment variable so tests can find the service
	os.Setenv("CATALOG_GRPC_ADDRESS", catalogGRPCAddress)

	// Run all tests
	exitCode := m.Run()

	// Cleanup
	log.Println("========================================")
	log.Println("Cleaning up testcontainers...")
	log.Println("========================================")

	// Use a fresh context for cleanup to avoid timeout issues
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cleanupCancel()

	if err := globalTestSetup.Teardown(cleanupCtx); err != nil {
		log.Printf("Warning: Failed to teardown testcontainers: %v", err)
	}

	log.Println("Cleanup complete!")
	os.Exit(exitCode)
}
