package media

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"

	"github.com/neoscoder/aura-backend/internal/storage"
)

func TestNormalizeVariantParamAllowlist(t *testing.T) {
	cases := map[string]string{
		"original":   VariantOriginal,
		"display":    VariantDisplay,
		"thumbnail":  VariantThumbnail,
		"transcoded": VariantTranscoded,
	}
	for raw, expected := range cases {
		actual, err := normalizeVariantParam(raw)
		if err != nil {
			t.Fatalf("expected %s to be valid: %v", raw, err)
		}
		if actual != expected {
			t.Fatalf("expected %s, got %s", expected, actual)
		}
	}

	if _, err := normalizeVariantParam("../original"); err == nil {
		t.Fatal("expected path traversal style variant to fail")
	}
	if _, err := normalizeVariantParam("blurred"); err == nil {
		t.Fatal("expected non-route variant to fail")
	}
}

func TestStreamUploadRejectsOversizeWithoutReadAll(t *testing.T) {
	provider, err := storage.NewLocalProvider(t.TempDir())
	if err != nil {
		t.Fatalf("storage provider: %v", err)
	}

	_, err = streamUploadToTemp(context.Background(), bytes.NewReader([]byte("123456")), provider, 5)
	if err == nil {
		t.Fatal("expected oversize upload to fail")
	}
}

func TestValidatePhotoFileDecodesDimensions(t *testing.T) {
	provider, err := storage.NewLocalProvider(t.TempDir())
	if err != nil {
		t.Fatalf("storage provider: %v", err)
	}

	var body bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	img.Set(0, 0, color.White)
	if err := jpeg.Encode(&body, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}

	upload, err := streamUploadToTemp(context.Background(), bytes.NewReader(body.Bytes()), provider, int64(body.Len()+1))
	if err != nil {
		t.Fatalf("stream upload: %v", err)
	}
	defer cleanupTemp(upload.Temp)

	if err := validatePhotoFile(upload, int64(body.Len()+1)); err != nil {
		t.Fatalf("expected valid photo: %v", err)
	}
	if upload.Width != 400 || upload.Height != 400 {
		t.Fatalf("expected decoded 400x400 dimensions, got %dx%d", upload.Width, upload.Height)
	}

	_ = os.Remove(upload.Temp.Path)
}
