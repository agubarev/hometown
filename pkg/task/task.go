package task

import "github.com/gocraft/dbr/v2"

// Process represents a process wrapper function
type Process func() (ch chan Status, err error)

type Task struct {
	Name      string
	StartedAt dbr.NullTime
	ExecTime  dbr.NullTime

	fn Process
}
