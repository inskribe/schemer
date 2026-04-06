package apply

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/inskribe/schemer/internal/utils"
	tu "github.com/inskribe/schemer/internal/utils/testutils"
)

func TestDownCommand(t *testing.T) {
	tu.SetupTestTable(t)
	tempDir := tu.CreateTestDeltaFiles(t)

	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	testCases := []struct {
		name    string
		request CommandArgs
		verify  func(t *testing.T)
	}{
		{
			name:    "Apply last",
			request: CommandArgs{},
			verify: func(t *testing.T) {
				var count int
				err := tu.SharedConnection.QueryRow(context.Background(),
					`SELECT COUNT(*) FROM schemer`).Scan(&count)
				if err != nil {
					t.Fatalf("failed to count rows: %v", err)
				}
				if count != 3 {
					t.Fatalf("expected three rows,but found %d", count)
				}
			},
		},
		{
			name: "Cherry_Pick",
			request: CommandArgs{
				cherryPickedVersions: []string{"000", "003"},
			},
			verify: func(t *testing.T) {
				var count int
				err := tu.SharedConnection.QueryRow(context.Background(),
					`SELECT COUNT(*) FROM schemer WHERE tag IN (0, 3)`).Scan(&count)
				if err != nil {
					t.Fatalf("failed to count rows: %v", err)
				}
				if count != 0 {
					t.Fatalf("expected zero rows,but found %d", count)
				}
			},
		},
		{
			name:    "From_002",
			request: CommandArgs{fromTag: "002"},
			verify: func(t *testing.T) {
				var count int
				err := tu.SharedConnection.QueryRow(context.Background(),
					`SELECT COUNT(*) FROM schemer WHERE tag IN (0,1,2)`).Scan(&count)
				if err != nil {
					t.Fatalf("failed to count rows: %v", err)
				}
				if count != 0 {
					t.Fatalf("expected zero rows,but found %d", count)
				}
			},
		},
		{
			name:    "To_001",
			request: CommandArgs{toTag: "001"},
			verify: func(t *testing.T) {
				var count int
				err := tu.SharedConnection.QueryRow(context.Background(),
					`SELECT COUNT(*) FROM schemer WHERE tag IN (3,2,1)`).Scan(&count)
				if err != nil {
					t.Fatalf("failed to count rows: %v", err)
				}
				if count != 0 {
					t.Fatalf("expected zero rows,but found %d", count)
				}
			},
		},
	}

	utils.WithConn = func(connString string, fn func(*pgx.Conn, context.Context) error) error {
		return fn(tu.SharedConnection, context.Background())
	}
	originalState := parseApplyCommand
	parseApplyCommand = func(request *CommandArgs) error {
		return nil
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tu.SetupTestTable(t)

			insertStatement := `INSERT INTO schemer (tag) VALUES (0),(1),(2),(3)`
			if _, err := tu.SharedConnection.Exec(context.Background(), insertStatement); err != nil {
				t.Fatalf("failed to insert statement: %s\n v\nadditionaly: %v", insertStatement, err)
			}

			downRequest = tc.request

			downCmd.Run(&cobra.Command{}, []string{})

			tc.verify(t)
		})
	}

	parseApplyCommand = originalState
}

