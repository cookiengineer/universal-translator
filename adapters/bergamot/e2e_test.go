//go:build e2e

package bergamot_test

import (
	"testing"

	"universal-translator/adapters/bergamot"
	catalog "universal-translator/models"
	"universal-translator/repositories"
	"universal-translator/services/downloading"
	modelservice "universal-translator/services/models"
)

// TestEndToEndTranslation exercises the full pipeline: download, extract,
// load and translate a tiny model.
func TestEndToEndTranslation(t *testing.T) {
	store, err := repositories.NewModelStore()
	if err != nil {
		t.Fatal(err)
	}

	manager := modelservice.NewModelManager(store, downloading.NewDownloader())

	pair := catalog.LanguagePair{
		Source: catalog.Language{Name: "English", Code: "en"},
		Target: catalog.Language{Name: "German", Code: "de"},
	}

	prepared, err := manager.Prepare(pair, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("model config: %s", prepared.FirstConfigPath)

	engine, err := bergamot.NewEngine(2, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	model, err := engine.LoadModel(prepared.FirstConfigPath)
	if err != nil {
		t.Fatalf("load model: %v", err)
	}
	defer model.Close()

	translated, err := engine.Translate(model, "Hello world. This is a test.")
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	t.Logf("translated: %q", translated)
	if translated == "" {
		t.Fatal("translation is empty")
	}
}

// TestPivotTranslation exercises a two-model pivot through English.
func TestPivotTranslation(t *testing.T) {
	store, err := repositories.NewModelStore()
	if err != nil {
		t.Fatal(err)
	}

	manager := modelservice.NewModelManager(store, downloading.NewDownloader())

	pair := catalog.LanguagePair{
		Source: catalog.Language{Name: "Czech", Code: "cs"},
		Target: catalog.Language{Name: "German", Code: "de"},
	}

	prepared, err := manager.Prepare(pair, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.IsPivot {
		t.Fatalf("expected pivot, got direct")
	}

	engine, err := bergamot.NewEngine(2, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	first, err := engine.LoadModel(prepared.FirstConfigPath)
	if err != nil {
		t.Fatalf("load first: %v", err)
	}
	defer first.Close()

	second, err := engine.LoadModel(prepared.SecondConfigPath)
	if err != nil {
		t.Fatalf("load second: %v", err)
	}
	defer second.Close()

	translated, err := engine.Pivot(first, second, "Hello world.")
	if err != nil {
		t.Fatalf("pivot: %v", err)
	}

	t.Logf("pivoted: %q", translated)
	if translated == "" {
		t.Fatal("translation is empty")
	}
}

// TestFirefoxModelTranslation exercises a Firefox Translations model, which is
// downloaded as individual files (model + vocab + lex) with a generated config.
func TestFirefoxModelTranslation(t *testing.T) {
	store, err := repositories.NewModelStore()
	if err != nil {
		t.Fatal(err)
	}

	manager := modelservice.NewModelManager(store, downloading.NewDownloader())

	pair := catalog.LanguagePair{
		Source: catalog.Language{Name: "English", Code: "en"},
		Target: catalog.Language{Name: "Russian", Code: "ru"},
	}

	prepared, err := manager.Prepare(pair, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("model config: %s", prepared.FirstConfigPath)

	engine, err := bergamot.NewEngine(2, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	model, err := engine.LoadModel(prepared.FirstConfigPath)
	if err != nil {
		t.Fatalf("load model: %v", err)
	}
	defer model.Close()

	translated, err := engine.Translate(model, "Hello world. This is a test.")
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	t.Logf("translated: %q", translated)
	if translated == "" {
		t.Fatal("translation is empty")
	}
}
