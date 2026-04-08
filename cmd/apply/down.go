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
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/inskribe/schemer/cmd"
	"github.com/inskribe/schemer/internal/errschemer"
	"github.com/inskribe/schemer/internal/glog"
	"github.com/inskribe/schemer/internal/utils"
)

var (
	downForce   bool
	downRequest CommandArgs
	downCmd     = &cobra.Command{
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
		Run: func(command *cobra.Command, args []string) {
			if cmd.RootCmd.PersistentPreRun != nil {
				cmd.RootCmd.PersistentPreRun(command, args)
			}

			_, err := utils.LoadDotEnv()
			if err != nil {
				glog.Error("%v", err)
			}

			if err := parseApplyCommand(&downRequest); err != nil {
				glog.Error("%v", err)
				return
			}

			if shouldOnlyApplyLast() {
				err := utils.WithConn(downRequest.connString, applyForLastUpDelta)
				if err != nil {
					glog.Error("%v", err)
					return
				}
				return
			}

			err = utils.WithConn(downRequest.connString, executeDownCommand)
			if err != nil {
				glog.Error("%v", err)
				return
			}
		},
	}
)

func init() {
	cmd.RootCmd.AddCommand(downCmd)
	downCmd.PersistentFlags().StringVarP(&downRequest.connKey, "conn-key", "k", "", "The key to fetch the environment variable value for the database connection string.")
	downCmd.PersistentFlags().BoolVarP(&downRequest.dryRun, "dry-run", "d", false, "Performs a dry run and outputs the actions. No actions will be commited against the database.")
	downCmd.PersistentFlags().StringVarP(&downRequest.connString, "conn-string", "s", "", "The driver specific connection string. If passed the connection key will be ignored.")
	downCmd.PersistentFlags().BoolVar(&downRequest.PruneNoOp, "prune", false, `Enable no-operation file prunning. Scan delta files and skip applying files
that only contains comments and empty lines. This can be useful for large replays to avoid unnessecarry database calls.`)
	downCmd.PersistentFlags().StringVarP(&downRequest.toTag, "to", "t", "", `Specify the version to end at. Accepted formats are: 
  4   - No Padding
  004 - Padded zeros`)

	downCmd.PersistentFlags().StringVarP(&downRequest.fromTag, "from", "f", "", `Specify the version to begin at. Accepted formats are:
  4   - No Padding
  004 - Padded zeros`)
	downCmd.PersistentFlags().StringArrayVarP(&downRequest.cherryPickedVersions, "cherry-pick", "c", nil, `Specify deltas to execute againg the database.
It is possible to cherry pick non-consecutive deltas. This is not reccomended and do so at your own risk.
Accepted formats are:
  4   - No Padding
  004 - Padded zeros
		`)

	downCmd.PersistentFlags().BoolVarP(&downForce, "force", "", false, `Force applying down deltas even if the corresponding up delta is not applied.
This can be useful for rolling back changes that were applied outside of schemer or for recovering from an inconsistent state, 
but use with caution as it can lead to an inconsistent state between the database and schemer's tracking.`)
}

func shouldOnlyApplyLast() bool {
	return len(downRequest.cherryPickedVersions) == 0 && downRequest.fromTag == "" && downRequest.toTag == ""
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
	applied, err := GetAppliedDeltas(connection, ctx)
	if err != nil {
		return err
	}

	glog.Info("Found %d applied deltas", len(applied))

	request, err := downRequest.GetRequestedDeltas()
	if err != nil {
		return err
	}

	statements, err := loadDownDeltas(request)
	if err != nil {
		return err
	}

	if downRequest.PruneNoOp {
		PruneNoOp(&statements)
	}

	var deltasToApply []int
	var skippedDeltas []string

	for tag := range statements {
		_, ok := applied[tag]
		if !ok && !downForce {
			skippedDeltas = append(skippedDeltas, utils.ToPrefix(tag))
			continue
		}
		deltasToApply = append(deltasToApply, tag)
	}

	if len(skippedDeltas) > 0 {
		glog.Warn("The following deltas were skipped because they are not currently applied: %s \n use --force to apply them", strings.Join(skippedDeltas, ", "))
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
func loadDownDeltas(request *DeltaRequest) (map[int][]byte, error) {
	if request == nil {
		return nil, &errschemer.SchemerErr{
			Code:    "load-down-deltas-001",
			Message: "expected valid deltaRequest, recieved nil",
		}
	}

	deltaPath, err := utils.GetDeltaPath()
	if err != nil {
		return nil, err
	}

	expression := regexp.MustCompile(`^(\d+)_.*\.down\.sql$`)
	result := make(map[int][]byte)

	err = filepath.WalkDir(deltaPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return &errschemer.SchemerErr{
				Code:    "load-down-deltas-002",
				Message: "failed to access path: " + path,
				Err:     err,
			}
		}

		if d.IsDir() {
			return nil
		}

		matches := expression.FindStringSubmatch(d.Name())
		if matches == nil || len(matches) < 2 {
			return nil
		}

		tag, err := strconv.Atoi(matches[1])
		if err != nil {
			return &errschemer.SchemerErr{
				Code:    "load-down-deltas-003",
				Message: "malformed filename: " + d.Name(),
				Err:     err,
			}
		}

		if request.LastTag != nil {
			if tag != *request.LastTag {
				return nil
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				return &errschemer.SchemerErr{
					Code:    "load-down-deltas-004",
					Message: "failed to read file at path: " + path,
					Err:     err,
				}
			}

			result[tag] = contents

			return filepath.SkipDir
		}

		if request.Cherries != nil {
			if !(*request.Cherries)[tag] {
				return nil
			}
		} else {
			if request.From != nil && tag > *request.From {
				return nil
			}
			if request.To != nil && tag < *request.To {
				return nil
			}
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return &errschemer.SchemerErr{
				Code:    "load-down-deltas-005",
				Message: "failed to read file at path: " + path,
				Err:     err,
			}
		}

		if _, exists := result[tag]; exists {
			return &errschemer.SchemerErr{
				Code:    "load-down-deltas-006",
				Message: fmt.Sprintf("duplicate down delta tag found: %03d", tag),
			}
		}

		result[tag] = contents
		return nil
	})
	if err != nil {
		return nil, err
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
	appliedDeltas, err := GetAppliedDeltas(connection, ctx)
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

	request := &DeltaRequest{LastTag: &lastTag}

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
