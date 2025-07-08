package apply

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/inskribe/schemer/internal/glog"
	"github.com/inskribe/schemer/internal/utils/testutils"
)

func TestMain(m *testing.M) {
	glog.InitializeLogger(true)
	path, err := testutils.GetTestWorkingDir()
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
		fmt.Printf("failed to setup database connection: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := testutils.TearDown(); err != nil {
		fmt.Printf("database tear down failure: %v", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func TestIsNoOpSql(t *testing.T) {
	mockData := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name: "only line comments",
			input: `
-- SELECT * FROM users;
-- another comment
`,
			expected: true,
		},
		{
			name:     "only block comment, single-line",
			input:    `/* SELECT * FROM users; */`,
			expected: true,
		},
		{
			name: "only block comment, multi-line",
			input: `
/* 
    SELECT * 
    FROM users; 
*/
`,
			expected: true,
		},
		{
			name: "line and block comments",
			input: `
-- top comment
/*
  block comment
*/
-- trailing comment
`,
			expected: true,
		},
		{
			name: "actual SQL after comments",
			input: `
-- preamble
/* comment */
SELECT * FROM users;
`,
			expected: false,
		},
		{
			name: "actual SQL before comment",
			input: `
SELECT * FROM users;
/* this is ignored */
`,
			expected: false,
		},
		{
			name: "incomplete block comment followed by SQL",
			input: `
/* start block
SELECT * FROM users;
`,
			expected: true,
		},
		{
			name: "SQL after block comment",
			input: `
/* start */
-- ignored
/* block */ SELECT * FROM users; 
`,
			expected: false,
		},
		{
			name: "SQL between block comment",
			input: `
/* start */
-- ignored
/* block */ SELECT * FROM users; /**/
`,
			expected: false,
		},
	}

	for _, mock := range mockData {
		t.Run(mock.name, func(t *testing.T) {
			result := IsNoOpSql(mock.input)
			if result != mock.expected {
				t.Errorf("IsNoOpSql() = %v, expected %v\nInput:\n%s", result, mock.expected, mock.input)
			}
		})
	}
}

func TestPruneNoOp(t *testing.T) {
	glog.InitializeLogger(true)
	testData := map[int][]byte{
		1: []byte("-- comment only"),
		2: []byte("SELECT * FROM users;"),
		3: []byte("/* block */"),
		4: []byte("INSERT INTO x VALUES (1);"),
		5: []byte(`
			-- only comment
			/* and another */
		`),
	}

	expectedRemaining := map[int]bool{
		2: true,
		4: true,
	}

	PruneNoOp(&testData)

	for tag := range expectedRemaining {
		if _, ok := testData[tag]; !ok {
			t.Errorf("Expected tag %d to remain, but it was pruned", tag)
		}
	}

	for tag := range testData {
		if !expectedRemaining[tag] {
			t.Errorf("Expected tag %d to be pruned, but it remained", tag)
		}
	}
}

func BenchmarkPruneNoOp(b *testing.B) {
	glog.InitializeLogger(true)
	data := make(map[int][]byte, 10000)
	for i := 0; i < 10000; i++ {
		if i%5 == 0 {
			data[i] = []byte("SELECT * FROM table;")
		} else {
			data[i] = []byte("-- just a comment")
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clone := make(map[int][]byte, len(data))
		for k, v := range data {
			clone[k] = v
		}
		PruneNoOp(&clone)
	}
}

func TestPruneNoOpUp(t *testing.T) {
	glog.InitializeLogger(true)

	testData := map[int]upDelta{
		1: {Tag: 1, Data: []byte("-- just a comment")},
		2: {Tag: 2, Data: []byte("SELECT * FROM users;"), PostStatus: Pending},
		3: {Tag: 3, Data: []byte("/* block comment */")},
		4: {Tag: 4, Data: []byte("INSERT INTO table VALUES (1);")},
		5: {Tag: 5, Data: []byte(`
			-- only comment
			/* and another */
		`)},
	}

	expectedRemaining := map[int]bool{
		2: true,
		4: true,
	}

	PruneNoOpUp(&testData)

	for tag := range expectedRemaining {
		if _, ok := testData[tag]; !ok {
			t.Errorf("Expected tag %d to remain, but it was pruned", tag)
		}
	}

	for tag := range testData {
		if !expectedRemaining[tag] {
			t.Errorf("Expected tag %d to be pruned, but it remained", tag)
		}
	}
}

func BenchmarkPruneNoOpUp(b *testing.B) {
	glog.InitializeLogger(true)

	base := make(map[int]upDelta, 10000)
	for i := 0; i < 10000; i++ {
		var sql []byte
		if i%5 == 0 {
			sql = []byte("SELECT * FROM users;")
		} else {
			sql = []byte("-- no-op")
		}
		base[i] = upDelta{Tag: i, Data: sql, PostStatus: Pending}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clone := make(map[int]upDelta, len(base))
		for k, v := range base {
			clone[k] = v
		}
		PruneNoOpUp(&clone)
	}
}

func TestGetRequestedDeltas(t *testing.T) {
	mockData := []struct {
		name     string
		args     applyCommandArgs
		expected bool
		verify   func(req *deltaRequest) bool
	}{
		{
			name:     "apply all",
			args:     applyCommandArgs{},
			expected: true,
			verify: func(req *deltaRequest) bool {
				return req != nil && req.Cherries == nil && req.From == nil && req.To == nil
			},
		}, {
			name:     "apply from",
			args:     applyCommandArgs{fromTag: "001"},
			expected: true,
			verify: func(req *deltaRequest) bool {
				return *req.From == 1 && req.To == nil && req.Cherries == nil
			},
		}, {
			name:     "apply to",
			args:     applyCommandArgs{toTag: "001"},
			expected: true,
			verify: func(req *deltaRequest) bool {
				return *req.To == 1 && req.From == nil && req.Cherries == nil
			},
		}, {
			name:     "apply range",
			args:     applyCommandArgs{fromTag: "000", toTag: "003"},
			expected: true,
			verify: func(req *deltaRequest) bool {
				return *req.To == 3 && *req.From == 0 && req.Cherries == nil
			},
		}, {
			name:     "apply cherries",
			args:     applyCommandArgs{cherryPickedVersions: []string{"001", "003", "999"}},
			expected: true,
			verify: func(req *deltaRequest) bool {
				valid := req.To == nil && req.From == nil
				if (*req.Cherries)[1] && (*req.Cherries)[3] && (*req.Cherries)[999] {
					return valid
				}
				return false
			},
		},
	}

	for _, data := range mockData {
		t.Run(data.name, func(t *testing.T) {
			req, err := data.args.getRequestedDeltas()
			if err != nil {
				t.Errorf("%v", err)
			}
			if ok := data.verify(req); ok != data.expected {
				t.Errorf("getRequestedDeltas() = %v, expected: %v", ok, data.expected)
			}

		})
	}
}

func TestGetAppliedDeltas(t *testing.T) {
	testutils.SetupTestTable(t)

	testDeltas := []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	var placeholders []string

	for i := range testDeltas {
		placeholders = append(placeholders, fmt.Sprintf("($%d)", i+1))
	}

	insert := fmt.Sprintf(`INSERT INTO schemer (tag) 
	VALUES %s`, strings.Join(placeholders, ","))

	ctx := context.Background()

	if _, err := testutils.SharedConnection.Exec(ctx, insert, testDeltas...); err != nil {
		t.Errorf("failed to stage data: insert: %s", err.Error())
	}

	appliedDeltas, err := getAppliedDeltas(testutils.SharedConnection, ctx)
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	for _, tag := range testDeltas {
		tagInt, ok := tag.(int)
		if !ok {
			t.Fatalf("tag is not an int: %v (type %T)", tag, tag)
		}

		if _, ok := appliedDeltas[tagInt]; !ok {
			t.Errorf("expected applied deltas to contain delta tag: %d", tag)
		}
	}
}
