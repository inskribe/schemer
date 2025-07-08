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

package utils

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"github.com/inskribe/schemer/internal/errschemer"
	"github.com/inskribe/schemer/internal/glog"
)

// LoadDotEnv attempts to load environment variables from a .env file in the current directory.
// Logs a message if the file is missing (which is expected in production).
//
// Returns:
//   - bool: true if the .env file was successfully loaded; false if it was not found
//   - error: non-nil if an error occurred while attempting to read the file (other than not found)
func LoadDotEnv() (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	envFilepath := filepath.Join(cwd, ".env")
	err = godotenv.Load(envFilepath)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		glog.Info(`Failed to load .env file. This is expexted behaviour in a production environment. 
If you are in a devlopment environment, Schemer requies a .env file in the root directory of the project.
			`)
		return false, nil
	} else if err != nil {
		return false, &errschemer.SchemerErr{
			Code:    "0012",
			Message: "failed to load execute godotenv.Load",
			Err:     err,
		}
	}
	glog.Info("Dev environment detected, loaded .env variables.")
	return true, nil
}
