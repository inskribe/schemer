/*
Copyright © 2025 Roy Sowers <inskribe@inskribestudio.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package apply

import (
	"bufio"
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/inskribe/schemer/internal/errschemer"
	"github.com/inskribe/schemer/internal/glog"
	"github.com/inskribe/schemer/internal/utils"
)

// PruneNoOpUp removes no-op SQL deltas from the provided map in-place for upDelta.
// Uses concurrent workers to scan for and discard deltas that contain only comments or whitespace.
//
// Params:
//   - data: pointer to a map of delta tags to raw SQL bytes; will be mutated directly
func PruneNoOpUp(data *map[int]upDelta) {
	var group sync.WaitGroup

	noOps := make(chan int, len(*data))

	for tag, deltas := range *data {
		group.Add(1)
		go func(tag int, contents string) {
			defer group.Done()
			if IsNoOpSql(contents) {
				noOps <- tag
			}

		}(tag, string(deltas.Data))
	}
	group.Wait()
	close(noOps)

	for tag := range noOps {
		delete(*data, tag)
		glog.Warn("Skipping delta %s, would be a no-op database call.", utils.ToPrefix(tag))
	}

}

// PruneNoOp removes no-op SQL deltas from the provided map in-place.
// Uses concurrent workers to scan for and discard deltas that contain only comments or whitespace.
//
// Params:
//   - data: pointer to a map of delta tags to raw SQL bytes; will be mutated directly
func PruneNoOp(data *map[int][]byte) {
	var group sync.WaitGroup

	noOps := make(chan int, len(*data))

	for tag, contents := range *data {
		group.Add(1)
		go func(tag int, contents string) {
			defer group.Done()
			if IsNoOpSql(contents) {
				noOps <- tag
			}

		}(tag, string(contents))
	}
	group.Wait()
	close(noOps)

	for tag := range noOps {
		delete(*data, tag)
		glog.Warn("Skipping delta %s, would be a no-op database call.", utils.ToPrefix(tag))
	}
}

// IsNoOpSql returns true if the SQL string contains no executable statements.
// Ignores empty lines, line comments (--), and block comments (/* */).
//
// Params:
//   - data: the SQL content to evaluate
//
// Returns:
//   - bool: true if the SQL contains only comments or whitespace; false otherwise
func IsNoOpSql(data string) bool {
	scanner := bufio.NewScanner(strings.NewReader(data))
	var inBlockComment bool = false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		if inBlockComment {
			if strings.Contains(line, "*/") {
				inBlockComment = false
			}
			continue
		}

		if strings.HasPrefix(line, "/*") {
			if !strings.Contains(line, "*/") {
				inBlockComment = true
				continue
			}
			parts := strings.SplitAfter(line, "*/")
			if len(parts) > 1 {
				if IsNoOpSql(parts[1]) {
					continue
				}
			}
			return false
		}
		return false
	}
	return true
}

// getRequestedDeltas parses the user's migration flags into a DeltaRequest.
// Converts --from, --to, and --cherry-pick CLI inputs into a structured request.
//
// Returns:
//   - *deltaRequest: the constructed delta request with range or specific tags
//   - error: non-nil if any tag is not a valid integer
func (args applyCommandArgs) getRequestedDeltas() (*deltaRequest, error) {
	var result deltaRequest
	if args.fromTag != "" {
		val, err := strconv.Atoi(args.fromTag)
		if err != nil {
			return nil, &errschemer.SchemerErr{
				Code:    "0027",
				Message: "failed to convert --from tag" + args.fromTag,
				Err:     err,
			}
		}
		result.From = &val
	}

	if args.toTag != "" {
		val, err := strconv.Atoi(args.toTag)
		if err != nil {
			return nil, &errschemer.SchemerErr{
				Code:    "0026",
				Message: "failed to convert --to tag" + args.toTag,
				Err:     err,
			}

		}
		result.To = &val
	}

	if len(args.cherryPickedVersions) <= 0 {
		return &result, nil
	}

	cherries := make(map[int]bool)
	for _, raw := range args.cherryPickedVersions {
		val, err := strconv.Atoi(raw)
		if err != nil {
			return nil, &errschemer.SchemerErr{
				Code:    "0025",
				Message: "invalid cherry-picked tag: " + raw,
				Err:     err,
			}
		}

		cherries[val] = true
	}

	result.Cherries = &cherries
	return &result, nil
}

// getAppliedDeltas retrieves applied deltas from the schemer table.
// Queries the database for all applied delta tags and returns them as a map.
//
// Params:
//   - connection: pointer to a pgx.Conn representing the database connection
//   - ctx: context for controlling query timeout or cancellation
//
// Returns:
//   - map[int]bool: a map of applied delta tags where the key is the tag version and value is true
//   - error: non-nil if the schemer table is missing or a query/scan error occurs
func getAppliedDeltas(connection *pgx.Conn, ctx context.Context) (map[int]bool, error) {
	if connection == nil {
		return nil, &errschemer.SchemerErr{
			Code:    "0020",
			Message: "expected pointer to pgx.Conn, recived nil",
			Err:     nil,
		}
	}
	statement := `SELECT tag FROM schemer ORDER BY tag`
	rows, err := connection.Query(ctx, statement)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "42P01" {
				return nil, &errschemer.SchemerErr{
					Code: "0021",
					Message: `failed to find schemer table. Schemer table is used to track migrations and must be present.
Ensure project was setup with [schemer] init`,
					Err: nil,
				}

			}
		}
		return nil, &errschemer.SchemerErr{
			Code:    "0022",
			Message: "failed to query applied versions",
			Err:     err,
		}

	}
	defer rows.Close()

	applied := make(map[int]bool)

	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, &errschemer.SchemerErr{
				Code:    "0023",
				Message: "failed to scan version",
				Err:     nil,
			}

		}
		applied[version] = true
	}

	if err := rows.Err(); err != nil {
		return nil, &errschemer.SchemerErr{
			Code:    "0024",
			Message: "row iteration error:",
			Err:     err,
		}

	}
	return applied, nil
}
