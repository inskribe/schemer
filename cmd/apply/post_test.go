package apply

import (
	"context"
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
		request  *deltaRequest
		expected map[int]bool
	}{
		{
			name:     "Load_All",
			request:  &deltaRequest{},
			expected: map[int]bool{0: true, 1: true},
		},
		{
			name:     "From_001",
			request:  &deltaRequest{From: tu.Ptr(1)},
			expected: map[int]bool{1: true},
		},
		{
			name:     "To_000",
			request:  &deltaRequest{To: tu.Ptr(0)},
			expected: map[int]bool{0: true},
		},
		{
			name:     "Cherry_pick",
			request:  &deltaRequest{Cherries: &map[int]bool{1: true}},
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

	expected := map[int]postStatusEnum{0: 1, 1: 0, 2: 2}
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
		request  applyCommandArgs
		expected map[int]postStatusEnum
	}{
		{
			name:     "Apply_All",
			request:  applyCommandArgs{},
			expected: map[int]postStatusEnum{0: 2, 1: 2, 2: 0, 3: 0},
		},
		{
			name:     "From_001",
			request:  applyCommandArgs{fromTag: "001"},
			expected: map[int]postStatusEnum{0: 1, 1: 2, 2: 0, 3: 0},
		},
		{
			name:     "To_000",
			request:  applyCommandArgs{toTag: "000"},
			expected: map[int]postStatusEnum{0: 2, 1: 1, 2: 0, 3: 0},
		},
		{
			name:     "Cherry_pick",
			request:  applyCommandArgs{cherryPickedVersions: []string{"000", "001"}},
			expected: map[int]postStatusEnum{0: 2, 1: 2, 2: 0, 3: 0},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tu.SetupTestTable(t)
			if _, err := tu.SharedConnection.Exec(context.Background(), insertStatment); err != nil {
				t.Fatalf("failed to insert mock data: %v", err)
			}

			applyRequest = tc.request
			if err := executePostCommand(tu.SharedConnection, context.Background()); err != nil {
				t.Fatalf("failed to execute post command: %v", err)
			}

			rows, err := tu.SharedConnection.Query(context.Background(),
				`SELECT tag, post_status FROM schemer`)
			if err != nil {
				t.Fatalf("failed to query post_status: %v", err)
			}
			defer rows.Close()

			actual := map[int]postStatusEnum{}

			for rows.Next() {
				var tag int
				var status postStatusEnum
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
