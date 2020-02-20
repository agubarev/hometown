package util

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// WriteResponseErrorTo is a helper function that writes an error to
// a supplied http.ResponseWriter
func WriteResponseErrorTo(w http.ResponseWriter, key string, err error, code int) {
	payload, _err := json.Marshal(HTTPError{
		Key:     key,
		Message: err.Error(),
		Code:    code,
	})

	if _err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal error response: %s", _err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	http.Error(w, string(payload), code)
}
