package imageutil

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"github.com/chai2010/webp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type ProcessOptions struct {
	MaxWidth    int // e.g. 1024
	MaxHeight   int // e.g. 1024
	Quality     int // 1-100 for WebP (default 80)
	MaxFileSize int // target maximum byte size (default 50 * 1024 = 50KB)
}

const (
	MaxFileSizeBytes = 50 * 1024 // 50KB target limit
)

var DefaultOptions = ProcessOptions{
	MaxWidth:    1024,
	MaxHeight:   1024,
	Quality:     80,
	MaxFileSize: MaxFileSizeBytes,
}

type ProcessedImage struct {
	Data        []byte
	Width       int
	Height      int
	Extension   string
	ContentType string
}

// ProcessImage reads an image stream, resizes it if it exceeds MaxWidth/MaxHeight while preserving aspect ratio,
// ProcessImage reads an image stream, resizes it if it exceeds MaxWidth/MaxHeight while preserving aspect ratio,
// and compresses/converts it to .webp format to reduce file size (ensuring size <= MaxFileSize, e.g. 50KB).
// For SVG files, it ensures it does not exceed MaxFileSize.
func ProcessImage(r io.Reader, originalExt string, opts ...ProcessOptions) (*ProcessedImage, error) {
	opt := DefaultOptions
	if len(opts) > 0 {
		if opts[0].MaxWidth > 0 {
			opt.MaxWidth = opts[0].MaxWidth
		}
		if opts[0].MaxHeight > 0 {
			opt.MaxHeight = opts[0].MaxHeight
		}
		if opts[0].Quality > 0 {
			opt.Quality = opts[0].Quality
		}
		if opts[0].MaxFileSize > 0 {
			opt.MaxFileSize = opts[0].MaxFileSize
		}
	}

	ext := strings.ToLower(originalExt)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// SVG doesn't need raster resizing or WebP conversion
	if ext == ".svg" {
		raw, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		if opt.MaxFileSize > 0 && len(raw) > opt.MaxFileSize {
			return nil, fmt.Errorf("ขนาดไฟล์ SVG เกินกำหนด (สูงสุด %d KB)", opt.MaxFileSize/1024)
		}
		return &ProcessedImage{
			Data:        raw,
			Extension:   ".svg",
			ContentType: "image/svg+xml",
		}, nil
	}

	rawBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read image data failed: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(rawBytes))
	if err != nil {
		// Fallback: if decode fails, return original bytes if within size limit
		if opt.MaxFileSize > 0 && len(rawBytes) > opt.MaxFileSize {
			return nil, fmt.Errorf("ขนาดไฟล์รูปภาพเกินกำหนด (สูงสุด %d KB)", opt.MaxFileSize/1024)
		}
		return &ProcessedImage{
			Data:        rawBytes,
			Extension:   ext,
			ContentType: mimeByExt(ext),
		}, nil
	}

	origBounds := img.Bounds()
	origW := origBounds.Dx()
	origH := origBounds.Dy()

	targetW, targetH := calculateDimensions(origW, origH, opt.MaxWidth, opt.MaxHeight)

	var finalImg image.Image
	if targetW != origW || targetH != origH {
		dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, origBounds, draw.Over, nil)
		finalImg = dst
	} else {
		finalImg = img
	}

	var buf bytes.Buffer
	currQuality := opt.Quality
	err = webp.Encode(&buf, finalImg, &webp.Options{
		Quality:  float32(currQuality),
		Lossless: false,
	})
	if err != nil {
		return nil, fmt.Errorf("encode webp failed: %w", err)
	}

	currImg := finalImg
	currW := targetW
	currH := targetH

	// If MaxFileSize is configured and encoded size exceeds it, progressively optimize
	if opt.MaxFileSize > 0 && buf.Len() > opt.MaxFileSize {
		// 1. Lower quality in increments down to 20
		for buf.Len() > opt.MaxFileSize && currQuality > 20 {
			currQuality -= 15
			if currQuality < 20 {
				currQuality = 20
			}
			buf.Reset()
			if err := webp.Encode(&buf, currImg, &webp.Options{
				Quality:  float32(currQuality),
				Lossless: false,
			}); err != nil {
				return nil, fmt.Errorf("encode webp failed: %w", err)
			}
		}

		// 2. If still exceeding MaxFileSize, downscale dimensions
		for buf.Len() > opt.MaxFileSize && (currW > 100 || currH > 100) {
			currW = int(float64(currW) * 0.8)
			currH = int(float64(currH) * 0.8)
			if currW < 64 {
				currW = 64
			}
			if currH < 64 {
				currH = 64
			}
			dst := image.NewRGBA(image.Rect(0, 0, currW, currH))
			draw.CatmullRom.Scale(dst, dst.Bounds(), currImg, currImg.Bounds(), draw.Over, nil)
			currImg = dst

			buf.Reset()
			if err := webp.Encode(&buf, currImg, &webp.Options{
				Quality:  float32(currQuality),
				Lossless: false,
			}); err != nil {
				return nil, fmt.Errorf("encode webp failed: %w", err)
			}

			if currW == 64 && currH == 64 {
				break
			}
		}
	}

	finalBytes := buf.Bytes()
	if opt.MaxFileSize > 0 && len(finalBytes) > opt.MaxFileSize {
		return nil, fmt.Errorf("ไม่สามารถบีบอัดรูปภาพให้มีขนาดต่ำกว่า %d KB ได้", opt.MaxFileSize/1024)
	}

	return &ProcessedImage{
		Data:        finalBytes,
		Width:       currW,
		Height:      currH,
		Extension:   ".webp",
		ContentType: "image/webp",
	}, nil
}

func calculateDimensions(origW, origH, maxW, maxH int) (int, int) {
	if origW <= maxW && origH <= maxH {
		return origW, origH
	}

	ratioW := float64(maxW) / float64(origW)
	ratioH := float64(maxH) / float64(origH)

	ratio := ratioW
	if ratioH < ratioW {
		ratio = ratioH
	}

	newW := int(float64(origW) * ratio)
	newH := int(float64(origH) * ratio)

	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	return newW, newH
}

func mimeByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}
