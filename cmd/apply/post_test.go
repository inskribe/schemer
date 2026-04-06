package apply

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/inskribe/schemer/internal/utils"
	tu "github.com/inskribe/schemer/internal/utils/testutils"
)

func TestLoadPostDeltas(t *testing.T) {
	tempDir := tu.CreateTestDeltaFiles(t)
	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	testCases := []struct {
		name     string
		request  *DeltaRequest
		expected map[int]bool
	}{
		{
			name:     "Load_All",
			request:  &DeltaRequest{},
			expected: map[int]bool{0: true, 1: true},
		},
		{
			name:     "From_001",
			request:  &DeltaRequest{From: tu.Ptr(1)},
			expected: map[int]bool{1: true},
		},
		{
			name:     "To_000",
			request:  &DeltaRequest{To: tu.Ptr(0)},
			expected: map[int]bool{0: true},
		},
		{
			name:     "Cherry_pick",
			request:  &DeltaRequest{Cherries: &map[int]bool{1: true}},
			expected: map[int]bool{1: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tu.SetupTestTable(t)
			deltas, err := loadPostDeltas(tc.request, tu.SharedConnection, context.Background())
			if err != nil {
				t.Fatalf("failed to load deltas: %v", err)
			}
			for tag := range deltas {
				if _, ok := tc.expected[tag]; !ok {
					t.Fatalf("found unexected tag: %s", utils.ToPrefix(tag))
				}
			}
		})
	}
}

func TestFetchPostStatuses(t *testing.T) {
	tu.SetupTestTable(t)

	if _, err := tu.SharedConnection.Exec(context.Background(),
		`INSERT INTO schemer (tag, post_status) VALUES (0,1),(1,0),(2,2)`); err != nil {
		t.Fatalf("failed to insert mock data: %v", err)
	}

	expected := map[int]PostStatusEnum{0: 1, 1: 0, 2: 2}
	statuses, err := fetchPostStatuses(tu.SharedConnection, context.Background())
	if err != nil {
		t.Fatalf("failed to fetch post statuses: %v", err)
	}

	for tag, status := range statuses {
		expectedStatus, ok := expected[tag]
		if !ok {
			t.Fatalf("found unexpected tag %s", utils.ToPrefix(tag))
		}
		if status != expectedStatus {
			t.Fatalf("post status mismatch, expected %s recieved %s", expectedStatus.String(), status.String())
		}
	}
}

func TestExecutePostCommand(t *testing.T) {
	tempDir := tu.CreateTestDeltaFiles(t)
	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	insertStatment := `INSERT INTO schemer (tag,post_status) 
	VALUES (0,1),(1,1),(2,0),(3,0)`

	testCases := []struct {
		name     string
		request  CommandArgs
		expected map[int]PostStatusEnum
	}{
		{
			name:     "Apply_All",
			request:  CommandArgs{},
			expected: map[int]PostStatusEnum{0: 2, 1: 2, 2: 0, 3: 0},
		},
		{
			name:     "From_001",
			request:  CommandArgs{fromTag: "001"},
			expected: map[int]PostStatusEnum{0: 1, 1: 2, 2: 0, 3: 0},
		},
		{
			name:     "To_000",
			request:  CommandArgs{toTag: "000"},
			expected: map[int]PostStatusEnum{0: 2, 1: 1, 2: 0, 3: 0},
		},
		{
			name:     "Cherry_pick",
			request:  CommandArgs{cherryPickedVersions: []string{"000", "001"}},
			expected: map[int]PostStatusEnum{0: 2, 1: 2, 2: 0, 3: 0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tu.SetupTestTable(t)
			if _, err := tu.SharedConnection.Exec(context.Background(), insertStatment); err != nil {
				t.Fatalf("failed to insert mock data: %v", err)
			}

			postRequest = tc.request
			if err := executePostCommand(tu.SharedConnection, context.Background()); err != nil {
				t.Fatalf("failed to execute post command: %v", err)
			}

			rows, err := tu.SharedConnection.Query(context.Background(),
				`SELECT tag, post_status FROM schemer`)
			if err != nil {
				t.Fatalf("failed to query post_status: %v", err)
			}
			defer rows.Close()

			actual := map[int]PostStatusEnum{}

			for rows.Next() {
				var tag int
				var status PostStatusEnum
				if err := rows.Scan(&tag, &status); err != nil {
					t.Fatalf("failed to scan row: %v", err)
				}
				actual[tag] = status
			}

			if err := rows.Err(); err != nil {
				t.Fatalf("rows iteration error: %v", err)
			}

			for tag, expectedStatus := range tc.expected {
				gotStatus, ok := actual[tag]
				if !ok {
					t.Errorf("expected tag %s to be in DB, but wasn't", utils.ToPrefix(tag))
					continue
				}
				if gotStatus != expectedStatus {
					t.Errorf("tag %s:: expected status %v, got %v", utils.ToPrefix(tag), expectedStatus, gotStatus)
				}
			}
		})
	}
}

