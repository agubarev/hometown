package util

// HTTPError represents a common error wrapper to be used
// as an HTTP error response
type HTTPError struct {
	Scope   string `json:"scope"`
	Key     string `json:"key"`
	Message string `json:"msg"`
	Code    int    `json:"code"`
}
