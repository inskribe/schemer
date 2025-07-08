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
	"os"
	"path/filepath"

	"github.com/inskribe/schemer/internal/errschemer"
)

// GetDeltaPath returns the absolute path to the deltas directory.
// Assumes the deltas directory is located in the current working directory.
//
// Returns:
//   - string: full path to the deltas directory
//   - error: non-nil if the current working directory cannot be resolved
var GetDeltaPath = func() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", &errschemer.SchemerErr{
			Code:    "0011",
			Message: "failed to get current working.",
			Err:     err,
		}
	}
	return filepath.Join(cwd, "deltas"), nil
}
