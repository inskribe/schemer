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

package glog

import (
	"fmt"

	"github.com/inskribe/schemer/internal/glog/enums"
)

const (
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	White  = "\033[97m"
	Reset  = "\033[0m"
)

func debugLogger(category enums.LogCategory, msg string, args ...interface{}) {
	var color string
	switch category {
	case enums.Info:
		color = White
	case enums.Warn:
		color = Yellow
	case enums.Error:
		color = Red
	case enums.Fatal:
		color = Red
	}
	formattedMessage := fmt.Sprintf(msg, args...)
	outputString := fmt.Sprintf("%s[DEBUG][%s]: %s%s\n", color, category, formattedMessage, Reset)
	if category == enums.Fatal {
		panic(outputString)
	}

	fmt.Print(outputString)
}
