package storage

import (
	"bytes"
	"fmt"
	"image"

	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
)

type ImageProcessor struct {
	MaxSize int64 // bytes (default: 5MB)
}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{MaxSize: 5 * 1024 * 1024} // 5MB
}

// Check JPEG/PNG, throw err nếu file > max size
func (p *ImageProcessor) ValidateImage(data []byte) error {
	if int64(len(data)) > p.MaxSize {
		return fmt.Errorf("image exceeds %dMB", p.MaxSize/(1024*1024))
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("not an image: %w", err)
	}
	switch format {
	case "jpeg", "png":
		return nil
	default:
		return fmt.Errorf("image format %s not allowed (only jpeg/png)", format)
	}
}

// Trả về map[variant][]byte: resize → enc JPEG chất lượng 90
func (p *ImageProcessor) ProcessImage(data []byte) (map[string][]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("cannot decode image: %w", err)
	}
	vs := map[string]int{"large": 1200, "medium": 600, "thumbnail": 300}
	variants := map[string][]byte{}
	for name, size := range vs {
		resized := imaging.Fit(img, size, size, imaging.Lanczos)
		b := new(bytes.Buffer)
		if err := jpeg.Encode(b, resized, &jpeg.Options{Quality: 90}); err != nil {
			return nil, fmt.Errorf("cannot encode %s: %w", name, err)
		}
		variants[name] = b.Bytes()
	}
	return variants, nil
}
