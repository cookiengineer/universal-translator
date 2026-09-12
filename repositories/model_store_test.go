package repositories

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFakeModel(t *testing.T, base string, shortName string) string {
	t.Helper()
	directory := filepath.Join(base, shortName)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, ModelConfigFileName), []byte("models: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return directory
}

func TestSystemWideModelResolution(t *testing.T) {
	store := &ModelStore{
		userModelsDirectory:   t.TempDir(),
		systemModelsDirectory: t.TempDir(),
	}

	modelDir := writeFakeModel(t, store.systemModelsDirectory, "en-de-tiny")

	if !store.IsInstalled("en-de-tiny") {
		t.Fatal("expected system-wide model to be installed")
	}
	if got := store.ModelDirectory("en-de-tiny"); got != modelDir {
		t.Fatalf("ModelDirectory = %q, want %q", got, modelDir)
	}
	if store.IsInstalled("missing-model") {
		t.Fatal("missing model should not be reported as installed")
	}
}

func TestUserCacheTakesPrecedence(t *testing.T) {
	store := &ModelStore{
		userModelsDirectory:   t.TempDir(),
		systemModelsDirectory: t.TempDir(),
	}

	writeFakeModel(t, store.systemModelsDirectory, "en-fr-tiny")
	userModelDir := writeFakeModel(t, store.userModelsDirectory, "en-fr-tiny")

	if got := store.ModelDirectory("en-fr-tiny"); got != userModelDir {
		t.Fatalf("ModelDirectory = %q, want user cache %q", got, userModelDir)
	}
}
