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

	// StagingDirectory returns the temporary directory used to assemble a
	// model before it is committed.
	StagingDirectory(shortName string) string

	// Commit validates a populated staging directory and moves it into place,
	// returning the installed model.
	Commit(shortName string, stagingDir string) (types.InstalledModel, error)
}
