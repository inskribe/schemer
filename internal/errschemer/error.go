package errschemer

import "fmt"

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
