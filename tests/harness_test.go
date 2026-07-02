package tests

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// startProvisionedPostgres builds the dev Postgres image (which pre-creates the
// role-aligned users) and starts a fresh container, returning its host and mapped
// 5432 port. The container is terminated when the test ends. Each caller gets its
// own clean database.
func startProvisionedPostgres(t *testing.T) (host, port string) {
	t.Helper()
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    "docker",
				Dockerfile: "Dockerfile",
				KeepImage:  true,
			},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_DB":       testDatabase,
			},
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(180 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("build/start dev postgres: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err = container.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	mapped, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("mapped port: %v", err)
	}
	return host, mapped.Port()
}
