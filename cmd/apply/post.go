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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/inskribe/schemer/internal/errschemer"
	"github.com/inskribe/schemer/internal/glog"
	"github.com/inskribe/schemer/internal/utils"
)

var (
	postRequest postCmdRequest

	postCmd = &cobra.Command{
		Use:   "post [options]",
		Short: "Apply post-migration deltas",
		Long: `The post command applies post-migration cleanup Sql associated with previously applied deltas.

By default, all pending post deltas will be applied in tag order. Post deltas are optional .post.sql files
used for cleanup or secondary operations following an up migration.

You can limit which deltas are applied using --from, --to, or --cherry-pick.
Use --force to apply post deltas that were added after the corresponding up delta was applied.

Examples:
  schemer post                          # Apply all pending post deltas
  schemer post --from 002 --to 005      # Apply post deltas in the given range
  schemer post --cherry-pick 003,006    # Apply only selected post deltas
  schemer post --cherry-pick 004 --force
`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := utils.WithConn(applyRequest.connString, executePostCommand); err != nil {
				glog.Error("%v", err)
				return
			}
		},
	}
)

func init() {
	applyCmd.AddCommand(postCmd)
	postCmd.Flags().BoolVar(&postRequest.Force, "force", false, `Force will apply a post delta even if the corresponding up delta tag is not marked with
a post delta. This is a convinece flag to recover from a unintended state. 
Post file should be created with schemer create [name] --post which will attach the 
post delta to the up delta. When using force schemer will attach the post to the corresponding up manualy. `)
}

// fetchPostStatuses retrieves all deltas with a post status from the schemer table.
// Filters for entries where post_status > 0.
//
// Params:
//   - conn: pointer to a pgx.Conn for executing the query
//   - ctx: context for controlling query execution
//
// Returns:
//   - map[int]PostStatusEnum: mapping of delta tags to their post status values
//   - error: non-nil if the query, scan, or row iteration fails
func fetchPostStatuses(conn *pgx.Conn, ctx context.Context) (map[int]postStatusEnum, error) {
	statement := `SELECT tag, post_status FROM schemer WHERE post_status > 0;`
	rows, err := conn.Query(ctx, statement)
	if err != nil {
		return nil, &errschemer.SchemerErr{
			Code:    "0039",
			Message: "failed to execute select query for schemer table.",
			Err:     err,
		}
	}

	defer rows.Close()

	result := make(map[int]postStatusEnum)
	for rows.Next() {
		var tag int
		var postStatus postStatusEnum
		if err := rows.Scan(&tag, &postStatus); err != nil {
			return nil, &errschemer.SchemerErr{
				Code:    "0041",
				Message: "failed to scan row",
				Err:     err,
			}
		}
		result[tag] = postStatus
	}

	if err := rows.Err(); err != nil {
		return nil, &errschemer.SchemerErr{
			Code:    "0042",
			Message: "iteration failure on pgx.Rows",
			Err:     err,
		}
	}

	return result, nil
}

