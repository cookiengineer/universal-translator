// Package repositories provides the concrete, on-disk model store.
package repositories

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"universal-translator/caches"
	"universal-translator/types"
)

// ModelStore manages models on disk. It looks up models in two locations: the
// writable user cache directory and the read-only, system-wide directory that
// distribution packages install into. The user cache takes precedence.
type ModelStore struct {
	userModelsDirectory   string
	systemModelsDirectory string
}

// NewModelStore creates a store backed by the user's model cache directory
// and the system-wide models directory.
func NewModelStore() (*ModelStore, error) {
	directory, err := caches.ModelsDirectory()
	if err != nil {
		return nil, err
	}

	return &ModelStore{
		userModelsDirectory:   directory,
		systemModelsDirectory: caches.SystemModelsPath,
	}, nil
}

// InstalledModels returns the short names of all models present on disk.
func (store *ModelStore) InstalledModels() []string {
	seen := map[string]bool{}
	shortNames := []string{}

	for _, directory := range store.modelDirectories() {
		entries, err := os.ReadDir(directory)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() || seen[entry.Name()] {
				continue
			}
			if hasFile(filepath.Join(directory, entry.Name()), types.ModelConfigFileName) {
				seen[entry.Name()] = true
				shortNames = append(shortNames, entry.Name())
			}
		}
	}

	sort.Strings(shortNames)
	return shortNames
}

// IsInstalled reports whether the given model is available locally, either in
// the user cache or system-wide.
func (store *ModelStore) IsInstalled(shortName string) bool {
	_, ok := store.locateModel(shortName)
	return ok
}

// ModelDirectory returns the directory holding the given model, preferring the
// user cache over the system-wide location. If the model is not present
// anywhere it returns the user-cache path where it would be installed.
func (store *ModelStore) ModelDirectory(shortName string) string {
	directory, _ := store.locateModel(shortName)
	return directory
}

// StagingDirectory returns the temporary directory used to assemble the model
// before it is committed into the user cache.
func (store *ModelStore) StagingDirectory(shortName string) string {
	return filepath.Join(store.userModelsDirectory, shortName+".extracting")
}

// Commit validates a populated staging directory and moves it into place,
// returning the installed model.
func (store *ModelStore) Commit(shortName string, stagingDir string) (types.InstalledModel, error) {
	directory := filepath.Join(store.userModelsDirectory, shortName)

	if !hasFile(stagingDir, types.ModelConfigFileName) {
		os.RemoveAll(stagingDir)
		return types.InstalledModel{}, fmt.Errorf("downloaded model %s is missing %s", shortName, types.ModelConfigFileName)
	}

	if err := os.Rename(stagingDir, directory); err != nil {
		os.RemoveAll(stagingDir)
		return types.InstalledModel{}, err
	}

	return store.installedModel(shortName), nil
}

func (store *ModelStore) installedModel(shortName string) types.InstalledModel {
	directory := store.ModelDirectory(shortName)
	return types.InstalledModel{
		ShortName:  shortName,
		Directory:  directory,
		ConfigPath: filepath.Join(directory, types.ModelConfigFileName),
	}
}

// modelDirectories returns the search roots in priority order: the writable
// user cache first, then the read-only system-wide directory.
func (store *ModelStore) modelDirectories() []string {
	return []string{store.userModelsDirectory, store.systemModelsDirectory}
}

// locateModel returns the directory holding the model and whether it exists.
func (store *ModelStore) locateModel(shortName string) (string, bool) {
	for _, directory := range store.modelDirectories() {
		if hasFile(filepath.Join(directory, shortName), types.ModelConfigFileName) {
			return filepath.Join(directory, shortName), true
		}
	}
	return filepath.Join(store.userModelsDirectory, shortName), false
}

func hasFile(directory string, name string) bool {
	info, err := os.Stat(filepath.Join(directory, name))
	return err == nil && !info.IsDir()
}
