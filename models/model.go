// Package models holds the embedded catalog of downloadable translation
// models and the queries used to resolve a language pair into one or two
// models (direct translation or a pivot through English).
package models

// Model describes a single downloadable translation model and where to fetch
// it from. The catalog is embedded at build time so no network request is
// needed to discover available models.
//
// A model is downloaded in one of two forms:
//   - an archive (URL/Checksum) containing the model files and its config;
//   - a set of individual files (Files) plus a generated config (ConfigYAML).
type Model struct {
	ShortName  string      `json:"shortName"`
	Name       string      `json:"name"`
	SourceName string      `json:"sourceName"`
	SourceCode string      `json:"sourceCode"`
	TargetName string      `json:"targetName"`
	TargetCode string      `json:"targetCode"`
	ModelType  string      `json:"type"`
	URL        string      `json:"url,omitempty"`
	Checksum   string      `json:"checksum,omitempty"`
	Files      []ModelFile `json:"files,omitempty"`
	ConfigYAML string      `json:"config,omitempty"`
}

// ModelFile is a single downloadable artifact of a multi-file model.
type ModelFile struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
}

// IsArchive reports whether the model is distributed as a single archive.
func (model Model) IsArchive() bool {
	return model.URL != ""
}

// Language is a display name plus its ISO code.
type Language struct {
	Name string
	Code string
}

// LanguagePair is the source and target of a translation request.
type LanguagePair struct {
	Source Language
	Target Language
}

// Resolution describes which model(s) serve a language pair. For a direct
// translation only First is set. For a pivot translation First translates to
// English and Second translates from English to the target.
type Resolution struct {
	Direct bool
	First  *Model
	Second *Model
}
