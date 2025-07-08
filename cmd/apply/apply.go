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
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/inskribe/schemer/cmd"
	er "github.com/inskribe/schemer/internal/errschemer"
	"github.com/inskribe/schemer/internal/glog"
	"github.com/inskribe/schemer/internal/utils"
)

var (
	applyRequest applyCommandArgs

	applyCmd = &cobra.Command{
		Use:   "apply [sub-command] [options]",
		Short: "Run a migration command (up, down, or post)",
		Long: `Apply executes the specified migration direction against your database.

Supported subcommands:
  up     - applies unapplied deltas
  down   - rolls back previously applied deltas
  post   - executes post-migration cleanup steps

Each subcommand supports range-based or cherry-picked delta selection using --from, --to, or --cherry-pick.
Use --dry-run to preview changes without applying them.
`,
		PersistentPreRunE: func(command *cobra.Command, args []string) error {
			if cmd.RootCmd.PersistentPreRun != nil {
				cmd.RootCmd.PersistentPreRun(command, args)
			}

			_, err := utils.LoadDotEnv()
			if err != nil {
				glog.Error("%v", err)
				return err
			}

			if err := parseApplyCommand(); err != nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	cmd.RootCmd.AddCommand(applyCmd)
	applyCmd.PersistentFlags().StringVarP(&applyRequest.connKey, "conn-key", "k", "", "The key to fetch the environment variable value for the database connection string.")
	applyCmd.PersistentFlags().BoolVarP(&applyRequest.dryRun, "dry-run", "d", false, "Performs a dry run and outputs the actions. No actions will be commited against the database.")
	applyCmd.PersistentFlags().StringVarP(&applyRequest.connString, "conn-string", "s", "", "The driver specific connection string. If passed the connection key will be ignored.")
	applyCmd.PersistentFlags().BoolVar(&applyRequest.PruneNoOp, "prune", false, `Enable no-operation file prunning. Scan delta files and skip applying files
that only contains comments and empty lines. This can be useful for large replays to avoid unnessecarry database calls.`)
	applyCmd.PersistentFlags().StringVarP(&applyRequest.toTag, "to", "t", "", `Specify the version to end at. Accepted formats are: 
  4   - No Padding
  004 - Padded zeros`)

	applyCmd.PersistentFlags().StringVarP(&applyRequest.fromTag, "from", "f", "", `Specify the version to begin at. Accepted formats are:
  4   - No Padding
  004 - Padded zeros`)
	applyCmd.PersistentFlags().StringArrayVarP(&applyRequest.cherryPickedVersions, "cherry-pick", "c", nil, `Specify deltas to execute againg the database.
It is possible to cherry pick non-consecutive deltas. This is not reccomended and do so at your own risk.
Accepted formats are:
  4   - No Padding
  004 - Padded zeros
		`)

}

// parseApplyCommand validates and resolves input flags for the apply command.
// Ensures that either --conn-key or --conn-string is provided, and enforces that
// --cherry-pick cannot be used with --from or --to.
//
// Returns:
//   - error: SchemerErr with a specific code if validation fails
func parseApplyCommand() error {
	if applyRequest.connKey == "" && applyRequest.connString == "" {
		return &er.SchemerErr{
			Code:    "0001",
			Message: "--conn-key or --conn-string must be used.",
			Err:     nil,
		}
	}

	if applyRequest.connString == "" {
		applyRequest.connString = os.Getenv(applyRequest.connKey)
		if applyRequest.connString == "" {
			return &er.SchemerErr{
				Code:    "0002",
				Message: fmt.Sprintf("failed to get environment variable value for key: %s", applyRequest.connKey),
				Err:     nil,
			}
		}
	}

	if len(applyRequest.cherryPickedVersions) > 0 && (applyRequest.toTag != "" || applyRequest.fromTag != "") {
		return &er.SchemerErr{
			Code:    "0003",
			Message: "flags --from/--to cannot be used with --cherry-pick",
			Err:     nil,
		}
	}
	return nil
}