// loadPostDeltas loads eligible post deltas from the delta directory.
// Filters based on post status in the schemer table and the current DeltaRequest.
// Skips post deltas that are already applied or not linked to a known up delta (unless --force is used).
//
// Params:
//   - conn: pointer to a pgx.Conn used to fetch post statuses
//   - ctx: context for database queries and file operations
//
// Returns:
//   - map[int]PostDelta: a map of tag numbers to their corresponding PostDelta
//   - error: non-nil if fetching statuses, reading deltas, or scanning tags fails
func loadPostDeltas(request *deltaRequest, conn *pgx.Conn, ctx context.Context) (map[int]postDelta, error) {
	deltaPath, err := utils.GetDeltaPath()
	if err != nil {
		return nil, err
	}

	avaliabePost, err := fetchPostStatuses(conn, ctx)
	if err != nil {
		return nil, err
	}

	pendingDeltas := 0
	appliedDeltas := 0
	for _, status := range avaliabePost {
		if status == Applied {
			appliedDeltas++
		} else if status == Pending {
			pendingDeltas++
		}
	}

	glog.Info("\n  Found %d pending post deltas\n  Found %d applied post deltas", pendingDeltas, appliedDeltas)

	entries, err := os.ReadDir(deltaPath)
	if err != nil {
		return nil, &errschemer.SchemerErr{
			Code:    "0043",
			Message: "failed to read directory at: " + deltaPath,
			Err:     err,
		}
	}
	if len(entries) == 0 {
		return nil, &errschemer.SchemerErr{
			Code:    "0044",
			Message: "deltas directory is empty.",
			Err:     nil,
		}

	}

	expression := regexp.MustCompile(`^(\d+)_.*\.post\.sql$`)
	result := make(map[int]postDelta)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := expression.FindStringSubmatch(entry.Name())
		if matches == nil || len(matches) < 2 {
			continue
		}

		tag, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, &errschemer.SchemerErr{
				Code:    "0045",
				Message: "malformed delta tag: " + matches[0],
				Err:     err,
			}
		}

		if request.Cherries != nil {
			if !(*request.Cherries)[tag] {
				continue
			}
		} else {
			if request.From != nil && tag < *request.From {
				continue
			}
			if request.To != nil && tag > *request.To {
				continue
			}
		}

		path := filepath.Join(deltaPath, entry.Name())
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, &errschemer.SchemerErr{
				Code:    "0046",
				Message: "failed to read delta file at: " + path,
				Err:     err,
			}
		}

		val, exist := avaliabePost[tag]
		if !exist && !postRequest.Force {
			glog.Warn(`Skipping post delta %s: up delta %s has no knowledge of post.
If the post delta was added after up delta was created you can apply the post delta 
with --force to recover from current state.`, utils.ToPrefix(tag), utils.ToPrefix(tag))
			continue
		} else if val == Applied {
			glog.Warn("Skipping delta %s: post has already been applied.", utils.ToPrefix(tag))
			continue
		}

		postDelta := postDelta{
			Tag:        tag,
			Data:       contents,
			PostStatus: Applied,
		}

		result[tag] = postDelta
	}
	return result, nil
}

// executePostCommand runs all eligible post deltas.
// Executes each post delta, marks it as applied in the schemer table, and stops on first failure.
//
// Params:
//   - conn: pointer to a pgx.Conn for executing SQL statements
//   - ctx: context for controlling database operations
//
// Returns:
//   - error: non-nil if any post delta fails to apply or if schemer table update fails
func executePostCommand(conn *pgx.Conn, ctx context.Context) error {
	request, err := applyRequest.getRequestedDeltas()
	if err != nil {
		return err
	}
	deltas, err := loadPostDeltas(request, conn, ctx)
	if err != nil {
		return err
	}

	var execErr error
	var placeholders []string
	var args []any
	i := 0
	for _, delta := range deltas {
		_, err = conn.Exec(ctx, string(delta.Data))
		if err != nil {
			execErr = &errschemer.SchemerErr{
				Code:    "0047",
				Message: "failed to apply post delta: " + utils.ToPrefix(delta.Tag),
				Err:     err,
			}
			break
		}
		glog.Info("Successfully applied post delta %s", utils.ToPrefix(delta.Tag))

		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, delta.Tag)
		i++
	}

	if len(placeholders) == 0 {
		return execErr
	}

	schemerStatment := fmt.Sprintf(`UPDATE schemer SET post_status = 2 WHERE tag IN (%s)`, strings.Join(placeholders, ","))
	_, err = conn.Exec(ctx, schemerStatment, args...)
	if err != nil {
		tableErr := &errschemer.SchemerErr{
			Code:    "0048",
			Message: "failed to update schemer table.",
			Err:     err,
		}
		if execErr != nil {
			return fmt.Errorf("table error: %v\nexecution error: %w", err, execErr)
		}
		return tableErr
	}

	return nil
}
