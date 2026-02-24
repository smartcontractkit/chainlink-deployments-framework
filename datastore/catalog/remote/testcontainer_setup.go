package remote

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresImage         = "postgres:16-alpine"
	postgresContainerName = "postgres-container"
	postgresUser          = "postgres"
	postgresPassword      = "postgres"
	postgresDB            = "op_catalog_db"
	postgresPort          = "5432"
	catalogImageEnv       = "CATALOG_SERVICE_IMAGE"
	catalogImageRepo      = "op-catalog-service"
	catalogImageTag       = "latest"
	catalogPort           = "8080"
	networkName           = "chainlink_catalog_network"
)

// TestContainerSetup holds all testcontainer resources for catalog service testing
type TestContainerSetup struct {
	PostgresContainer *postgres.PostgresContainer
	CatalogContainer  testcontainers.Container
	PostgresDSN       string
	CatalogGRPCAddr   string
	Network           *testcontainers.DockerNetwork
}

// SetupCatalogTestContainers initializes all required containers for testing:
// 1. PostgreSQL database
// 2. Catalog service (with db migration)
func SetupCatalogTestContainers(ctx context.Context) (*TestContainerSetup, error) {
	setup := &TestContainerSetup{}

	// Create a network for containers to communicate
	net, err := network.New(ctx, network.WithDriver("bridge"))
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}
	setup.Network = net

	// Start PostgreSQL
	log.Println("Starting PostgreSQL container...")
	if err := setup.startPostgres(ctx); err != nil {
		return nil, fmt.Errorf("failed to start postgres: %w", err)
	}
	log.Printf("PostgreSQL started at: %s", setup.PostgresDSN)

	// Start Catalog Service (which includes migration step)
	log.Println("Starting Catalog service container...")
	if err := setup.startCatalogService(ctx); err != nil {
		return nil, fmt.Errorf("failed to start catalog service: %w", err)
	}
	log.Printf("Catalog service started at: %s", setup.CatalogGRPCAddr)

	return setup, nil
}

// startPostgres starts a PostgreSQL container
func (s *TestContainerSetup) startPostgres(ctx context.Context) error {
	postgresContainer, err := postgres.Run(ctx,
		postgresImage,
		postgres.WithDatabase(postgresDB),
		postgres.WithUsername(postgresUser),
		postgres.WithPassword(postgresPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
		withNetwork(s.Network.Name, postgresContainerName),
	)
	if err != nil {
		return fmt.Errorf("failed to start postgres container: %w", err)
	}

	s.PostgresContainer = postgresContainer

	// Get the mapped port for local access
	localPort, err := postgresContainer.MappedPort(ctx, postgresPort)
	if err != nil {
		return fmt.Errorf("failed to get postgres mapped port: %w", err)
	}

	// Build DSN for local access
	s.PostgresDSN = fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable",
		postgresUser, postgresPassword, localPort.Int(), postgresDB)

	return nil
}

// withNetwork is a helper function to attach a container to a network with a specific name
func withNetwork(networkName, containerName string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		req.Networks = append(req.Networks, networkName)
		if containerName != "" {
			if req.NetworkAliases == nil {
				req.NetworkAliases = make(map[string][]string)
			}
			req.NetworkAliases[networkName] = []string{containerName}
		}

		return nil
	}
}

