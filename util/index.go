package util

import (
	"fmt"
	"path/filepath"

	"github.com/blevesearch/bleve"
)

// InitBleveIndex opens a bleve index, creates and opens if it doesn't exist
func InitBleveIndex(indexDir string) (bleve.Index, error) {
	var index bleve.Index

	// bleve requires a directory instead of just a file
	indexPath := filepath.Join(indexDir)

	// creating new index; will return an error if the index has already been created
	index, err := bleve.New(indexPath, bleve.NewIndexMapping())
	if err != nil {
		if err == bleve.ErrorIndexPathExists {
			// index already exists, trying to load instead
			index, err = bleve.Open(indexPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open existing index: %s", err)
			}
		} else {
			// unhandled error
			return nil, fmt.Errorf("failed to create new index: %s", err)
		}
	}

	return index, nil
}
