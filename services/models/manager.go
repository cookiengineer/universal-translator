// Package models implements the application service that resolves a language
// pair to model(s) and ensures they are downloaded and installed.
package models

import (
	"fmt"
	"path/filepath"

	"universal-translator/interfaces"
	catalog "universal-translator/models"
	"universal-translator/repositories"
	"universal-translator/services/downloading"
	"universal-translator/types"
)

// ModelManager resolves language pairs to models and ensures the required
// models are downloaded and installed before they are loaded by a translator.
type ModelManager struct {
	store      interfaces.ModelStore
	downloader interfaces.Downloader
}

// NewModelManager creates a model manager backed by the local store and a
// concrete downloader.
func NewModelManager(store *repositories.ModelStore, downloader *downloading.Downloader) *ModelManager {
	return &ModelManager{store: store, downloader: downloader}
}

// PreparedModels holds the config file paths of the models that serve a pair.
type PreparedModels struct {
	FirstConfigPath  string
	SecondConfigPath string
	IsPivot          bool
}

// IsAvailable reports whether any model (direct or pivot) exists for the pair.
func (manager *ModelManager) IsAvailable(pair catalog.LanguagePair) bool {
	_, ok := catalog.Resolve(pair)
	return ok
}

// IsInstalled reports whether every model required for the pair is already
// present in the local cache, without triggering any download.
func (manager *ModelManager) IsInstalled(pair catalog.LanguagePair) bool {
	resolution, ok := catalog.Resolve(pair)
	if !ok {
		return false
	}

	if !manager.store.IsInstalled(resolution.First.ShortName) {
		return false
	}
	if resolution.Second != nil && !manager.store.IsInstalled(resolution.Second.ShortName) {
		return false
	}

	return true
}

// Prepare ensures the model(s) for the language pair are downloaded and
// installed, returning the config paths to load.
func (manager *ModelManager) Prepare(pair catalog.LanguagePair, onProgress func(received int64, total int64)) (PreparedModels, error) {
	resolution, ok := catalog.Resolve(pair)
	if !ok {
		return PreparedModels{}, fmt.Errorf("no translation model available for %s to %s", pair.Source.Name, pair.Target.Name)
	}

	first, err := manager.ensureInstalled(resolution.First, onProgress)
	if err != nil {
		return PreparedModels{}, err
	}

	prepared := PreparedModels{
		FirstConfigPath: first.ConfigPath,
		IsPivot:         !resolution.Direct,
	}

	if resolution.Second != nil {
		second, err := manager.ensureInstalled(resolution.Second, onProgress)
		if err != nil {
			return PreparedModels{}, err
		}
		prepared.SecondConfigPath = second.ConfigPath
	}

	return prepared, nil
}

func (manager *ModelManager) ensureInstalled(model *catalog.Model, onProgress func(int64, int64)) (types.InstalledModel, error) {
	if manager.store.IsInstalled(model.ShortName) {
		return manager.installedModel(model.ShortName), nil
	}

	archivePath, err := manager.downloader.Download(*model, onProgress)
	if err != nil {
		return types.InstalledModel{}, err
	}

	return manager.store.Install(model.ShortName, archivePath)
}

func (manager *ModelManager) installedModel(shortName string) types.InstalledModel {
	directory := manager.store.ModelDirectory(shortName)
	return types.InstalledModel{
		ShortName:  shortName,
		Directory:  directory,
		ConfigPath: filepath.Join(directory, repositories.ModelConfigFileName),
	}
}