func TestLoadDownDeltas(t *testing.T) {
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
			name:    "load_all",
			request: &DeltaRequest{},
			verify: func(args *DeltaRequest) {
				deltas, err := loadDownDeltas(args)
				if err != nil {
					t.Fatalf("loadUpDeltas failed: %v", err)
				}
				for _, tag := range []int{0, 1, 2, 3} {
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
				deltas, err := loadDownDeltas(args)
				if err != nil {
					t.Fatalf("loadUpDeltas failed: %v", err)
				}
				if _, ok := deltas[1]; !ok {
					t.Fatalf("expected 001 to be loaded")
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
				deltas, err := loadDownDeltas(args)
				if err != nil {
					t.Fatalf("loadUpDeltas failed: %v", err)
				}
				if _, ok := deltas[0]; !ok {
					t.Fatalf("delta 000 failed to load")
				}
				if _, ok := deltas[1]; !ok {
					t.Fatalf("delta 001 failed to load")
				}
				if _, ok := deltas[2]; !ok {
					t.Fatalf("delta 002 failed to load")
				}
				if _, ok := deltas[3]; ok {
					t.Fatalf("delta 003 loaded, expected delta 000,001,002")
				}
			},
		},
		{
			name:    "To_001",
			request: &DeltaRequest{To: tu.Ptr(1)},
			verify: func(args *DeltaRequest) {
				deltas, err := loadDownDeltas(args)
				if err != nil {
					t.Fatalf("loadDownDeltas failed: %v", err)
				}
				if _, ok := deltas[0]; ok {
					t.Fatalf("delta 000 loaded, expected delta 001 and above only")
				}
				if _, ok := deltas[1]; !ok {
					t.Fatalf("delta 001 failed to load")
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
			name:    "Last",
			request: &DeltaRequest{LastTag: tu.Ptr(3)},
			verify: func(args *DeltaRequest) {
				deltas, err := loadDownDeltas(args)
				if err != nil {
					t.Fatalf("failed to load down deltas: %v", err)
				}

				if len(deltas) > 1 {
					t.Fatalf("loadDownDeltas() returned %d deltas, expected 1 deltas", len(deltas))
				}
				if _, ok := deltas[3]; !ok {
					t.Fatalf("expected delta 003 to be loaded")
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

func TestApplyForLastUpDelta(t *testing.T) {
	tu.SetupTestTable(t)
	tempDir := tu.CreateTestDeltaFiles(t)

	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	insertStatement := `INSERT INTO schemer (tag) VALUES (0),(1),(2)`
	if _, err := tu.SharedConnection.Exec(context.Background(), insertStatement); err != nil {
		t.Fatalf("failed to insert statement: %s\n v\nadditionaly: %v", insertStatement, err)
	}

	if err := applyForLastUpDelta(tu.SharedConnection, context.Background()); err != nil {
		t.Fatalf("failed to apply down delta: %v", err)
	}

	verifyStatement := `SELECT tag FROM schemer WHERE tag = 2`

	row := tu.SharedConnection.QueryRow(context.Background(), verifyStatement)
	var tag int
	var status PostStatusEnum
	var createdAt time.Time
	if err := row.Scan(&tag, &status, &createdAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return
		} else {
			t.Fatalf("failed to query row: %v", err)
		}
	}

	t.Fatalf("expected delta 002 not to be applied, recived tag: %s, post_status: %s", utils.ToPrefix(tag), status.String())
}

func TestExecuteDownCommand(t *testing.T) {
	tempDir := tu.CreateTestDeltaFiles(t)

	utils.GetDeltaPath = func() (string, error) {
		return tempDir, nil
	}

	testCases := []struct {
		name    string
		request CommandArgs
		verify  func()
	}{
		{
			name:    "load_all",
			request: CommandArgs{},
			verify: func() {
				var count int
				err := tu.SharedConnection.QueryRow(context.Background(),
					`SELECT COUNT(*) FROM schemer`).Scan(&count)
				if err != nil {
					t.Fatalf("failed to count rows: %v", err)
				}
				if count != 0 {
					t.Fatalf("expected zero rows,but found %d", count)
				}
			},
		},
		{
			name: "Cherry_Pick",
			request: CommandArgs{
				cherryPickedVersions: []string{"000", "003"},
			},
			verify: func() {
				var count int
				err := tu.SharedConnection.QueryRow(context.Background(),
					`SELECT COUNT(*) FROM schemer WHERE tag IN (0, 3)`).Scan(&count)
				if err != nil {
					t.Fatalf("failed to count rows: %v", err)
				}
				if count != 0 {
					t.Fatalf("expected zero rows,but found %d", count)
				}
			},
		},
		{
			name:    "From_002",
			request: CommandArgs{fromTag: "002"},
			verify: func() {
				var count int
				err := tu.SharedConnection.QueryRow(context.Background(),
					`SELECT COUNT(*) FROM schemer WHERE tag IN (0,1,2)`).Scan(&count)
				if err != nil {
					t.Fatalf("failed to count rows: %v", err)
				}
				if count != 0 {
					t.Fatalf("expected zero rows,but found %d", count)
				}
			},
		},
		{
			name:    "To_001",
			request: CommandArgs{toTag: "001"},
			verify: func() {
				var count int
				err := tu.SharedConnection.QueryRow(context.Background(),
					`SELECT COUNT(*) FROM schemer WHERE tag IN (3,2,1)`).Scan(&count)
				if err != nil {
					t.Fatalf("failed to count rows: %v", err)
				}
				if count != 0 {
					t.Fatalf("expected zero rows,but found %d", count)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tu.SetupTestTable(t)

			insertStatement := `INSERT INTO schemer (tag) VALUES (0),(1),(2),(3)`
			if _, err := tu.SharedConnection.Exec(context.Background(), insertStatement); err != nil {
				t.Fatalf("failed to insert statement: %s\n v\nadditionaly: %v", insertStatement, err)
			}

			downRequest = tc.request
			if err := executeDownCommand(tu.SharedConnection, context.Background()); err != nil {
				t.Fatalf("Failed test case %s: %v", tc.name, err)
			}
			tc.verify()
		})
	}
}

func TestLoadDownDeltas_Recursive(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"001_root.down.sql":                "-- root down",
		"users/002_add_user.down.sql":      "-- users down",
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

	deltas, err := loadDownDeltas(&DeltaRequest{})
	if err != nil {
		t.Fatalf("failed to load down deltas: %v", err)
	}

	if len(deltas) != 3 {
		t.Fatalf("expected 3 down deltas, received %d", len(deltas))
	}

	for _, tag := range []int{1, 2, 3} {
		if _, ok := deltas[tag]; !ok {
			t.Fatalf("expected delta %03d to be loaded", tag)
		}
	}
}

func TestLoadDownDeltas_Recursive_FromTo(t *testing.T) {
	tempDir := t.TempDir()

	files := []string{
		"001_root.down.sql",
		"users/002_add_user.down.sql",
		"billing/003_add_invoice.down.sql",
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

	deltas, err := loadDownDeltas(&DeltaRequest{
		From: tu.Ptr(2),
		To:   tu.Ptr(2),
	})
	if err != nil {
		t.Fatalf("failed to load down deltas: %v", err)
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

func TestLoadDownDeltas_LastTag_Recursive(t *testing.T) {
	tempDir := t.TempDir()

	files := []string{
		"001_root.down.sql",
		"users/002_add_user.down.sql",
		"billing/003_add_invoice.down.sql",
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

	deltas, err := loadDownDeltas(&DeltaRequest{
		LastTag: tu.Ptr(3),
	})
	if err != nil {
		t.Fatalf("failed to load down deltas: %v", err)
	}

	if len(deltas) != 1 {
		t.Fatalf("expected 1 delta, received %d", len(deltas))
	}

	if _, ok := deltas[3]; !ok {
		t.Fatalf("expected delta 003 to be loaded")
	}
}
