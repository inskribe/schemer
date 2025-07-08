package testutils

import (
	"os"
	"path/filepath"
	"testing"
)

func CreateTestDeltaFiles(t *testing.T) string {
	tempDir := t.TempDir()

	files := []struct {
		filename string
		data     []byte
	}{
		{
			filename: "ignore.up.sql",
			data:     []byte("-- ignore"),
		},
		{
			filename: "000_test.up.sql",
			data:     []byte("-- up"),
		},
		{
			filename: "000_test.post.sql",
			data:     []byte("-- post"),
		},
		{
			filename: "000_test.down.sql",
			data:     []byte("-- down"),
		},
		{
			filename: "001_test.up.sql",
			data:     []byte("-- up"),
		},
		{
			filename: "001_test.post.sql",
			data:     []byte("-- post"),
		},
		{
			filename: "001_test.down.sql",
			data:     []byte("-- down"),
		},
		{
			filename: "002_test.up.sql",
			data:     []byte("-- up"),
		},
		{
			filename: "002_test.down.sql",
			data:     []byte("-- down"),
		},
		{
			filename: "003_test.up.sql",
			data:     []byte("-- up"),
		},
		{
			filename: "003_test.down.sql",
			data:     []byte("-- down"),
		},
	}

	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tempDir, file.filename), file.data, 0644); err != nil {
			t.Fatalf("failed to write %s: %v", file.filename, err)
		}
	}
	return tempDir

}