// startCatalogService starts the catalog service container from ECR image
func (s *TestContainerSetup) startCatalogService(ctx context.Context) error {
	// Get the catalog image from environment or use default
	catalogImage := os.Getenv(catalogImageEnv)
	if catalogImage == "" {
		// Default to local image or ECR format
		catalogImage = fmt.Sprintf("%s:%s", catalogImageRepo, catalogImageTag)
		log.Printf("CATALOG_SERVICE_IMAGE not set, using default: %s", catalogImage)
	}

	log.Printf("Running database migration using catalog image...")

	// Run migration in a one-off container
	// The migration container must be on the same network as PostgreSQL
	// so it can connect using the container name instead of localhost
	// NOTE: Must override entrypoint since Dockerfile sets it to "service start"

	// Build DSN using postgres container name for network communication
	// Format: postgres://user:pass@postgres-container:5432/dbname?sslmode=disable
	migrationDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUser,
		postgresPassword,
		postgresContainerName,
		postgresPort,
		postgresDB,
	)

	migrationReq := testcontainers.ContainerRequest{
		Image:      catalogImage,
		Entrypoint: []string{"/app/op-catalog"}, // Override the Dockerfile ENTRYPOINT
		Cmd: []string{
			"migrate",
			"up",
			"--postgres-dsn",
			migrationDSN, // Use postgres container name for network communication
		},
		Networks: []string{s.Network.Name}, // IMPORTANT: Must be on same network as postgres
		WaitingFor: wait.ForExit().
			WithExitTimeout(2 * time.Minute),
	}

	migrationContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: migrationReq,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start migration container: %w", err)
	}

	// Wait for migration to complete and check exit code
	state, err := migrationContainer.State(ctx)
	if err != nil {
		return fmt.Errorf("failed to get migration container state: %w", err)
	}

	if state.ExitCode != 0 {
		// Get logs for debugging
		logReader, logErr := migrationContainer.Logs(ctx)
		if logErr == nil && logReader != nil {
			logBytes := make([]byte, 8192)
			n, _ := logReader.Read(logBytes)
			sanitized := strings.ReplaceAll(string(logBytes[:n]), "\n", "\\n")
			sanitized = strings.ReplaceAll(sanitized, "\r", "\\r")
			log.Printf("Migration logs: %s", sanitized)
		}

		return fmt.Errorf("migration failed with exit code: %d", state.ExitCode)
	}

	// Terminate migration container
	if terminateErr := migrationContainer.Terminate(ctx); terminateErr != nil {
		log.Printf("Warning: failed to terminate migration container: %v", terminateErr)
	}

	log.Println("Database migration completed successfully")

	// Get postgres container IP for service-to-db communication
	postgresIP, err := s.PostgresContainer.ContainerIP(ctx)
	if err != nil {
		return fmt.Errorf("failed to get postgres IP: %w", err)
	}

	// Build DSN for container-to-container communication
	containerPostgresDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		postgresUser, postgresPassword, postgresIP, postgresPort, postgresDB)

	// Now start the catalog service
	// NOTE: Must override entrypoint for the service to use env vars
	serviceReq := testcontainers.ContainerRequest{
		Image:      catalogImage,
		Entrypoint: []string{"/app/op-catalog"}, // Override the Dockerfile ENTRYPOINT
		Cmd: []string{
			"service",
			"start",
		},
		Env: map[string]string{
			"CATALOG_POSTGRES_DSN":   containerPostgresDSN,
			"CATALOG_LISTEN_ADDRESS": "0.0.0.0",
			"CATALOG_LISTEN_PORT":    catalogPort,
		},
		ExposedPorts: []string{catalogPort + "/tcp"},
		Networks:     []string{s.Network.Name},
		WaitingFor: wait.ForLog("gRPC server listening").
			WithStartupTimeout(30 * time.Second).
			WithPollInterval(500 * time.Millisecond),
	}

	catalogContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: serviceReq,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start catalog service: %w", err)
	}

	s.CatalogContainer = catalogContainer

	// Get the mapped port for local gRPC access
	mappedPort, err := catalogContainer.MappedPort(ctx, catalogPort)
	if err != nil {
		return fmt.Errorf("failed to get catalog service mapped port: %w", err)
	}

	s.CatalogGRPCAddr = fmt.Sprintf("localhost:%d", mappedPort.Int())

	return nil
}

// Teardown cleans up all containers and networks
func (s *TestContainerSetup) Teardown(ctx context.Context) error {
	log.Println("Tearing down test containers...")

	var errs []string

	// Terminate catalog service
	if s.CatalogContainer != nil {
		if err := s.CatalogContainer.Terminate(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("failed to terminate catalog container: %v", err))
		}
	}

	// Terminate postgres
	if s.PostgresContainer != nil {
		if err := s.PostgresContainer.Terminate(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("failed to terminate postgres container: %v", err))
		}
	}

	// Remove network
	if s.Network != nil {
		if err := s.Network.Remove(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("failed to remove network: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("teardown errors: %s", strings.Join(errs, "; "))
	}

	log.Println("Teardown complete")

	return nil
}

// GetCatalogGRPCAddress returns the gRPC address for catalog service
func (s *TestContainerSetup) GetCatalogGRPCAddress() string {
	return s.CatalogGRPCAddr
}
