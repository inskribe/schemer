package errschemer

import (
	"errors"
	"fmt"
	"strings"
)

type SchemerErr struct {
	Code    string
	Message string
	Err     error
}

func (e *SchemerErr) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("Error %s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("Error %s: %s", e.Code, e.Message)
}

func (e *SchemerErr) Unwrap() error {
	return e.Err
}

func FormatChain(err error) string {
	if err == nil {
		return ""
	}

	var b strings.Builder
	i := 0

	for err != nil {
		// Special-case your error type for richer output
		if se, ok := err.(*SchemerErr); ok {
			if se.Err != nil {
				fmt.Fprintf(&b, "%d) SchemerErr code=%s msg=%q inner=%v\n", i, se.Code, se.Message, se.Err)
			} else {
				fmt.Fprintf(&b, "%d) SchemerErr code=%s msg=%q\n", i, se.Code, se.Message)
			}
		} else {
			fmt.Fprintf(&b, "%d) %T: %v\n", i, err, err)
		}

		err = errors.Unwrap(err)
		i++
	}

	return b.String()
}
