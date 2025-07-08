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

	"github.com/inskribe/schemer/internal/errschemer"
)

//go:embed assets/schemer.env
var envTemplate string

type EnvTemplateArgs struct {
	UrlKey   string
	UrlValue string
}

func (args *EnvTemplateArgs) WriteEnvFile(directoryPath string) error {
	templ, err := template.New("env").Parse(envTemplate)
	if err != nil {
		return &errschemer.SchemerErr{
			Code:    "0013",
			Message: "failed to parse env template.",
			Err:     err,
		}

	}
	file, err := os.Create(filepath.Join(directoryPath, ".env"))
	if err != nil {
		return &errschemer.SchemerErr{
			Code:    "0014",
			Message: "failed to create .env file.",
			Err:     err,
		}
	}
	err = templ.Execute(file, args)
	if err != nil {
		return &errschemer.SchemerErr{
			Code:    "0015",
			Message: "failed to wirte template to .env file.",
			Err:     err,
		}
	}
	return nil
}
