package vision

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"
)

func createSamplePNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 2), G: uint8(y * 2), B: 128, A: 255})
		}
	}
	buf := &bytes.Buffer{}
	_ = png.Encode(buf, img)
	return buf.Bytes()
}

func TestONNXEngine_ExtractEmbedding(t *testing.T) {
	modelPath := "../../../models/clip_vision_model.onnx"
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		modelPath = "models/clip_vision_model.onnx"
	}
	sharedLib := FindDefaultSharedLibraryPath()

	engine, err := NewONNXEngine(modelPath, sharedLib)
	if err != nil {
		t.Fatalf("failed to init ONNXEngine: %v", err)
	}
	defer engine.Close()

	imgBytes := createSamplePNG()
	vec, model, err := engine.ExtractEmbedding(context.Background(), imgBytes)
	if err != nil {
		t.Fatalf("failed to extract embedding: %v", err)
	}

	if model != "Xenova/clip-vit-base-patch32" {
		t.Errorf("unexpected model name: %s", model)
	}

	if len(vec) != 512 {
		t.Fatalf("expected 512 dimensions, got %d", len(vec))
	}

	// Verify L2 normalized: magnitude should be ~1.0
	var sumSq float64
	for _, val := range vec {
		sumSq += float64(val) * float64(val)
	}
	mag := math.Sqrt(sumSq)
	if math.Abs(mag-1.0) > 0.001 {
		t.Errorf("expected unit norm (1.0), got %f", mag)
	}
}
