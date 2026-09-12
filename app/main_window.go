// Package ui builds the GTK4 user interface for the translator.
package app

import (
	"universal-translator/bindings/gtk"
	"universal-translator/models"
)

// MainWindow is the application's main GTK window.
type MainWindow struct {
	window *gtk.Window
	root   *gtk.Box

	sourceDropdown *gtk.DropDown
	targetDropdown *gtk.DropDown

	inputTextView  *gtk.TextView
	outputTextView *gtk.TextView

	actionButton *gtk.Button
	progressBar  *gtk.ProgressBar
	statusLabel  *gtk.Label

	sourceLanguages []models.Language
	targetLanguages []models.Language

	onLanguageChanged func(models.LanguagePair)
	onAction          func()
}

// NewMainWindow builds the window and its widgets.
func NewMainWindow(app *gtk.Application, sourceLanguages []models.Language, targetLanguages []models.Language) *MainWindow {
	window := gtk.NewWindow(app)
	window.SetTitle("Universal Translator")
	window.SetDefaultSize(960, 640)

	main := &MainWindow{
		window:          window,
		sourceLanguages: sourceLanguages,
		targetLanguages: targetLanguages,
	}

	main.buildInterface()
	window.SetChild(main.root.AsPtr())
	return main
}

func (main *MainWindow) buildInterface() {
	root := gtk.NewBox(gtk.OrientationVertical, 12)
	root.SetMarginStart(16)
	root.SetMarginEnd(16)
	root.SetMarginTop(16)
	root.SetMarginBottom(12)

	selector := gtk.NewBox(gtk.OrientationHorizontal, 12)
	sourceColumn, sourceDropdown := buildLabeledDropdown("From", main.sourceLanguages, main.notifyLanguageChanged)
	targetColumn, targetDropdown := buildLabeledDropdown("To", main.targetLanguages, main.notifyLanguageChanged)
	main.sourceDropdown = sourceDropdown
	main.targetDropdown = targetDropdown
	selector.Append(sourceColumn.AsPtr())
	selector.Append(targetColumn.AsPtr())
	root.Append(selector.AsPtr())

	panes := gtk.NewBox(gtk.OrientationHorizontal, 12)
	panes.SetVExpand(true)

	main.inputTextView = gtk.NewTextView()
	main.inputTextView.SetWrapMode(gtk.WrapWord)
	panes.Append(buildScrolledTextView(main.inputTextView).AsPtr())

	main.outputTextView = gtk.NewTextView()
	main.outputTextView.SetWrapMode(gtk.WrapWord)
	main.outputTextView.SetEditable(false)
	panes.Append(buildScrolledTextView(main.outputTextView).AsPtr())

	root.Append(panes.AsPtr())

	actionRow := gtk.NewBox(gtk.OrientationHorizontal, 12)

	main.actionButton = gtk.NewButton("Translate")
	main.actionButton.OnClick(func() {
		if main.onAction != nil {
			main.onAction()
		}
	})
	actionRow.Append(main.actionButton.AsPtr())

	main.progressBar = gtk.NewProgressBar()
	main.progressBar.SetHExpand(true)
	main.progressBar.SetVisible(false)
	actionRow.Append(main.progressBar.AsPtr())

	root.Append(actionRow.AsPtr())

	main.statusLabel = gtk.NewLabel("")
	main.statusLabel.SetWrap(true)
	main.statusLabel.SetHAlign(gtk.AlignStart)
	root.Append(main.statusLabel.AsPtr())

	main.root = root
}

func buildLabeledDropdown(title string, languages []models.Language, onChanged func()) (*gtk.Box, *gtk.DropDown) {
	column := gtk.NewBox(gtk.OrientationVertical, 4)
	column.SetHExpand(true)

	label := gtk.NewLabel(title)
	label.SetHAlign(gtk.AlignStart)
	column.Append(label.AsPtr())

	names := make([]string, len(languages))
	for i, language := range languages {
		names[i] = language.Name
	}

	dropdown := gtk.NewDropDown(names)
	dropdown.SetHExpand(true)
	dropdown.OnChanged(onChanged)
	column.Append(dropdown.AsPtr())

	return column, dropdown
}

func buildScrolledTextView(textView *gtk.TextView) *gtk.ScrolledWindow {
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyAutomatic)
	scrolled.SetChild(textView.AsPtr())
	scrolled.SetHExpand(true)
	scrolled.SetVExpand(true)
	return scrolled
}

func (main *MainWindow) notifyLanguageChanged() {
	if main.onLanguageChanged != nil {
		main.onLanguageChanged(main.SelectedPair())
	}
}

// Present shows the window.
func (main *MainWindow) Present() {
	main.window.Present()
}

// OnDestroy registers a callback invoked when the window is destroyed.
func (main *MainWindow) OnDestroy(callback func()) {
	main.window.OnDestroy(callback)
}

// SelectedPair returns the currently selected language pair.
func (main *MainWindow) SelectedPair() models.LanguagePair {
	return models.LanguagePair{
		Source: main.selectedLanguage(main.sourceDropdown, main.sourceLanguages),
		Target: main.selectedLanguage(main.targetDropdown, main.targetLanguages),
	}
}

// SelectLanguages selects the given language codes in the dropdowns. Changing
// the selection triggers the language-changed callback.
func (main *MainWindow) SelectLanguages(sourceCode string, targetCode string) {
	main.sourceDropdown.SetSelected(uint(languageIndex(main.sourceLanguages, sourceCode)))
	main.targetDropdown.SetSelected(uint(languageIndex(main.targetLanguages, targetCode)))
}

func languageIndex(languages []models.Language, code string) int {
	for i, language := range languages {
		if language.Code == code {
			return i
		}
	}
	return 0
}

func (main *MainWindow) selectedLanguage(dropdown *gtk.DropDown, languages []models.Language) models.Language {
	if len(languages) == 0 {
		return models.Language{}
	}

	index := dropdown.GetSelected()
	if int(index) >= len(languages) {
		return languages[0]
	}
	return languages[index]
}

// InputText returns the current text of the source text area.
func (main *MainWindow) InputText() string {
	return main.inputTextView.GetText()
}

// SetOutputText replaces the contents of the target text area.
func (main *MainWindow) SetOutputText(text string) {
	main.outputTextView.SetText(text)
}

// SetOnLanguageChanged registers a callback invoked when either dropdown
// changes.
func (main *MainWindow) SetOnLanguageChanged(callback func(models.LanguagePair)) {
	main.onLanguageChanged = callback
}

// SetOnAction registers a callback invoked when the action button (Translate /
// Download Model) is clicked.
func (main *MainWindow) SetOnAction(callback func()) {
	main.onAction = callback
}

// SetButtonLabel changes the action button label.
func (main *MainWindow) SetButtonLabel(label string) {
	main.actionButton.SetLabel(label)
}

// SetButtonSensitive enables or disables the action button.
func (main *MainWindow) SetButtonSensitive(sensitive bool) {
	main.actionButton.SetSensitive(sensitive)
}

// SetBusy shows or hides the progress indicator.
func (main *MainWindow) SetBusy(busy bool) {
	main.progressBar.SetVisible(busy)
}

// SetStatus updates the status label.
func (main *MainWindow) SetStatus(text string) {
	main.statusLabel.SetText(text)
}
