// Package repositories provides the concrete, on-disk model store.
package repositories

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"universal-translator/caches"
	"universal-translator/types"
)

// ModelConfigFileName is the name of the Marian configuration file shipped
// inside every model directory.
const ModelConfigFileName = "config.intgemm8bitalpha.yml"

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
			if hasFile(filepath.Join(directory, entry.Name()), ModelConfigFileName) {
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

// Install extracts a downloaded archive into the user model cache and returns
// the installed model.
func (store *ModelStore) Install(shortName string, archivePath string) (types.InstalledModel, error) {
	if store.IsInstalled(shortName) {
		return store.installedModel(shortName), nil
	}

	directory := filepath.Join(store.userModelsDirectory, shortName)

	stagingDirectory := directory + ".extracting"
	if err := os.RemoveAll(stagingDirectory); err != nil {
		return types.InstalledModel{}, err
	}

	if err := os.MkdirAll(stagingDirectory, 0o755); err != nil {
		return types.InstalledModel{}, err
	}

	if err := extractArchive(archivePath, stagingDirectory); err != nil {
		os.RemoveAll(stagingDirectory)
		return types.InstalledModel{}, err
	}

	if !hasFile(stagingDirectory, ModelConfigFileName) {
		os.RemoveAll(stagingDirectory)
		return types.InstalledModel{}, fmt.Errorf("downloaded archive for %s is missing %s", shortName, ModelConfigFileName)
	}

	if err := os.Rename(stagingDirectory, directory); err != nil {
		os.RemoveAll(stagingDirectory)
		return types.InstalledModel{}, err
	}

	return store.installedModel(shortName), nil
}

func (store *ModelStore) installedModel(shortName string) types.InstalledModel {
	directory := store.ModelDirectory(shortName)
	return types.InstalledModel{
		ShortName:  shortName,
		Directory:  directory,
		ConfigPath: filepath.Join(directory, ModelConfigFileName),
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
		if hasFile(filepath.Join(directory, shortName), ModelConfigFileName) {
			return filepath.Join(directory, shortName), true
		}
	}
	return filepath.Join(store.userModelsDirectory, shortName), false
}

func hasFile(directory string, name string) bool {
	info, err := os.Stat(filepath.Join(directory, name))
	return err == nil && !info.IsDir()
}

// extractArchive extracts a .tar.gz archive into destinationDirectory,
// stripping the single leading directory component shared by all entries.
func extractArchive(archivePath string, destinationDirectory string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.New("downloaded file is not a valid gzip archive")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		relativePath := stripLeadingComponent(header.Name)
		if relativePath == "" || strings.Contains(relativePath, "..") {
			continue
		}

		targetPath := filepath.Join(destinationDirectory, filepath.FromSlash(relativePath))
		if !strings.HasPrefix(targetPath, filepath.Clean(destinationDirectory)+string(os.PathSeparator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			output, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(output, tarReader); err != nil {
				output.Close()
				return err
			}
			output.Close()
		}
	}
}

func stripLeadingComponent(path string) string {
	cleaned := strings.TrimPrefix(filepath.ToSlash(path), "./")
	segments := strings.Split(cleaned, "/")
	if len(segments) <= 1 {
		return cleaned
	}
	return strings.Join(segments[1:], "/")
}
