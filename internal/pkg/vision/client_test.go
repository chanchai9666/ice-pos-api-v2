package vision

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ice-pos-api-v2/internal/domain"
)

func TestExtractEmbedding_Success(t *testing.T) {
	expectedEmbedding := domain.FloatVector{0.1, 0.2, 0.3}
	expectedModel := "Xenova/clip-vit-base-patch32"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/embed" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Ensure file was uploaded
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("missing file part: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"embedding": expectedEmbedding,
			"model":     expectedModel,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)
	emb, model, err := client.ExtractEmbedding(context.Background(), []byte("fake-image-bytes"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model != expectedModel {
		t.Errorf("expected model %s, got %s", expectedModel, model)
	}

	if len(emb) != len(expectedEmbedding) {
		t.Fatalf("expected len %d, got %d", len(expectedEmbedding), len(emb))
	}
	for i := range emb {
		if emb[i] != expectedEmbedding[i] {
			t.Errorf("emb[%d] = %v, want %v", i, emb[i], expectedEmbedding[i])
		}
	}
}

func TestExtractEmbedding_ServerOffline_ReturnsError(t *testing.T) {
	// Point to unreachable port
	client := NewClient("http://127.0.0.1:54321", 500*time.Millisecond)
	_, _, err := client.ExtractEmbedding(context.Background(), []byte("fake-image-bytes"))
	if err == nil {
		t.Fatalf("expected error when server is offline, got nil")
	}
}

func TestExtractEmbedding_Unconfigured_ReturnsError(t *testing.T) {
	client := NewClient("", 500*time.Millisecond)
	_, _, err := client.ExtractEmbedding(context.Background(), []byte("fake-image-bytes"))
	if err == nil {
		t.Fatalf("expected error when service URL is empty, got nil")
	}
}
