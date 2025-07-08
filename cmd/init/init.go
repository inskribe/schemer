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
package init

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/inskribe/schemer/cmd"
	"github.com/inskribe/schemer/internal/errschemer"
	"github.com/inskribe/schemer/internal/glog"
	"github.com/inskribe/schemer/internal/templates"
	"github.com/inskribe/schemer/internal/utils"
)

// initCmd represents the init command
var (
	DatabaseArgs templates.EnvTemplateArgs

	initCmd = &cobra.Command{
		Use:   "init [options]",
		Short: "Initialize a schemer project in the current directory",
		Long: `The init command sets up a new schemer project scaffold in the current working directory.

It creates a deltas directory (if it doesn't already exist) and generates a .env file
for storing environment-specific variables like the database connection string.

Use the --url-key flag to customize the generated environment variable key.

Note: This command is intended for local development or initial setup and should not be
run in production environments, as it may overwrite existing configuration files.
`,
		Run: func(cmd *cobra.Command, args []string) {
			glog.Info("Initializing Schemer project\n\n")

			_, err := utils.LoadDotEnv()
			if err != nil {
				glog.Error(err.Error())
			}

			if err := executeInitCommand(); err != nil {
				glog.Error(err.Error())
			}
		},
	}
)

func init() {
	cmd.RootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&DatabaseArgs.UrlValue, "url", "u", "", "The connection string to your database.")
	initCmd.Flags().StringVarP(&DatabaseArgs.UrlKey, "key", "k", "", "An environment variable key to retrieve connection string.")
}

// executeInitCommand initializes a new schemer project in the current directory.
// Creates the deltas directory, generates a .env file, writes the schemer.sql template,
// and optionally creates the schemer tracking table using the resolved database connection.
//
// Returns:
//   - error: non-nil if any step in the initialization process fails
func executeInitCommand() error {
	cwd, err := os.Getwd()
	if err != nil {
		return &errschemer.SchemerErr{
			Code:    "0067",
			Message: "failed to get working directory.",
			Err:     err,
		}
	}

	if DatabaseArgs.UrlKey == "" {
		DatabaseArgs.UrlKey = "DATABASE_URL"
		glog.Info("No --key provided, defaulting to DATABASE_URL")
	}

	if DatabaseArgs.UrlValue == "" {
		val, ok := os.LookupEnv(DatabaseArgs.UrlKey)
		if ok && val != "" {
			DatabaseArgs.UrlValue = val
			glog.Info("Loaded database URL from env var %s", DatabaseArgs.UrlKey)
		} else {
			DatabaseArgs.UrlValue = "YOUR_DATABASE_URL_VALUE"
			glog.Info(`No --url or environment value found, defaulting to placeholder.
Please update after initialization.`)
		}
	}

	deltaPath := filepath.Join(cwd, "deltas")
	if err = createDeltasDirectory(deltaPath); err != nil {
		return err
	}

	if err := createEnvFile(cwd); err != nil {
		return err
	}

	schemerArgs := templates.SchemerTemplateArgs{
		TableName: "schemer",
	}

	if err := schemerArgs.WriteTemplate(deltaPath); err != nil {
		return err
	}

	if DatabaseArgs.UrlValue == "YOUR_DATABASE_URL_VALUE" {
		glog.Info(`Skipping schemer table creation.
Please update .env file or pass a valid [--url] or [--key]
Once updated you can run init again to create table or Schemer will create
the table when you run the first apply up command.`)
		return nil
	}

	if err = utils.WithConn(DatabaseArgs.UrlValue, utils.CreateSchemerTable); err != nil {
		return err
	}

	return nil
}

// createDeltasDirectory ensures the deltas directory exists.
// If the directory already exists, it logs a message and exits cleanly.
// If the path is empty, returns a PathError with ErrInvalidPath.
//
// Params:
//   - deltaPath: path to the directory where deltas should be stored
//
// Returns:
//   - error:  if the path is invalid, or other filesystem-related errors
func createDeltasDirectory(deltaPath string) error {
	if deltaPath == "" {
		return &errschemer.SchemerErr{
			Code:    "0066",
			Message: "empty directory path",
		}
	}

	err := os.Mkdir(deltaPath, 0775)
	if errors.Is(err, fs.ErrExist) {
		glog.Info("Deltas directory detected. Using existing directory.")
		return nil
	}

	if err != nil && !errors.Is(err, fs.ErrExist) {
		return &errschemer.SchemerErr{
			Code:    "0063",
			Message: "failed to create deltas directory.",
			Err:     err,
		}
	}

	glog.Info("Created deltas directory")

	return nil
}

// createEnvFile ensures a valid .env file exists in the working directory.
// If a .env file already exists, it logs a reminder about the DATABASE_URL format.
// If not found, it creates one using values from DatabaseArgs.
// Returns a PathError if the path is invalid.
//
// Params:
//   - workingDir: directory where the .env file should be created
//
// Returns:
//   - error: for invalid path, or I/O errors during file check or creation
func createEnvFile(workingDir string) error {
	if workingDir == "" {
		return &errschemer.SchemerErr{
			Code:    "0064",
			Message: "empty working directory.",
			Err:     nil,
		}
	}

	info, err := os.Stat(filepath.Join(workingDir, ".env"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	} else if err == nil && info.IsDir() {
		return &errschemer.SchemerErr{
			Code:    "0065",
			Message: "Schemer does not support .env directory",
			Err:     nil,
		}
	} else if err == nil {
		glog.Info(`Detected a .env file. Ensure key:value exist for database connection string.`)
		return nil
	}

	glog.Info("Env file not found. Creating .env file in current directory")

	return DatabaseArgs.WriteEnvFile(workingDir)
}
