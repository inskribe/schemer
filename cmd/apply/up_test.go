package apply

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/inskribe/schemer/internal/templates"
	"github.com/inskribe/schemer/internal/utils"
	tu "github.com/inskribe/schemer/internal/utils/testutils"
)

func TestLoadUpDeltas(t *testing.T) {
	tempDir := tu.CreateTestDeltaFiles(t)

	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	testCases := []struct {
		name    string
		request *DeltaRequest
		verify  func(args *DeltaRequest)
	}{
		{
			name:    "valid_post_status",
			request: &DeltaRequest{},
			verify: func(args *DeltaRequest) {
				deltas, err := loadUpDeltas(args)
				if err != nil {
					t.Fatalf("loadUpDeltas failed: %v", err)
				}
				if delta, ok := deltas[1]; !ok || delta.PostStatus != Pending {
					t.Fatalf("expected 001 to be Pending, got: %+v", delta)
				}
				for _, tag := range []int{2, 3} {
					if _, ok := deltas[tag]; !ok {
						t.Fatalf("expected delta tag %03d", tag)
					}
				}
			},
		},
		{
			name: "Cherry_Pick",
			request: &DeltaRequest{
				Cherries: &map[int]bool{1: true, 3: true},
			},
			verify: func(args *DeltaRequest) {
				deltas, err := loadUpDeltas(args)
				if err != nil {
					t.Fatalf("loadUpDeltas failed: %v", err)
				}
				if delta, ok := deltas[1]; !ok || delta.PostStatus != Pending {
					t.Fatalf("expected 001 to be Pending, got: %+v", delta)
				}
				if _, ok := deltas[3]; !ok {
					t.Fatalf("expected 003 to be loaded")
				}
				if _, ok := deltas[2]; ok {
					t.Fatalf("delta 002 loaded expected: 001, 003")
				}
			},
		},
		{
			name:    "From_002",
			request: &DeltaRequest{From: tu.Ptr(2)},
			verify: func(args *DeltaRequest) {
				deltas, err := loadUpDeltas(args)
				if err != nil {
					t.Fatalf("loadUpDeltas failed: %v", err)
				}
				if _, ok := deltas[1]; ok {
					t.Fatalf("delta 001 loaded, expected delta 002,003")
				}
				if _, ok := deltas[2]; !ok {
					t.Fatalf("delta 002 failed to load")
				}
				if _, ok := deltas[3]; !ok {
					t.Fatalf("delta 003 failed to load")
				}
			},
		},
		{
			name:    "To_001",
			request: &DeltaRequest{To: tu.Ptr(1)},
			verify: func(args *DeltaRequest) {
				deltas, err := loadUpDeltas(args)
				if err != nil {
					t.Fatalf("loadUpDeltas failed: %v", err)
				}
				if _, ok := deltas[1]; !ok {
					t.Fatalf("delta 001 failed to load")
				}
				if _, ok := deltas[2]; ok {
					t.Fatalf("delta 002 loaded, expected delta 001")
				}
				if _, ok := deltas[3]; ok {
					t.Fatalf("delta 003 loaded, expected delta 001")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.verify(tc.request)
		})
	}
}

func TestApplyUpDeltas(t *testing.T) {
	tu.SetupTestTable(t)

	tempDir := t.TempDir()
	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}
	upRequest = CommandArgs{PruneNoOp: false}

	deltas := map[int]UpDelta{
		0: {Tag: 0, Data: []byte(""), PostStatus: Pending},
		1: {Tag: 1, Data: []byte(""), PostStatus: NoExist},
	}
	applied := map[int]bool{}

	schemerArgs := templates.SchemerTemplateArgs{
		TableName: "schemer",
	}

	if err := schemerArgs.WriteTemplate(tempDir); err != nil {
		t.Fatalf("failed to write table template: %v", err)
	}

	if err := applyUpDeltas(applied, deltas, tu.SharedConnection, context.Background()); err != nil {
		t.Fatalf("failed to apply deltas: %v", err)
	}

	verifyStatement := `SELECT tag, post_status FROM schemer`
	rows, err := tu.SharedConnection.Query(context.Background(), verifyStatement)
	if err != nil {
		t.Fatalf("query: [%s] failed: %v", verifyStatement, err)
	}

	for rows.Next() {
		var tag int
		var status PostStatusEnum

		if err := rows.Scan(&tag, &status); err != nil {
			t.Fatalf("row scan failed: %v", err)
		}
		expected, ok := deltas[tag]
		if !ok {
			t.Errorf("unexpected tag %03d found in table", tag)
			continue
		}

		if status != expected.PostStatus {
			t.Errorf("tag %03d: expected post status %v, got %v", tag, expected.PostStatus, status)
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("error after row iteration: %v", err)
	}
}

func TestExecuteUpCommand(t *testing.T) {
	tu.SetupTestTable(t)

	tempDir := tu.CreateTestDeltaFiles(t)
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	upRequest = CommandArgs{
		cherryPickedVersions: []string{"000", "002"},
	}

	schemerArgs := templates.SchemerTemplateArgs{
		TableName: "schemer",
	}

	if err := schemerArgs.WriteTemplate(tempDir); err != nil {
		t.Fatalf("failed to write table template: %v", err)
	}

	if err := executeUpCommand(tu.SharedConnection, context.Background()); err != nil {
		t.Fatalf("%v", err)
	}

	verifyStatement := `SELECT tag, post_status FROM schemer WHERE tag IN ($1,$2)`

	rows, err := tu.SharedConnection.Query(context.Background(), verifyStatement, []any{1, 2}...)
	if err != nil {
		t.Fatalf("failed to verify deltas applied: %v", err)
	}

	deltas := map[int]PostStatusEnum{
		1: Pending,
		2: NoExist,
	}

	for rows.Next() {
		var tag int
		var status PostStatusEnum

		rows.Scan(&tag, &status)

		post, ok := deltas[tag]
		if !ok {
			t.Fatalf("expected delta: 001, 002, recived %03d", tag)
		}
		if status != post {
			t.Fatalf("expected post status %s, recived %s", post.String(), status.String())
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("error after row iteration: %v", err)
	}
}

func TestLoadUpDeltas_Recursive(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"001_root.up.sql":                  "-- root up",
		"001_root.post.sql":                "-- root post",
		"users/002_add_user.up.sql":        "-- users up",
		"users/002_add_user.post.sql":      "-- users post",
		"billing/003_add_invoice.up.sql":   "-- billing up",
		"billing/003_add_invoice.down.sql": "-- billing down",
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

	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	deltas, err := loadUpDeltas(&DeltaRequest{})
	if err != nil {
		t.Fatalf("loadUpDeltas failed: %v", err)
	}

	if len(deltas) != 3 {
		t.Fatalf("expected 3 up deltas, received %d", len(deltas))
	}

	if delta, ok := deltas[1]; !ok {
		t.Fatalf("expected delta 001 to be loaded")
	} else if delta.PostStatus != Pending {
		t.Fatalf("expected delta 001 post status Pending, got %+v", delta.PostStatus)
	}

	if delta, ok := deltas[2]; !ok {
		t.Fatalf("expected delta 002 to be loaded")
	} else if delta.PostStatus != Pending {
		t.Fatalf("expected delta 002 post status Pending, got %+v", delta.PostStatus)
	}

	if delta, ok := deltas[3]; !ok {
		t.Fatalf("expected delta 003 to be loaded")
	} else if delta.PostStatus != NoExist {
		t.Fatalf("expected delta 003 post status NoExist, got %+v", delta.PostStatus)
	}
}

func TestLoadUpDeltas_Recursive_FromTo(t *testing.T) {
	tempDir := t.TempDir()

	files := []string{
		"root/001_root.up.sql",
		"users/002_add_user.up.sql",
		"billing/003_add_invoice.up.sql",
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

	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	deltas, err := loadUpDeltas(&DeltaRequest{From: tu.Ptr(2), To: tu.Ptr(2)})
	if err != nil {
		t.Fatalf("loadUpDeltas failed: %v", err)
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
