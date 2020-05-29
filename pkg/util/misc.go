package util

import (
	"fmt"

	"github.com/r3labs/diff"
)

func ProtectedChangelog(allowedFields map[string]bool, before, after interface{}) (diff.Changelog, error) {
	changelog, err := diff.Diff(before, after)
	if err != nil {
		return nil, err
	}

	// going through changes and checking whether every changed field is allowed
	for _, change := range changelog {
		if _, ok := allowedFields[change.Path[0]]; !ok {
			return nil, fmt.Errorf("`%s` is protected and cannot be changed", change.Path[0])
		}
	}

	return changelog, nil
}
