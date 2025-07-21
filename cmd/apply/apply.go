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

	er "github.com/inskribe/schemer/internal/errschemer"
)

// parseApplyCommand validates and resolves input flags for the apply command.
// Ensures that either --conn-key or --conn-string is provided, and enforces that
// --cherry-pick cannot be used with --from or --to.
//
// Returns:
//   - error: SchemerErr with a specific code if validation fails
var parseApplyCommand = func(request *CommandArgs) error {
	if request.connKey == "" && request.connString == "" {
		return &er.SchemerErr{
			Code:    "0001",
			Message: "--conn-key or --conn-string must be used.",
			Err:     nil,
		}
	}

	if request.connString == "" {
		request.connString = os.Getenv(request.connKey)
		if request.connString == "" {
			return &er.SchemerErr{
				Code:    "0002",
				Message: fmt.Sprintf("failed to get environment variable value for key: %s", request.connKey),
				Err:     nil,
			}
		}
	}

	if len(request.cherryPickedVersions) > 0 && (request.toTag != "" || request.fromTag != "") {
		return &er.SchemerErr{
			Code:    "0003",
			Message: "flags --from/--to cannot be used with --cherry-pick",
			Err:     nil,
		}
	}
	return nil
}
