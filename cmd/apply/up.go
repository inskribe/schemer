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

	"github.com/inskribe/schemer/cmd"
	"github.com/inskribe/schemer/internal/errschemer"
	"github.com/inskribe/schemer/internal/glog"
	"github.com/inskribe/schemer/internal/utils"
)

var (
	upRequest CommandArgs
	upCmd     = &cobra.Command{
		Use:   "up [options]",
		Short: "Apply all unapplied delta files in version order",
		Long: `The up command applies unapplied deltas in sequential order.

By default, it applies all deltas that have not yet been executed, starting from the last applied version.
You can limit the applied range using --from and/or --to flags, or use --cherry-pick to apply specific versions.
Use --prune-no-op to skip deltas with no executable SQL.

Examples:
  schemer up
  schemer up --from 003 --to 006
  schemer up --cherry-pick 004,007
  schemer up --prune-no-op
`,
		Run: func(command *cobra.Command, args []string) {
			if cmd.RootCmd.PersistentPreRun != nil {
				cmd.RootCmd.PersistentPreRun(command, args)
			}

			_, err := utils.LoadDotEnv()
			if err != nil {
				glog.Error("%v", err)
				return
			}

			if err := parseApplyCommand(&upRequest); err != nil {
				glog.Error("%v", err)
				return
			}

			if err := utils.WithConn(upRequest.connString, executeUpCommand); err != nil {
				glog.Error("%v", err)
				return
			}
		},
	}
)

func init() {
	cmd.RootCmd.AddCommand(upCmd)
	upCmd.PersistentFlags().StringVarP(&upRequest.connKey, "conn-key", "k", "", "The key to fetch the environment variable value for the database connection string.")
	upCmd.PersistentFlags().BoolVarP(&upRequest.dryRun, "dry-run", "d", false, "Performs a dry run and outputs the actions. No actions will be commited against the database.")
	upCmd.PersistentFlags().StringVarP(&upRequest.connString, "conn-string", "s", "", "The driver specific connection string. If passed the connection key will be ignored.")
	upCmd.PersistentFlags().BoolVar(&upRequest.PruneNoOp, "prune", false, `Enable no-operation file prunning. Scan delta files and skip applying files
that only contains comments and empty lines. This can be useful for large replays to avoid unnessecarry database calls.`)
	upCmd.PersistentFlags().StringVarP(&upRequest.toTag, "to", "t", "", `Specify the version to end at. Accepted formats are: 
  4   - No Padding
  004 - Padded zeros`)

	upCmd.PersistentFlags().StringVarP(&upRequest.fromTag, "from", "f", "", `Specify the version to begin at. Accepted formats are:
  4   - No Padding
  004 - Padded zeros`)
	upCmd.PersistentFlags().StringArrayVarP(&upRequest.cherryPickedVersions, "cherry-pick", "c", nil, `Specify deltas to execute againg the database.
It is possible to cherry pick non-consecutive deltas. This is not reccomended and do so at your own risk.
Accepted formats are:
  4   - No Padding
  004 - Padded zeros
		`)

}

// loadUpDeltas loads all eligible up deltas from the delta directory.
// Filters and parses .up.sql files based on the provided DeltaRequest range or cherry-picked tags.
//
// Params:
//   - request: pointer to DeltaRequest specifying tag filters (from, to, or cherries)
//
// Returns:
//   - map[int]UpDelta: a map of tag numbers to corresponding UpDelta structs
//   - error: non-nil if delta path can't be resolved, files can't be read, or tag parsing fails
func loadUpDeltas(request *DeltaRequest) (map[int]UpDelta, error) {
	deltaPath, err := utils.GetDeltaPath()
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(deltaPath)
	if err != nil {
		return nil, &errschemer.SchemerErr{
			Code:    "0049",
			Message: "failed to read directory at: " + deltaPath,
			Err:     err,
		}
	}

	expression := regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)
	result := make(map[int]UpDelta)

	for _, entry := range files {
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
				Code:    "0050",
				Message: "malformed delta tag" + matches[0],
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
				Code:    "0051",
				Message: "failed to read file at: " + path,
				Err:     err,
			}

		}

		var status PostStatusEnum = NoExist
		var TagWithName string
		before, after, found := strings.Cut(entry.Name(), ".")
		if found && after == "up.sql" {
			TagWithName = before
			if _, err := os.Stat(filepath.Join(deltaPath, strings.Join([]string{TagWithName, "post", "sql"}, "."))); err == nil {
				status = Pending
			}
		}
		delta := UpDelta{
			Tag:        tag,
			Data:       contents,
			PostStatus: status,
		}

		result[tag] = delta
	}
	return result, nil
}