func TestLoadPostDeltas_Recursive(t *testing.T) {
	tempDir := t.TempDir()
	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	files := map[string]string{
		"001_root.post.sql":                "-- root post",
		"users/002_add_user.post.sql":      "-- users post",
		"billing/003_add_invoice.post.sql": "-- billing post",
	}

	for rel, contents := range files {
		full := filepath.Join(tempDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), os.ModePerm); err != nil {
			t.Fatalf("failed to create dir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", rel, err)
		}
	}

	tu.SetupTestTable(t)
	if _, err := tu.SharedConnection.Exec(context.Background(),
		`INSERT INTO schemer (tag, post_status) VALUES (1,1),(2,1),(3,1)`); err != nil {
		t.Fatalf("failed to insert mock data: %v", err)
	}

	deltas, err := loadPostDeltas(&DeltaRequest{}, tu.SharedConnection, context.Background())
	if err != nil {
		t.Fatalf("failed to load post deltas: %v", err)
	}

	if len(deltas) != 3 {
		t.Fatalf("expected 3 post deltas, received %d", len(deltas))
	}

	for _, tag := range []int{1, 2, 3} {
		if _, ok := deltas[tag]; !ok {
			t.Fatalf("expected delta %03d to be loaded", tag)
		}
	}
}

func TestLoadPostDeltas_Recursive_FromTo(t *testing.T) {
	tempDir := t.TempDir()
	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	files := []string{
		"001_root.post.sql",
		"users/002_add_user.post.sql",
		"billing/003_add_invoice.post.sql",
	}

	for _, rel := range files {
		full := filepath.Join(tempDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), os.ModePerm); err != nil {
			t.Fatalf("failed to create dir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte("-- test"), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", rel, err)
		}
	}

	tu.SetupTestTable(t)
	if _, err := tu.SharedConnection.Exec(context.Background(),
		`INSERT INTO schemer (tag, post_status) VALUES (1,1),(2,1),(3,1)`); err != nil {
		t.Fatalf("failed to insert mock data: %v", err)
	}

	deltas, err := loadPostDeltas(&DeltaRequest{
		From: tu.Ptr(2),
		To:   tu.Ptr(2),
	}, tu.SharedConnection, context.Background())
	if err != nil {
		t.Fatalf("failed to load post deltas: %v", err)
	}

	if len(deltas) != 1 {
		t.Fatalf("expected 1 delta, received %d", len(deltas))
	}

	if _, ok := deltas[2]; !ok {
		t.Fatalf("expected delta 002 to be loaded")
	}
	if _, ok := deltas[1]; ok {
		t.Fatalf("did not expect delta 001 to be loaded")
	}
	if _, ok := deltas[3]; ok {
		t.Fatalf("did not expect delta 003 to be loaded")
	}
}

func TestLoadPostDeltas_Recursive_Force(t *testing.T) {
	tempDir := t.TempDir()
	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	full := filepath.Join(tempDir, "users", "002_add_user.post.sql")
	if err := os.MkdirAll(filepath.Dir(full), os.ModePerm); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(full, []byte("-- test"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	tu.SetupTestTable(t)

	postoptions.Force = false
	deltas, err := loadPostDeltas(&DeltaRequest{}, tu.SharedConnection, context.Background())
	if err != nil {
		t.Fatalf("failed to load post deltas without force: %v", err)
	}
	if _, ok := deltas[2]; ok {
		t.Fatalf("did not expect delta 002 without force")
	}

	postoptions.Force = true
	defer func() { postoptions.Force = false }()

	deltas, err = loadPostDeltas(&DeltaRequest{}, tu.SharedConnection, context.Background())
	if err != nil {
		t.Fatalf("failed to load post deltas with force: %v", err)
	}
	if _, ok := deltas[2]; !ok {
		t.Fatalf("expected delta 002 with force")
	}
}
