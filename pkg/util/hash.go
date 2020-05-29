package util

import (
	"hash/crc32"

	"github.com/cespare/xxhash"
	"github.com/pkg/errors"
)

type work struct {
	kind    string
	payload []byte
	result  chan interface{}
}

var workline = make(chan work, 100)

// TODO: resolve deadlock
func init() {
	go func() {
		xx := xxhash.New()
		crc32hash := crc32.NewIEEE()

		for {
			select {
			case w := <-workline:
				switch w.kind {
				case "xxhash":
					if _, err := xx.Write(w.payload); err != nil {
						close(w.result)
						panic(errors.Wrap(err, "failed to write data to xxhash"))
					}

					w.result <- xx.Sum64()
					xx.Reset()
				case "crc32":
					if _, err := crc32hash.Write(w.payload); err != nil {
						close(w.result)
						panic(errors.Wrap(err, "failed to write data to crc32"))
					}

					w.result <- crc32hash.Sum32()
					crc32hash.Reset()
				default:
					panic(errors.Errorf("unhandled hash kind: %s", w.kind))
				}

				return
			}
		}
	}()
}

// HashKey produces a `xxhash` hash from a given byte slice
// NOTE: https://github.com/cespare/xxhash for more details
func HashKey(payload []byte) (result uint64) {
	resultChan := make(chan interface{}, 0)

	workline <- work{
		kind:    "xxhash",
		payload: payload,
		result:  resultChan,
	}

	result = (<-resultChan).(uint64)
	close(resultChan)

	return result
}

// HashCRC32 produces a CRC32 checksum from a given payload
func HashCRC32(payload []byte) (result int64) {
	resultChan := make(chan interface{}, 0)

	workline <- work{
		kind:    "crc32",
		payload: payload,
		result:  resultChan,
	}

	result = (<-resultChan).(int64)
	close(resultChan)

	return result
}
