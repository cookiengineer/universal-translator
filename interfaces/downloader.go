package interfaces

import "universal-translator/models"

// Downloader fetches a model's files onto disk.
type Downloader interface {
	// Download materialises a model into destinationDir, so that
	// destinationDir/config.intgemm8bitalpha.yml exists when it returns. For
	// archive models it downloads and extracts the archive; for multi-file
	// models it downloads each file and writes the generated config. Progress
	// is reported through the callback.
	Download(model models.Model, destinationDir string, onProgress func(received int64, total int64)) error
}
