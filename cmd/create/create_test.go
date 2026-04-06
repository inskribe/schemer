package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inskribe/schemer/internal/glog"
	"github.com/inskribe/schemer/internal/utils"
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

	code := m.Run()

	os.Exit(code)
}

func TestDetermineNextTag(t *testing.T) {
	tempDir := tu.CreateTestDeltaFiles(t)
	nextTag, err := determineNextTag(tempDir)
	if err != nil {
		t.Fatalf("failed to get nextTag: %v", err)
	}
	if nextTag != 4 {
		t.Fatalf("expected tag 4 recived %s", utils.ToPrefix(nextTag))
	}
}

func TestDetermineNextTag_Recursive(t *testing.T) {
	tempDir := t.TempDir()

	paths := []string{
		filepath.Join(tempDir, "001_init.up.sql"),
		filepath.Join(tempDir, "001_init.down.sql"),
		filepath.Join(tempDir, "users", "002_add_user.up.sql"),
		filepath.Join(tempDir, "users", "002_add_user.down.sql"),
		filepath.Join(tempDir, "billing", "003_add_invoice.up.sql"),
		filepath.Join(tempDir, "billing", "003_add_invoice.down.sql"),
	}

	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			t.Fatalf("failed to create dir for %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("-- test"), 0o644); err != nil {
			t.Fatalf("failed to create file %s: %v", path, err)
		}
	}

	nextTag, err := determineNextTag(tempDir)
	if err != nil {
		t.Fatalf("failed to get nextTag: %v", err)
	}

	if nextTag != 4 {
		t.Fatalf("expected tag 4 recived %s", utils.ToPrefix(nextTag))
	}
}

func TestCreateDeltaFile(t *testing.T) {
	tempDir := t.TempDir()
	expectedFilename := "test_filename"
	nextTag := 100

	CreateRequest.Post = true

	if err := createDeltaFiles(expectedFilename, nextTag, tempDir); err != nil {
		t.Fatalf("failed to create delta file: %v", err)
	}

	prefix := fmt.Sprintf("100_%s", expectedFilename)
	up := filepath.Join(tempDir, strings.Join([]string{prefix, "up", "sql"}, "."))
	down := filepath.Join(tempDir, strings.Join([]string{prefix, "down", "sql"}, "."))
	post := filepath.Join(tempDir, strings.Join([]string{prefix, "post", "sql"}, "."))

	if _, err := os.Stat(up); err != nil {
		t.Fatalf("failed to fetch up file: %v", err)
	}

	if _, err := os.Stat(down); err != nil {
		t.Fatalf("failed to fetch down file: %v", err)
	}

	if _, err := os.Stat(post); err != nil {
		t.Fatalf("failed to fetch post file: %v", err)
	}
}

func TestCreateDeltaFile_SubDir(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "users")
	expectedFilename := "test_filename"
	nextTag := 100

	CreateRequest.Post = true

	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	if err := createDeltaFiles(expectedFilename, nextTag, targetDir); err != nil {
		t.Fatalf("failed to create delta file: %v", err)
	}

	prefix := fmt.Sprintf("100_%s", expectedFilename)
	up := filepath.Join(targetDir, strings.Join([]string{prefix, "up", "sql"}, "."))
	down := filepath.Join(targetDir, strings.Join([]string{prefix, "down", "sql"}, "."))
	post := filepath.Join(targetDir, strings.Join([]string{prefix, "post", "sql"}, "."))

	if _, err := os.Stat(up); err != nil {
		t.Fatalf("failed to fetch up file: %v", err)
	}

	if _, err := os.Stat(down); err != nil {
		t.Fatalf("failed to fetch down file: %v", err)
	}

	if _, err := os.Stat(post); err != nil {
		t.Fatalf("failed to fetch post file: %v", err)
	}
}