// executeUpCommand runs the full "up" migration flow.
// Retrieves applied deltas, parses user input, loads new up deltas, and applies them in order.
//
// Params:
//   - connection: pointer to a pgx.Conn for querying and executing SQL
//   - ctx: context for database operations
//
// Returns:
//   - error: non-nil if any step in the up migration process fails
func executeUpCommand(connection *pgx.Conn, ctx context.Context) error {
	applied, err := GetAppliedDeltas(connection, ctx)
	if err != nil {
		return err
	}

	glog.Info("Found %d applied deltas", len(applied))

	deltas, err := upRequest.GetRequestedDeltas()
	if err != nil {
		return err
	}

	statements, err := loadUpDeltas(deltas)
	if err != nil {
		return err
	}

	return applyUpDeltas(applied, statements, connection, ctx)
}

// applyUpDeltas applies unapplied up deltas to the database.
// Executes each delta in order, skipping any already recorded in the schemer table.
// After successful execution, records each applied delta and its post status.
//
// Params:
//   - appliedDeltas: map of already applied delta tags
//   - deltas: map of tag numbers to their corresponding UpDelta
//   - connection: pointer to a pgx.Conn for executing SQL statements
//   - ctx: context for controlling query execution
//
// Returns:
//   - error: non-nil if any delta fails to apply or if the schemer table update fails
func applyUpDeltas(appliedDeltas map[int]bool, deltas map[int]UpDelta, connection *pgx.Conn, ctx context.Context) error {
	if upRequest.PruneNoOp {
		PruneNoOpUp(&deltas)
	}

	var tagsToApply []int
	for tag := range deltas {
		_, ok := appliedDeltas[tag]
		if ok {
			glog.Warn("Skipping delta %s: has already been applied", utils.ToPrefix(tag))
			continue
		}
		tagsToApply = append(tagsToApply, tag)
	}

	if len(tagsToApply) == 0 {
		return &errschemer.SchemerErr{
			Code:    "0052",
			Message: "all requested deltas have been already applied.",
			Err:     nil,
		}

	}

	/*
	* It is possible that the table was not created during initialization.
	* If it exist this is redundant and wasteful.
	 */
	// TODO: cache does exist.
	if err := utils.CreateSchemerTable(connection, ctx); err != nil {
		return err
	}

	sort.Ints(tagsToApply)

	var args []any
	var placeholders []string
	var execErr error = nil

	for i, tag := range tagsToApply {
		statement := string(deltas[tag].Data)

		_, err := connection.Exec(ctx, statement)
		if err != nil {
			execErr = &errschemer.SchemerErr{
				Code:    "0053",
				Message: "failed to apply delta: " + utils.ToPrefix(tag),
				Err:     err,
			}
			break
		}

		glog.Info("Applied delta %s successfully", utils.ToPrefix(tag))
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, tag, deltas[tag].PostStatus)
	}

	if len(placeholders) == 0 {
		return execErr
	}

	schemerStatement := fmt.Sprintf("INSERT INTO schemer (tag, post_status) VALUES %s", strings.Join(placeholders, ", "))

	_, err := connection.Exec(ctx, schemerStatement, args...)
	if err != nil {
		tableErr := &errschemer.SchemerErr{
			Code:    "0054",
			Message: "failed to update schemer table.",
			Err:     err,
		}
		if execErr != nil {
			return fmt.Errorf("table error:  %v\nexecuteion error: %w", err, execErr)
		}
		return tableErr
	}

	return execErr
}
