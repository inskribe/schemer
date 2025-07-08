package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

var SharedConnection *pgx.Conn

func GetTestWorkingDir() (string, error) {

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working dir: %v", err)
	}

	parts := strings.Split(cwd, string(os.PathSeparator))

	schemerDirIndex := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "schemer" {
			schemerDirIndex = i
			break
		}
	}

	if schemerDirIndex == -1 {
		return "", fmt.Errorf("failed to locate 'schemer' directory in path: %s", cwd)
	}

	return filepath.Join("/", filepath.Join(parts[:schemerDirIndex+1]...)), nil
}

func SetupTestDatabase() error {

	// ok, err := utils.LoadDotEnv()
	// if err != nil {
	// 	return fmt.Errorf("%s", err.Error())
	// }
	// if !ok {
	// 	return fmt.Errorf("failed to load .env")
	// }

	ctx := context.Background()
	dns, ok := os.LookupEnv("DATABASE_URL")
	if !ok || dns == "" {
		return fmt.Errorf("failed to find env var for mock database.")
	}

	connection, err := pgx.Connect(ctx, dns)
	if err != nil {
		return fmt.Errorf("failed to setup database: connect: %v", err)
	}

	if connection == nil {
		return fmt.Errorf("database connection is nil")
	}

	SharedConnection = connection
	return nil
}

func SetupTestTable(t *testing.T) {
	if SharedConnection == nil {
		t.Fatalf("shared database connection is nil")
	}
	ctx := context.Background()

	_, err := SharedConnection.Exec(ctx, `
		DROP TABLE IF EXISTS schemer;
		CREATE TABLE schemer (
			tag INT PRIMARY KEY,
			post_status INT DEFAULT 0,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	t.Cleanup(func() {
		_, _ = SharedConnection.Exec(ctx, `DROP TABLE IF EXISTS schemer`)
	})
}

func TearDown() error {
	if SharedConnection == nil {
		return fmt.Errorf("SharedConnection was nil at teardown.")
	}

	if err := SharedConnection.Close(context.Background()); err != nil {
		return fmt.Errorf("%s", err.Error())
	}
	return nil
}
