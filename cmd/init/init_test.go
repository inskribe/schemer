package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/inskribe/schemer/internal/glog"
	"github.com/inskribe/schemer/internal/utils/testutils"
	tu "github.com/inskribe/schemer/internal/utils/testutils"
)

func TestMain(m *testing.M) {
	glog.InitializeLogger(true)
	path, err := tu.GetTestWorkingDir()
	if err != nil {
		fmt.Printf("failed to get test working dir: %v", err)
		os.Exit(1)
	}
	if path == "" {
		fmt.Println("recived empty working dir")
		os.Exit(1)
	}

	if err := os.Chdir(path); err != nil {
		fmt.Println("os.Chdir failed:", err)
		os.Exit(1)
	}

	if err := testutils.SetupTestDatabase(); err != nil {
		fmt.Println("failed to setup database:", err)
		os.Exit(1)
	}

	code := m.Run()

	os.Exit(code)

}

func TestCreateDeltasDirectory(t *testing.T) {
	tempDir := t.TempDir()
	deltasDir := filepath.Join(tempDir, "deltas")
	if err := createDeltasDirectory(deltasDir); err != nil {
		t.Fatalf("failed to create deltas directory: %v", err)
	}

	info, err := os.Stat(deltasDir)
	if err != nil {
		t.Fatalf("failed to find deltas directory: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("failed to get directory")
	}
}

func TestCreateEnvFile(t *testing.T) {
	tempDir := t.TempDir()
	if err := createEnvFile(tempDir); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	info, err := os.Stat(filepath.Join(tempDir, ".env"))
	if err != nil {
		t.Fatalf("failed to find .env file: %v", err)
	}
	if info.IsDir() {
		t.Fatalf(".env is a directory")
	}
}

func TestExecuteInitCommand(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory.")
	}

	DatabaseArgs.UrlValue = os.Getenv("DATABASE_URL")

	if err := executeInitCommand(); err != nil {
		t.Fatalf("failed to execute init command: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "deltas")); err != nil {
		t.Fatalf("failed to create deltas dir.")
	}

	if _, err := os.Stat(filepath.Join(tempDir, ".env")); err != nil {
		t.Fatalf("failed to create .env file.")
	}

	var exists bool
	if err := tu.SharedConnection.QueryRow(
		context.Background(),
		`SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'schemer'
		);
	`).Scan(&exists); err != nil {
		t.Fatalf("failed to scan row: %v", err)
	}

	if !exists {
		t.Fatalf("failed to query for schemer table.")
	}
}
