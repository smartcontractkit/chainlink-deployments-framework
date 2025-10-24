# Remote implementation of Catalog datastore APIs

This implementation calls a gRPC service which is backed by the Catalog postgres
database. The service offers a streaming gRPC API, which allows for transaction
state to be maintained through that stream connection. If the stream closes,
normal cleanup will rollback the transaction. The [Catalog service APIs] include
message to begin, commit, and roll-back a transaction.

[Catalog service APIs]: http://github.com/smartcontractkit/op-catalog

## Running Tests

The tests in this package use **testcontainers** to automatically start the required services:
- PostgreSQL database
- Catalog service (from ECR or local image)
- Database migration

**Important:** Testcontainers uses images from your local Docker daemon. You must either:
1. Build the image locally: `docker build -t op-catalog-service:latest .`
2. Pull from ECR with a specific tag: `docker pull 123123123123.dkr.ecr.us-east-1.amazonaws.com/chainlink-catalog-service:TAG`
   - Use `aws ecr describe-images` to find available tags
   - Common patterns: version tags (v0.0.1)

### Quick Start (Local Build)

```bash
# Build the catalog service image first
cd op-catalog
docker build -t op-catalog-service:latest .

# Run the tests (must be in the remote directory)
cd ../chainlink-deployments-framework/datastore/catalog/remote
go test -v
```

## Configuration Options

### Environment Variables

- `CATALOG_SERVICE_IMAGE`: The Docker image to use for the catalog service
  - Default: `op-catalog-service:latest`
  - CI: Set to the ECR image URL
  - Example: `export CATALOG_SERVICE_IMAGE="123456789.dkr.ecr.us-east-1.amazonaws.com/op-catalog-service:latest"`

- `CATALOG_GRPC_ADDRESS`: Connect to an existing catalog service instead of starting containers
  - Example: `export CATALOG_GRPC_ADDRESS="localhost:8080"`

## Running Without Testcontainers

If you prefer to manage the services manually:

### Option 1: Docker Compose

```bash
cd op-catalog
docker-compose up -d

# Wait for the service to be ready
sleep 5

# Run tests pointing to the local service
cd ../chainlink-deployments-framework/datastore/catalog/remote
export CATALOG_GRPC_ADDRESS="localhost:8080"
go test -v
```

## CI/CD Integration

The CI workflow automatically:
1. Downloads the latest Catalog service image from ECR
2. Runs the tests with testcontainers using the downloaded image
3. Generates coverage reports

See `.github/workflows/pull-request-main.yml` for the full configuration.

## Test Workflow

The test setup follows this flow:

1. **TestMain** (`main_test.go`) - Entry point that:
   - Checks for `CATALOG_GRPC_ADDRESS` environment variable
   - If not set, starts testcontainers automatically
   - Sets up PostgreSQL and Catalog service
   - Runs all tests
   - Cleans up containers

2. **TestContainerSetup** (`testcontainer_setup.go`) - Manages:
   - Docker network creation
   - PostgreSQL container startup
   - Database migration via catalog service
   - Catalog service container startup
   - Port mapping and connection details
