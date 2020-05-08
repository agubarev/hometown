package task

import "github.com/gocraft/dbr/v2"

type Fn func() (ch chan Status, err error)

type Task struct {
	Name      string
	StartedAt dbr.NullTime
	ExecTime  dbr.NullTime

	fn Fn
}
