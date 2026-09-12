// Package types holds small value types shared across the application layers.
package types

// InstalledModel locates a model that has been downloaded and extracted into
// the local model cache.
type InstalledModel struct {
	ShortName  string
	Directory  string
	ConfigPath string
}
