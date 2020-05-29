package util

// MapStringKeys returns a slice of map keys, given
// that this map is string-indexed
func MapStringKeys(m map[string]interface{}) (keys []string) {
	keys = make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}

	return keys
}
