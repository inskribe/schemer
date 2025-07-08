package enums

//go:generate stringer -type=LogCategory

type LogCategory int

const (
	Info LogCategory = iota
	Debug
	Warn
	Error
	Fatal
)
