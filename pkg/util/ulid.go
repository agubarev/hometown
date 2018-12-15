package util

import (
	"math/rand"
	"time"

	"github.com/oklog/ulid"
)

var ulidChannel chan ulid.ULID

// initULIDChannel running a goroutine that pushes ulid.ULIDs into ulidChannel
// this goroutine is meant to run for as long as the app is working
func initULIDChannel() {
	if ulidChannel != nil {
		return
	}

	// initializing the channel and running a goroutine
	ulidChannel = make(chan ulid.ULID, 100)
	go func() {
		t := time.Now()
		entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
		for {
			ulidChannel <- ulid.MustNew(ulid.Timestamp(t), entropy)
		}
	}()
}

// NewULID returns a new ulid.ULID
func NewULID() ulid.ULID {
	return <-ulidChannel
}

func init() {
	initULIDChannel()
}
