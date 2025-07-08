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
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/inskribe/schemer/internal/errschemer"
	"github.com/inskribe/schemer/internal/glog"
	"github.com/inskribe/schemer/internal/utils"
)

var downCmd = &cobra.Command{
	Use:   "down [options]",
	Short: "Roll back previously applied deltas",
	Long: `The down command rolls back applied deltas in reverse order.

By default, it rolls back only the most recently applied delta.
You can specify a rollback range using --from and --to flags, or use --cherry-pick to target specific deltas.
Use --prune-no-op to skip deltas that contain no executable SQL.

Examples:
  schemer down                        # Roll back the most recent delta
  schemer down --from 005             # Roll back from 005 down to 000
  schemer down --from 005 --to 003    # Roll back from 005 down to 003
  schemer down --cherry-pick 001,004  # Roll back only 001 and 004
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			err := utils.WithConn(applyRequest.connString, applyForLastUpDelta)
			if err != nil {
				glog.Error("%v", err)
				return
			}
			return
		}

		err := utils.WithConn(applyRequest.connString, executeDownCommand)
		if err != nil {
			glog.Error("%v", err)
			return
		}
	},
}

func init() {
	applyCmd.AddCommand(downCmd)
}

// executeDownCommand runs the full "down" migration flow.
// Retrieves applied deltas, loads requested down deltas, optionally prunes no-ops,
// executes them in reverse order, and updates the schemer table accordingly.
//
// Params:
//   - connection: pointer to a pgx.Conn for executing SQL statements
//   - ctx: context for query execution and cancellation
//
// Returns:
//   - error: non-nil if any delta fails to apply or if the schemer table update fails
func executeDownCommand(connection *pgx.Conn, ctx context.Context) error {
	applied, err := getAppliedDeltas(connection, ctx)
	if err != nil {
		return err
	}

	glog.Info("Found %d applied deltas", len(applied))

	request, err := applyRequest.getRequestedDeltas()
	if err != nil {
		return err
	}

	statements, err := loadDownDeltas(request)
	if err != nil {
		return err
	}

	if applyRequest.PruneNoOp {
		PruneNoOp(&statements)
	}

	var deltasToApply []int

	for tag := range statements {
		_, ok := applied[tag]
		if !ok {
			glog.Warn("Skipping delta %s: present in migration directory but not currently applied.", utils.ToPrefix(tag))
			continue
		}
		deltasToApply = append(deltasToApply, tag)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(deltasToApply)))

	var executedDeltas []any
	var placeholders []string
	var executeErr error = nil
	for index, tag := range deltasToApply {
		statement := string(statements[tag])

		_, err = connection.Exec(ctx, statement)
		if err != nil {
			// It is possible that previous deltas were executed.
			// Record error and break to allow for updating schemer table
			// with applied deltas.
			executeErr = &errschemer.SchemerErr{
				Code:    "0028",
				Message: "failed to apply delta: " + utils.ToPrefix(tag),
				Err:     err,
			}
			break
		}
		glog.Info("Successfully applied down delta %s", utils.ToPrefix(tag))
		executedDeltas = append(executedDeltas, tag)
		placeholders = append(placeholders, fmt.Sprintf("$%d", index+1))
	}
	if len(executedDeltas) > 0 {
		schemerStatement := fmt.Sprintf(`DELETE FROM schemer WHERE (tag) IN (%s)`, strings.Join(placeholders, ", "))
		_, err = connection.Exec(ctx, schemerStatement, executedDeltas...)

		// If there is a migration error, wrap both errors.
		if err != nil {
			tableErr := &errschemer.SchemerErr{
				Code:    "0029",
				Message: "failed to update schemer table",
				Err:     err,
			}
			if executeErr != nil {
				return fmt.Errorf("table error: %v\nexecution error: %w", tableErr, executeErr)
			}
			return tableErr
		}
		glog.Info("Successfully updated schemer table")
	}

	return executeErr
}

// loadDownDeltas loads all eligible down deltas from the delta directory.
// Filters and parses .down.sql files based on the provided DeltaRequest range or cherry-picked tags.
//
// Returns:
//   - map[int][]byte: a map of tag numbers to raw SQL data for down deltas
//   - error: non-nil if delta path resolution, file parsing, or tag extraction fails
func loadDownDeltas(request *deltaRequest) (map[int][]byte, error) {

	if request == nil {
		return nil, &errschemer.SchemerErr{
			Code:    "0030",
			Message: "expected valid deltaRequest, recieved nil",
		}
	}

	deltaPath, err := utils.GetDeltaPath()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(deltaPath)
	if err != nil {
		return nil, &errschemer.SchemerErr{
			Code:    "0031",
			Message: "failed to read directory from path: " + deltaPath,
			Err:     err,
		}
	}

	expression := regexp.MustCompile(`^(\d+)_.*\.down\.sql$`)
	result := make(map[int][]byte)

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
				Code:    "0032",
				Message: "malformed filename: " + matches[0],
				Err:     err,
			}
		}

		if request.LastTag != nil {
			if tag != *request.LastTag {
				continue
			}

			path := filepath.Join(deltaPath, entry.Name())
			contents, err := os.ReadFile(path)
			if err != nil {
				return nil, &errschemer.SchemerErr{
					Code:    "0033",
					Message: "failed to read file at path: " + path,
					Err:     err,
				}
			}

			result[tag] = contents
			return result, nil
		}

		if request.Cherries != nil {
			if !(*request.Cherries)[tag] {
				continue
			}
		} else {
			if request.From != nil && tag > *request.From {
				continue
			}
			if request.To != nil && tag < *request.To {
				continue
			}
		}

		path := filepath.Join(deltaPath, entry.Name())
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, &errschemer.SchemerErr{
				Code:    "0034",
				Message: "failed to read file at path: " + path,
				Err:     err,
			}
		}

		result[tag] = contents
	}

	return result, nil
}

// applyForLastUpDelta rolls back only the most recently applied delta.
// Loads and executes the corresponding down delta and updates the schemer table.
//
// Params:
//   - connection: pointer to a pgx.Conn used to query and execute statements
//   - ctx: context for query execution
//
// Returns:
//   - error: non-nil if no deltas are applied, the down delta is missing, or execution fails
func applyForLastUpDelta(connection *pgx.Conn, ctx context.Context) error {
	appliedDeltas, err := getAppliedDeltas(connection, ctx)
	if err != nil {
		return err
	}

	if len(appliedDeltas) == 0 {
		return &errschemer.SchemerErr{
			Code:    "0035",
			Message: "There are no applied deltas in the schemer table, Aborting apply last delta.",
		}
	}

	var lastTag int = -1
	for tag := range appliedDeltas {
		if tag > lastTag {
			lastTag = tag
		}
	}

	request := &deltaRequest{LastTag: &lastTag}

	deltaFile, err := loadDownDeltas(request)
	if err != nil {
		return err
	}

	data, ok := deltaFile[lastTag]
	if !ok {
		return &errschemer.SchemerErr{
			Code:    "0036",
			Message: "failed to find down delta for last applied up delta: " + utils.ToPrefix(lastTag),
		}
	}

	var executeErr error = nil
	statement := string(data)

	_, err = connection.Exec(ctx, statement)
	if err != nil {
		executeErr = &errschemer.SchemerErr{
			Code:    "0037",
			Message: "failed to apply delta: " + utils.ToPrefix(lastTag),
			Err:     err,
		}
	}

	glog.Info("Successfully applied down delta %s", utils.ToPrefix(lastTag))

	schemerStatement := `DELETE FROM schemer WHERE (tag) IN ($1)`
	_, err = connection.Exec(ctx, schemerStatement, lastTag)

	// If there is a execution error, wrap both errors.
	if err != nil {
		tableErr := &errschemer.SchemerErr{
			Code:    "0038",
			Message: "failed to update schemer table.",
			Err:     err,
		}

		if executeErr != nil {
			return fmt.Errorf("table error: %v\nexecution error: %w", executeErr, err)
		}

		return tableErr
	}
	glog.Info("Successfully updated schemer table")

	return executeErr
}
