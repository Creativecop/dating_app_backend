package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/neoscoder/aura-backend/internal/storage"
)

const (
	MimeJPEG = "image/jpeg"
	MimePNG  = "image/png"
	MimeMP4  = "video/mp4"
	MimeMOV  = "video/quicktime"
	MimeWebM = "video/webm"
)

type StoredUpload struct {
	Temp         *storage.TempFile
	SizeBytes    int64
	Checksum     string
	MimeType     string
	Width        int
	Height       int
	OriginalName string
}

func streamUploadToTemp(ctx context.Context, reader io.Reader, provider storage.Provider, maxBytes int64) (*StoredUpload, error) {
	temp, err := provider.CreateTemp(ctx)
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	var sniff bytes.Buffer
	writer := io.MultiWriter(temp.File, hasher)
	limited := &limitWriter{writer: writer, max: maxBytes}

	buffer := make([]byte, 32*1024)
	for {
		n, readErr := reader.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			if sniff.Len() < 512 {
				remaining := 512 - sniff.Len()
				if len(chunk) < remaining {
					remaining = len(chunk)
				}
				_, _ = sniff.Write(chunk[:remaining])
			}
			if _, err := limited.Write(chunk); err != nil {
				_ = temp.File.Close()
				_ = os.Remove(temp.Path)
				return nil, err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = temp.File.Close()
			_ = os.Remove(temp.Path)
			return nil, readErr
		}
	}
	if err := temp.File.Close(); err != nil {
		_ = os.Remove(temp.Path)
		return nil, err
	}

	return &StoredUpload{
		Temp:      temp,
		SizeBytes: limited.written,
		Checksum:  hex.EncodeToString(hasher.Sum(nil)),
		MimeType:  http.DetectContentType(sniff.Bytes()),
	}, nil
}

type limitWriter struct {
	writer  io.Writer
	max     int64
	written int64
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if w.written+int64(len(p)) > w.max {
		return 0, fmt.Errorf("file exceeds maximum size")
	}
	n, err := w.writer.Write(p)
	w.written += int64(n)
	return n, err
}

func validatePhotoFile(upload *StoredUpload, maxBytes int64) error {
	if upload.SizeBytes <= 0 {
		return validationError("File is required", map[string]any{"field": "file"})
	}
	if upload.SizeBytes > maxBytes {
		return validationError("Photo is too large", map[string]any{"field": "file"})
	}
	if upload.MimeType != MimeJPEG && upload.MimeType != MimePNG {
		return validationError("Only JPEG and PNG photos are supported", map[string]any{"field": "file"})
	}

	file, err := os.Open(upload.Temp.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return validationError("Invalid image file", map[string]any{"field": "file"})
	}
	upload.Width = cfg.Width
	upload.Height = cfg.Height
	if cfg.Width < 400 || cfg.Height < 400 {
		return validationError("Photo must be at least 400x400 pixels", map[string]any{"field": "file"})
	}
	return nil
}

func validateVideoFile(upload *StoredUpload, maxBytes int64) error {
	if upload.SizeBytes <= 0 {
		return validationError("File is required", map[string]any{"field": "file"})
	}
	if upload.SizeBytes > maxBytes {
		return validationError("Video is too large", map[string]any{"field": "file"})
	}
	switch upload.MimeType {
	case MimeMP4, MimeMOV, MimeWebM:
		return nil
	default:
		return validationError("Only MP4, MOV, and WebM videos are supported", map[string]any{"field": "file"})
	}
}

func extensionForMime(mimeType string) string {
	switch mimeType {
	case MimeJPEG:
		return ".jpg"
	case MimePNG:
		return ".png"
	case MimeMP4:
		return ".mp4"
	case MimeMOV:
		return ".mov"
	case MimeWebM:
		return ".webm"
	default:
		return ""
	}
}

func normalizeVariantParam(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "original":
		return VariantOriginal, nil
	case "display":
		return VariantDisplay, nil
	case "thumbnail":
		return VariantThumbnail, nil
	case "transcoded":
		return VariantTranscoded, nil
	default:
		return "", validationError("Media variant is invalid", map[string]any{"variant": value})
	}
}
