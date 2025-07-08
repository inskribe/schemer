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
package create

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/inskribe/schemer/cmd"
	"github.com/inskribe/schemer/internal/errschemer"
	"github.com/inskribe/schemer/internal/glog"
	"github.com/inskribe/schemer/internal/utils"
)

// Represents user input for create command.
type CreateCmdRequest struct {
	Post bool // Identifies if the user requested a post file to be created.
}

var (
	CreateRequest CreateCmdRequest
	createCmd     = &cobra.Command{
		Use:   "create [name] [options]",
		Short: "Create versioned delta files for the given name",
		Long: `The create command generates a new group of delta files with the next available version tag.

By default, it creates a .up.sql and .down.sql file in the deltas directory.
Use the --post flag to include an optional .post.sql file for post-migration cleanup.

Examples:
  schemer create add_users
    → deltas/004_add_users.up.sql
      deltas/004_add_users.down.sql

  schemer create add_services --post
    → deltas/005_add_services.up.sql
      deltas/005_add_services.down.sql
      deltas/005_add_services.post.sql

Delta files follow the format: {version}_{name}.{up,down,post}.sql

This command does not apply the delta or connect to the database.
It simply prepares the file structure for future use.
`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := executeCreateCommand(args); err != nil {
				glog.Error("%v", err)
				return
			}
		},
	}
)

func init() {
	cmd.RootCmd.AddCommand(createCmd)
	createCmd.Flags().BoolVarP(&CreateRequest.Post, "post", "p", false, `Create a post delta along with the up and down deltas.`)
}

// determineNextTag returns the next available delta tag for a new delta file group.
// Scans the given deltaPath for existing *.up.sql files and determines the next tag in sequence.
//
// Params:
//   - deltaPath: the directory containing delta files
//
// Returns:
//   - int: the next tag number (0 if directory is empty)
//   - error: non-nil if the directory can't be read or a tag can't be parsed
func determineNextTag(deltaPath string) (int, error) {
	files, err := os.ReadDir(deltaPath)
	if err != nil {
		return -1, &errschemer.SchemerErr{
			Code:    "0056",
			Message: "failed to read directory at: " + deltaPath,
			Err:     err,
		}
	}

	// exit early if empty.
	if len(files) == 0 {
		return 0, nil
	}

	expression := regexp.MustCompile(`^(\d+)_.*\.*\.sql$`)

	next := -1
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		matches := expression.FindStringSubmatch(name)
		if matches == nil || len(matches) < 2 {
			continue
		}

		tag, err := strconv.Atoi(matches[1])
		if err != nil {
			return -1, &errschemer.SchemerErr{
				Code:    "0055",
				Message: "malformed delta tag: " + matches[0],
				Err:     err,
			}
		}

		if tag > next {
			next = tag
		}
	}

	return next + 1, nil
}

// createDeltaFiles creates a delta file group in the deltas directory.
//
// Naming format:
//
//	tag_filename.{up,down,post}.sql
//
// Example:
//
//	001_remove_user.up.sql
//	001_remove_user.down.sql
//	001_remove_user.post.sql
//
// If an error is returned, it will be of type PathError.
func createDeltaFiles(filename string, nextTag int, deltaPath string) error {

	ErrAlreadExist := "delta file alread exist for: "
	name := strings.Join([]string{utils.ToPrefix(nextTag), strings.Trim(filename, "_")}, "_")

	up := strings.Join([]string{name, "up", "sql"}, ".")
	down := strings.Join([]string{name, "down", "sql"}, ".")

	upPath := filepath.Join(deltaPath, up)
	downPath := filepath.Join(deltaPath, down)

	if _, err := os.Stat(upPath); err == nil {
		return &errschemer.SchemerErr{
			Code:    "0057",
			Message: ErrAlreadExist + upPath,
		}
	}

	if _, err := os.Stat(downPath); err == nil {
		return &errschemer.SchemerErr{
			Code:    "0058",
			Message: ErrAlreadExist + downPath,
		}
	}

	upFile, err := os.Create(upPath)
	if err != nil {
		return &errschemer.SchemerErr{
			Code:    "0059",
			Message: "failed to create up.sql file",
			Err:     err,
		}
	}

	defer upFile.Close()

	downFile, err := os.Create(downPath)
	if err != nil {
		return &errschemer.SchemerErr{
			Code:    "0060",
			Message: "failed to create down.sql file",
			Err:     err,
		}
	}

	defer downFile.Close()

	_, _ = upFile.WriteString("-- TODO: Add delta SQL here\n")
	_, _ = downFile.WriteString("-- TODO: Add rollback SQL here\n")

	// Exit early if user did not request a .post file
	if !CreateRequest.Post {
		glog.Info("Created deltas:\n  %s\n  %s\n", upPath, downPath)
		return nil
	}

	// Handle post file creation
	post := strings.Join([]string{name, "post", "sql"}, ".")
	postPath := filepath.Join(deltaPath, post)

	if _, err := os.Stat(postPath); err == nil {
		return &errschemer.SchemerErr{
			Code:    "0061",
			Message: ErrAlreadExist + postPath,
		}
	}

	postFile, err := os.Create(postPath)
	if err != nil {
		return &errschemer.SchemerErr{
			Code:    "0062",
			Message: "failed to create post.sql file",
			Err:     err,
		}
	}

	defer postFile.Close()

	_, _ = postFile.WriteString("-- TODO: Handle delta cleanup here\n")

	glog.Info("Created deltas:\n  %s\n  %s\n  %s\n", upPath, downPath, postPath)

	return nil
}

func executeCreateCommand(args []string) error {
	deltaPath, err := utils.GetDeltaPath()
	if err != nil {
		return err
	}

	nextTag, err := determineNextTag(deltaPath)
	if err != nil {
		return err
	}

	if err := createDeltaFiles(args[0], nextTag, deltaPath); err != nil {
		return err
	}
	return nil
}
