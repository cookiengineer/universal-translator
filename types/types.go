// Package types holds small value types shared across the application layers.
package types

// ModelConfigFileName is the name of the Marian configuration file that every
// model directory must contain.
const ModelConfigFileName = "config.intgemm8bitalpha.yml"

// InstalledModel locates a model that has been downloaded and extracted into
// the local model cache.
type InstalledModel struct {
	ShortName  string
	Directory  string
	ConfigPath string
}
