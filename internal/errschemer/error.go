package errschemer

import "fmt"

type SchemerErr struct {
	Code    string
	Message string
	Err     error
}

func (err *SchemerErr) Error() string {
	return fmt.Sprintf("Error %s: %s", err.Code, err.Message)
}

func (err *SchemerErr) Unwrap() error {
	return err.Err
}
