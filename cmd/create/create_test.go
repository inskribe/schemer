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

func TestCreateDeltaFiled(t *testing.T) {
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
