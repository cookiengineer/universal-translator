// Package downloading provides the HTTP model downloader with checksum
// verification and progress reporting.
package downloading

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"

	"universal-translator/models"
)

// Downloader fetches model archives over HTTP.
type Downloader struct {
	httpClient *http.Client
}

// NewDownloader creates a downloader with a default HTTP client.
func NewDownloader() *Downloader {
	return &Downloader{httpClient: &http.Client{}}
}

// Download fetches the model archive, verifying its SHA-256 checksum, and
// returns the path to the downloaded temporary archive file.
func (downloader *Downloader) Download(model models.Model, onProgress func(received int64, total int64)) (string, error) {
	response, err := downloader.httpClient.Get(model.URL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download of %s failed with status %s", model.ShortName, response.Status)
	}

	file, err := os.CreateTemp("", "universal-translator-*.tar.gz")
	if err != nil {
		return "", err
	}

	if err := downloader.streamToFile(response.Body, file, model.Checksum, response.ContentLength, onProgress); err != nil {
		file.Close()
		os.Remove(file.Name())
		return "", err
	}

	if err := file.Close(); err != nil {
		os.Remove(file.Name())
		return "", err
	}

	return file.Name(), nil
}

func (downloader *Downloader) streamToFile(reader io.Reader, file *os.File, expectedChecksum string, total int64, onProgress func(int64, int64)) error {
	hasher := sha256.New()
	buffer := make([]byte, 32*1024)
	var received int64

	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			received += int64(count)
			if _, writeErr := file.Write(buffer[:count]); writeErr != nil {
				return writeErr
			}
			hasher.Write(buffer[:count])
			if onProgress != nil {
				onProgress(received, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}
