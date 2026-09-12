package interfaces

import "universal-translator/models"

// Downloader fetches a model archive and verifies its checksum.
type Downloader interface {
	// Download fetches the model archive, reporting progress through the
	// callback, and returns the path to the downloaded archive file.
	Download(model models.Model, onProgress func(received int64, total int64)) (string, error)
}
