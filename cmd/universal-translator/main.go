// Command translator is the entry point for the universal translator
// application.
package main

import (
	"fmt"
	"os"
	"runtime"
	"sync/atomic"

	"universal-translator/bindings/gtk"
	catalog "universal-translator/models"
	"universal-translator/repositories"
	"universal-translator/services/downloading"
	modelservice "universal-translator/services/models"
	"universal-translator/services/translator"
	_app "universal-translator/app"
)

func main() {
	store, err := repositories.NewModelStore()
	if err != nil {
		fatal(err)
	}

	downloader := downloading.NewDownloader()
	modelManager := modelservice.NewModelManager(store, downloader)

	app := gtk.NewApplication("com.universaltranslator.app")

	app.OnActivate(func() {
		runApplication(app, modelManager)
	})

	os.Exit(app.Run())
}

// runApplication builds the window and wires the services together. GTK4
// requires application windows to be created after the application has been
// activated, hence the window is constructed here rather than in main.
func runApplication(app *gtk.Application, modelManager *modelservice.ModelManager) {
	window := _app.NewMainWindow(app, catalog.SourceLanguages(), catalog.TargetLanguages())

	// Tracks the most recent language selection so stale downloads that finish
	// after a newer selection can be discarded.
	var selectionGeneration atomic.Int64

	// UI state, only mutated on the main thread.
	busy := false
	available := true  // whether any model exists for the selected pair
	installed := false // whether the selected pair's model is on disk

	refreshButton := func() {
		switch {
		case !available:
			window.SetButtonLabel("No model")
			window.SetButtonSensitive(false)
		case !installed:
			window.SetButtonLabel("Download Model")
			window.SetButtonSensitive(!busy)
		default:
			window.SetButtonLabel("Translate")
			window.SetButtonSensitive(!busy)
		}
	}

	setBusy := func(value bool) {
		busy = value
		window.SetBusy(value)
		refreshButton()
	}

	translationService, err := translator.NewService(
		workerCount(),
		0,
		func(text string) {
			gtk.RunOnMain(func() { window.SetOutputText(text) })
		},
		func(err error) {
			gtk.RunOnMain(func() { window.SetStatus("Error: " + err.Error()) })
		},
		func(value bool) {
			gtk.RunOnMain(func() { setBusy(value) })
		},
		func() {
			gtk.RunOnMain(func() {
				if installed {
					window.SetStatus("")
				}
			})
		},
	)
	if err != nil {
		window.SetStatus("Error: " + err.Error())
		return
	}

	// prepareAndLoad ensures the model(s) for the pair are downloaded and
	// installed, then loads them into the engine. It runs the blocking work on
	// a background goroutine and discards its result if the selection changed.
	prepareAndLoad := func(pair catalog.LanguagePair, generation int64) {
		setBusy(true)
		window.SetStatus("Preparing model…")

		go func() {
			prepared, err := modelManager.Prepare(pair, func(received int64, total int64) {
				gtk.RunOnMain(func() { window.SetStatus(downloadStatus(received, total)) })
			})

			if generation != selectionGeneration.Load() {
				return
			}

			if err != nil {
				gtk.RunOnMain(func() {
					setBusy(false)
					window.SetStatus("Error: " + err.Error())
				})
				return
			}

			gtk.RunOnMain(func() {
				installed = true
				refreshButton()
				window.SetStatus("Loading model…")
			})

			translationService.Load(prepared.FirstConfigPath, prepared.SecondConfigPath)
		}()
	}

	window.SetOnLanguageChanged(func(pair catalog.LanguagePair) {
		generation := selectionGeneration.Add(1)
		window.SetOutputText("")
		setBusy(false)

		if !modelManager.IsAvailable(pair) {
			available = false
			installed = false
			refreshButton()
			window.SetStatus("No translation model available for this pair.")
			return
		}
		available = true

		if modelManager.IsInstalled(pair) {
			installed = true
			refreshButton()
			prepareAndLoad(pair, generation)
		} else {
			installed = false
			refreshButton()
			window.SetStatus("Model not downloaded yet — click \"Download Model\".")
		}
	})

	window.SetOnAction(func() {
		pair := window.SelectedPair()

		if !available {
			return
		}

		if !installed {
			prepareAndLoad(pair, selectionGeneration.Add(1))
			return
		}

		text := window.InputText()
		if text == "" {
			return
		}
		translationService.Translate(text)
	})

	// Load a sensible default pair (English -> German) so the app is usable
	// immediately.
	window.SelectLanguages("en", "de")
	window.OnDestroy(func() { app.Quit() })
	window.Present()
}

func workerCount() int {
	if count := runtime.NumCPU(); count < 4 {
		return count
	}
	return 4
}

func downloadStatus(received int64, total int64) string {
	if total > 0 {
		return fmt.Sprintf("Downloading model… %.1f / %.1f MB", megabytes(received), megabytes(total))
	}
	return fmt.Sprintf("Downloading model… %.1f MB", megabytes(received))
}

func megabytes(bytes int64) float64 {
	return float64(bytes) / (1024 * 1024)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "universal-translator:", err)
	os.Exit(1)
}
