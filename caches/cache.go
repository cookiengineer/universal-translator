// Package caches resolves the XDG-compliant cache locations used to store
// downloaded translation models.
package caches

import (
	"os"
	"path/filepath"
)

// modelsDirectoryName is the application-specific subdirectory of the user
// cache directory where model archives and extracted models live.
const modelsDirectoryName = "universal-translator"

// SystemModelsPath is the read-only, system-wide location where distribution
// packages (e.g. universal-translator-languages) install shared models.
const SystemModelsPath = "/usr/share/universal-translator/models"

// ModelsDirectory returns the directory where downloaded models are stored
// (e.g. ~/.cache/universal-translator/models), creating it if needed.
func ModelsDirectory() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	directory := filepath.Join(base, modelsDirectoryName, "models")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}

	return directory, nil
}
