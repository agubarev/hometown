package util

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/tidwall/pretty"
)

func PrettyPrint(marshal bool, val interface{}) {
	var buf []byte
	var err error

	if marshal {
		buf, err = json.Marshal(val)
		if err != nil {
			panic(errors.Wrap(err, "PrettyBuf(): failed to marshal value"))
		}
	}

	fmt.Printf("%s", pretty.Pretty(buf))
}
