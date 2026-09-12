// Package downloading provides the HTTP model downloader with checksum
// verification and progress reporting.
package downloading

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"universal-translator/models"
	"universal-translator/types"
)

// Downloader fetches model files over HTTP.
type Downloader struct {
	httpClient *http.Client
}

// NewDownloader creates a downloader with a default HTTP client.
func NewDownloader() *Downloader {
	return &Downloader{httpClient: &http.Client{}}
}

// Download materialises a model into destinationDir. For archive models it
// downloads and extracts the archive; for multi-file models it downloads each
// file and writes the generated config.
func (downloader *Downloader) Download(model models.Model, destinationDir string, onProgress func(received int64, total int64)) error {
	if model.IsArchive() {
		return downloader.downloadArchive(model, destinationDir, onProgress)
	}
	return downloader.downloadFiles(model, destinationDir, onProgress)
}

func (downloader *Downloader) downloadArchive(model models.Model, destinationDir string, onProgress func(int64, int64)) error {
	file, err := downloader.downloadToTempFile(model.URL, model.Checksum, onProgress)
	if err != nil {
		return err
	}
	defer os.Remove(file)

	return extractArchive(file, destinationDir)
}

func (downloader *Downloader) downloadFiles(model models.Model, destinationDir string, onProgress func(int64, int64)) error {
	for _, artifact := range model.Files {
		targetPath := filepath.Join(destinationDir, artifact.Filename)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		if err := downloader.downloadToPath(artifact.URL, artifact.SHA256, targetPath, onProgress); err != nil {
			return err
		}
	}

	if model.ConfigYAML == "" {
		return nil
	}
	return os.WriteFile(filepath.Join(destinationDir, types.ModelConfigFileName), []byte(model.ConfigYAML), 0o644)
}

func (downloader *Downloader) downloadToTempFile(url string, checksum string, onProgress func(int64, int64)) (string, error) {
	file, err := os.CreateTemp("", "universal-translator-*.tar.gz")
	if err != nil {
		return "", err
	}

	if err := downloader.streamToFile(url, checksum, file, onProgress); err != nil {
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

func (downloader *Downloader) downloadToPath(url string, checksum string, targetPath string, onProgress func(int64, int64)) error {
	file, err := os.Create(targetPath)
	if err != nil {
		return err
	}

	if err := downloader.streamToFile(url, checksum, file, onProgress); err != nil {
		file.Close()
		os.Remove(targetPath)
		return err
	}

	return file.Close()
}

func (downloader *Downloader) streamToFile(url string, checksum string, file *os.File, onProgress func(int64, int64)) error {
	response, err := downloader.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %s", response.Status)
	}

	hasher := sha256.New()
	buffer := make([]byte, 32*1024)
	var received int64
	total := response.ContentLength

	for {
		count, err := response.Body.Read(buffer)
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
	if actualChecksum != checksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", checksum, actualChecksum)
	}

	return nil
}

// extractArchive extracts a .tar.gz archive into destinationDirectory,
// stripping the single leading directory component shared by all entries.
func extractArchive(archivePath string, destinationDirectory string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.New("downloaded file is not a valid gzip archive")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		relativePath := stripLeadingComponent(header.Name)
		if relativePath == "" || strings.Contains(relativePath, "..") {
			continue
		}

		targetPath := filepath.Join(destinationDirectory, filepath.FromSlash(relativePath))
		if !strings.HasPrefix(targetPath, filepath.Clean(destinationDirectory)+string(os.PathSeparator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			output, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(output, tarReader); err != nil {
				output.Close()
				return err
			}
			output.Close()
		}
	}
}

func stripLeadingComponent(path string) string {
	cleaned := strings.TrimPrefix(filepath.ToSlash(path), "./")
	segments := strings.Split(cleaned, "/")
	if len(segments) <= 1 {
		return cleaned
	}
	return strings.Join(segments[1:], "/")
}
