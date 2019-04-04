package util

import (
	"fmt"

	"github.com/hokaccha/go-prettyjson"
)

// PrettyPrintJSON prints pretty JSON representation of a given object
func PrettyPrintJSON(obj interface{}) {
	s, _ := prettyjson.Marshal(obj)
	fmt.Println(string(s))
}
