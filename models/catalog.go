package models

import (
	_ "embed"
	"encoding/json"
	"sort"
)

// EnglishCode is the pivot language used when no direct model exists.
const EnglishCode = "en"

//go:embed catalog.json
var catalogJSON []byte

type catalogFile struct {
	Models []Model `json:"models"`
}

var catalog []Model

func init() {
	var file catalogFile
	if err := json.Unmarshal(catalogJSON, &file); err != nil {
		panic("could not parse embedded model catalog: " + err.Error())
	}
	catalog = file.Models
}

// All returns every model in the catalog.
func All() []Model {
	return catalog
}

// SourceLanguages returns the distinct languages that can be translated from.
func SourceLanguages() []Language {
	return distinctLanguages(func(model Model) Language {
		return Language{Name: model.SourceName, Code: model.SourceCode}
	})
}

// TargetLanguages returns the distinct languages that can be translated into.
func TargetLanguages() []Language {
	return distinctLanguages(func(model Model) Language {
		return Language{Name: model.TargetName, Code: model.TargetCode}
	})
}

// Resolve determines the model(s) needed for the given language pair, either
// a single direct model or two models pivoting through English.
func Resolve(pair LanguagePair) (Resolution, bool) {
	if direct := findModel(pair.Source.Code, pair.Target.Code); direct != nil {
		return Resolution{Direct: true, First: direct}, true
	}

	first := findModel(pair.Source.Code, EnglishCode)
	second := findModel(EnglishCode, pair.Target.Code)
	if first != nil && second != nil {
		return Resolution{Direct: false, First: first, Second: second}, true
	}

	return Resolution{}, false
}

// findModel returns a model for the given source and target codes, preferring
// a "tiny" variant when both are available.
func findModel(sourceCode string, targetCode string) *Model {
	var found *Model
	for i := range catalog {
		model := &catalog[i]
		if model.SourceCode != sourceCode || model.TargetCode != targetCode {
			continue
		}
		if found == nil || (found.ModelType != "tiny" && model.ModelType == "tiny") {
			found = model
		}
	}
	return found
}

func distinctLanguages(selectLanguage func(Model) Language) []Language {
	seen := map[string]bool{}
	languages := []Language{}

	for _, model := range catalog {
		language := selectLanguage(model)
		if seen[language.Code] {
			continue
		}
		seen[language.Code] = true
		languages = append(languages, language)
	}

	sort.Slice(languages, func(i, j int) bool {
		return languages[i].Name < languages[j].Name
	})

	return languages
}
