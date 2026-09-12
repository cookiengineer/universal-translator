package interfaces

import "universal-translator/types"

// ModelStore manages the local, on-disk collection of downloaded models.
type ModelStore interface {
	// InstalledModels returns the short names of every model on disk.
	InstalledModels() []string

	// IsInstalled reports whether the given model is available locally.
	IsInstalled(shortName string) bool

	// ModelDirectory returns the directory of an installed model.
	ModelDirectory(shortName string) string

	// Install extracts a downloaded archive into the model cache and returns
	// the installed model.
	Install(shortName string, archivePath string) (types.InstalledModel, error)
}
