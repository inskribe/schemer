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

package templates

import (
	_ "embed"
	"html/template"
	"os"
	"path/filepath"
	"regexp"

	"github.com/inskribe/schemer/internal/errschemer"
)

//go:embed assets/schemer.sql
var schemerTemplate string

type SchemerTemplateArgs struct {
	TableName string
}

func (args *SchemerTemplateArgs) WriteTemplate(deltasDirPath string) error {
	if err := args.Parse(); err != nil {
		return &errschemer.SchemerErr{
			Code:    "0016",
			Message: "failed to parse template args.",
			Err:     err,
		}
	}

	schemerFilepath := filepath.Join(deltasDirPath, "schemer.sql")
	file, err := os.Create(schemerFilepath)
	if err != nil {
		return &errschemer.SchemerErr{
			Code:    "0017",
			Message: "failed to create schemer.sql file in the deltas directory.",
			Err:     err,
		}
	}

	templ, err := template.New("schemer").Parse(schemerTemplate)
	if err != nil {
		return &errschemer.SchemerErr{
			Code:    "0018",
			Message: "failed to parse schemer template.",
			Err:     err,
		}
	}

	return templ.Execute(file, args)
}

func (args *SchemerTemplateArgs) Parse() error {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, args.TableName)
	if !matched {
		return &errschemer.SchemerErr{
			Code:    "0019",
			Message: "detected illegal character in table name",
			Err:     nil,
		}
	}
	return nil
}
