package service

import (
	"encoding/json"
	"fmt"
	"os"

	"streamingplatform/internal/domain"
)

type CatalogLoader struct{}

// LoadFiles keeps processing after a bad file and returns one result per input path.
func (CatalogLoader) LoadFiles(paths []string) map[string]error {
	results := make(map[string]error, len(paths))
	for _, path := range paths {
		results[path] = loadCatalogFile(path)
	}
	return results
}

func loadCatalogFile(path string) (err error) {
	// Go note: a named return lets the deferred recovery replace a panic with an error.
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("load catalog %s: recovered panic: %v", path, recovered)
		}
	}()

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open catalog %s: %w", path, err)
	}
	// This helper has one-file scope, so defer closes each file on every loop iteration.
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close catalog %s: %w", path, closeErr)
		}
	}()

	var episodes []domain.Episode
	if err := json.NewDecoder(file).Decode(&episodes); err != nil {
		return fmt.Errorf("decode catalog %s: %w", path, err)
	}
	return nil
}
