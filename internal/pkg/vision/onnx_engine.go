package vision

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"math"
	"os"
	"sync"

	"ice-pos-api-v2/internal/domain"

	_ "golang.org/x/image/webp"

	ort "github.com/yalue/onnxruntime_go"
	"golang.org/x/image/draw"
)

var (
	defaultCLIPMean = [3]float32{0.48145466, 0.4578275, 0.40821073}
	defaultCLIPStd  = [3]float32{0.26862954, 0.26130258, 0.27577711}

	onnxEnvOnce sync.Once
	onnxEnvErr  error
)

// ONNXEngine implements domain.VisionEmbeddingService by running ONNX CLIP vision model in-process in Go.
type ONNXEngine struct {
	modelPath    string
	modelName    string
	session      *ort.AdvancedSession
	inputTensor  *ort.Tensor[float32]
	outputTensor *ort.Tensor[float32]
	mu           sync.Mutex
}

// FindDefaultSharedLibraryPath searches standard paths for libonnxruntime.
func FindDefaultSharedLibraryPath() string {
	candidates := []string{
		os.Getenv("ONNXRUNTIME_SHARED_LIBRARY_PATH"),
		"/opt/homebrew/lib/libonnxruntime.dylib",
		"/usr/local/lib/libonnxruntime.dylib",
		"/usr/lib/libonnxruntime.so",
		"/usr/local/lib/libonnxruntime.so",
		"libonnxruntime.dylib",
		"libonnxruntime.so",
	}
	for _, p := range candidates {
		if p != "" {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

// NewONNXEngine initializes the in-process Go ONNX runtime engine with the specified model file.
func NewONNXEngine(modelPath string, sharedLibPath string) (*ONNXEngine, error) {
	if _, err := os.Stat(modelPath); err != nil {
		return nil, fmt.Errorf("onnx model file not found at %s: %w", modelPath, err)
	}

	if sharedLibPath == "" {
		sharedLibPath = FindDefaultSharedLibraryPath()
	}
	if sharedLibPath == "" {
		return nil, fmt.Errorf("libonnxruntime shared library not found in standard paths")
	}

	onnxEnvOnce.Do(func() {
		ort.SetSharedLibraryPath(sharedLibPath)
		onnxEnvErr = ort.InitializeEnvironment()
	})
	if onnxEnvErr != nil {
		return nil, fmt.Errorf("failed to initialize onnxruntime environment: %w", onnxEnvErr)
	}

	inputShape := ort.NewShape(1, 3, 224, 224)
	inputData := make([]float32, 1*3*224*224)
	inputTensor, err := ort.NewTensor(inputShape, inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to create input tensor: %w", err)
	}

	outputShape := ort.NewShape(1, 512)
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		inputTensor.Destroy()
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}

	session, err := ort.NewAdvancedSession(
		modelPath,
		[]string{"pixel_values"},
		[]string{"image_embeds"},
		[]ort.Value{inputTensor},
		[]ort.Value{outputTensor},
		nil,
	)
	if err != nil {
		inputTensor.Destroy()
		outputTensor.Destroy()
		return nil, fmt.Errorf("failed to create onnx advanced session: %w", err)
	}

	slog.Info("Initialized in-process Go ONNX Vision Engine", "model", modelPath)

	return &ONNXEngine{
		modelPath:    modelPath,
		modelName:    "Xenova/clip-vit-base-patch32",
		session:      session,
		inputTensor:  inputTensor,
		outputTensor: outputTensor,
	}, nil
}

// Close frees the ONNX session and tensors.
func (e *ONNXEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.session != nil {
		_ = e.session.Destroy()
	}
	if e.inputTensor != nil {
		_ = e.inputTensor.Destroy()
	}
	if e.outputTensor != nil {
		_ = e.outputTensor.Destroy()
	}
	return nil
}

// ExtractEmbedding processes image bytes and returns the 512-dim normalized CLIP feature vector.
func (e *ONNXEngine) ExtractEmbedding(ctx context.Context, imageBytes []byte) (domain.FloatVector, string, error) {
	if len(imageBytes) == 0 {
		return nil, "", domain.NewAppError(400, "empty image bytes provided")
	}

	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize to 224x224
	resized := image.NewRGBA(image.Rect(0, 0, 224, 224))
	draw.BiLinear.Scale(resized, resized.Bounds(), img, img.Bounds(), draw.Over, nil)

	// Preprocess into planar [1, 3, 224, 224] normalized RGB
	e.mu.Lock()
	defer e.mu.Unlock()

	tensorData := e.inputTensor.GetData()
	const planeSize = 224 * 224

	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			r, g, b, _ := resized.At(x, y).RGBA()
			rf := float32(r>>8) / 255.0
			gf := float32(g>>8) / 255.0
			bf := float32(b>>8) / 255.0

			idx := y*224 + x
			tensorData[idx] = (rf - defaultCLIPMean[0]) / defaultCLIPStd[0]
			tensorData[planeSize+idx] = (gf - defaultCLIPMean[1]) / defaultCLIPStd[1]
			tensorData[2*planeSize+idx] = (bf - defaultCLIPMean[2]) / defaultCLIPStd[2]
		}
	}

	if err := e.session.Run(); err != nil {
		return nil, "", fmt.Errorf("onnx inference failed: %w", err)
	}

	rawOutput := e.outputTensor.GetData()
	result := make(domain.FloatVector, len(rawOutput))
	copy(result, rawOutput)

	// L2 Normalize
	var sumSq float64
	for _, v := range result {
		sumSq += float64(v) * float64(v)
	}
	norm := float32(math.Sqrt(sumSq))
	if norm > 0 {
		for i := range result {
			result[i] /= norm
		}
	}

	return result, e.modelName, nil
}
