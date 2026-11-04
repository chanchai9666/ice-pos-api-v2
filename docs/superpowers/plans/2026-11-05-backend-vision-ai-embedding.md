# Backend Vision AI Vector Embedding & Matching Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

## Overview

Move feature vector extraction (embedding) from frontend browser Web Workers to the backend (Go API + Python Vision AI sidecar service). The Go backend will automatically extract and store 512-dimension vector embeddings in PostgreSQL when images are uploaded, and provide a server-side Vision Matching API (`POST /api/v1/vision/match`) for instant product recognition.

---

## File Structure & Responsibilities

| File                                               | Responsibility                                                                                    |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `config/config.go`                                 | Add `AI_EMBEDDING_SERVICE_URL` environment configuration                                          |
| `internal/domain/vision.go`                        | Domain models (`VisionMatchItem`, `VisionMatchResponse`), interface `VisionEmbeddingService`      |
| `internal/pkg/calc/vector.go`                      | Cosine similarity calculation between `FloatVector` instances                                     |
| `internal/pkg/calc/vector_test.go`                 | Unit tests for vector math & cosine similarity                                                    |
| `internal/pkg/vision/client.go`                    | HTTP client for external Vision AI embedding service                                              |
| `internal/pkg/vision/client_test.go`               | Unit tests for embedding client (mock server, success & fallback)                                 |
| `services/vision-ai/server.py`                     | Python FastAPI / ONNX server extracting CLIP embeddings                                           |
| `services/vision-ai/requirements.txt`              | Dependencies for vision service (`fastapi`, `uvicorn`, `pillow`, `onnxruntime` or `transformers`) |
| `internal/usecase/vision_usecase.go`               | Business logic for frame matching & batch re-indexing                                             |
| `internal/usecase/vision_usecase_test.go`          | Unit tests for vision matching logic                                                              |
| `internal/delivery/http/handler/upload_handler.go` | Auto-generate embedding on image upload if not supplied                                           |
| `internal/delivery/http/handler/vision_handler.go` | HTTP handlers for `POST /api/v1/vision/match` and `POST /api/v1/vision/reindex`                   |
| `internal/delivery/http/router/router.go`          | Register `/api/v1/vision` routes                                                                  |
| `docs/FRONTEND_VISION_AI_INTEGRATION.md`           | Update frontend guide for backend-driven vector workflow                                          |
| `tests/e2e_vision_ai_test.go`                      | End-to-end integration tests for backend embedding and matching                                   |

---

## Tasks

### Task 1: Vector Math Utilities & Cosine Similarity

- [ ] Write tests in `internal/pkg/calc/vector_test.go` for cosine similarity (identical vectors = 1.0, orthogonal = 0.0, opposite = -1.0, length mismatch = 0.0, zero vectors).
- [ ] Implement `CosineSimilarity(a, b domain.FloatVector) float64` in `internal/pkg/calc/vector.go`.
- [ ] Run `go test ./internal/pkg/calc/...` and verify tests pass.
- [ ] Commit: `feat(calc): add cosine similarity computation for float vectors`.

### Task 2: Domain Models & Configuration

- [ ] Add `AIEmbeddingServiceURL` to `config/config.go` with default `""` (e.g. read from `AI_EMBEDDING_SERVICE_URL`).
- [ ] Create `internal/domain/vision.go` with:
  - `VisionMatchItem`
  - `VisionMatchResponse`
  - `VisionEmbeddingService` interface
  - `VisionUsecase` interface
- [ ] Commit: `feat(vision): add vision domain models and service interfaces`.

### Task 3: Vision AI Embedding Client

- [ ] Write unit tests in `internal/pkg/vision/client_test.go` using `httptest.Server` simulating the embedding service response and connection failure handling.
- [ ] Implement `NewClient(serviceURL string, timeout time.Duration)` in `internal/pkg/vision/client.go` implementing `domain.VisionEmbeddingService`.
- [ ] Run `go test ./internal/pkg/vision/...` and verify tests pass.
- [ ] Commit: `feat(vision): add HTTP client for vision embedding service`.

### Task 4: Lightweight Python Vision AI Sidecar Service

- [ ] Create `services/vision-ai/requirements.txt` with `fastapi`, `uvicorn`, `pillow`, `requests`, and `transformers` / `torch` (or `onnxruntime`).
- [ ] Create `services/vision-ai/server.py` exposing:
  - `GET /health` -> `{"status": "ok", "model": "Xenova/clip-vit-base-patch32"}`
  - `POST /embed` -> accepts multipart image file, returns `{"embedding": [...512 floats], "model": "Xenova/clip-vit-base-patch32"}`.
- [ ] Add Makefile target `run-vision-ai` to launch the service on port `5001`.
- [ ] Commit: `feat(services): add Python FastAPI vision AI embedding sidecar`.

### Task 5: Auto-Generate Embeddings in Upload Handler & Catalog Usecase

- [ ] Inject `VisionEmbeddingService` into `UploadHandler`.
- [ ] In `AddProductImage`:
  - If incoming `embedding` is empty and `VisionEmbeddingService` is configured, call `ExtractEmbedding` on uploaded image bytes.
  - Set generated vector and model name before passing to `catalogUsecase.AddProductImage`.
- [ ] Update `UploadProductImage` (single primary image endpoint) similarly so primary image also gets vector.
- [ ] Test upload with mock embedding service in tests.
- [ ] Commit: `feat(upload): auto-extract vector embedding on product image upload`.

### Task 6: Server-side Vision Match & Reindex Handlers

- [ ] Implement `internal/usecase/vision_usecase.go`:
  - `MatchFrame(ctx context.Context, imageBytes []byte, threshold float64, limit int) (*domain.VisionMatchResponse, error)`
  - `ReindexImages(ctx context.Context) (int, error)`
- [ ] Implement `internal/delivery/http/handler/vision_handler.go`:
  - `POST /api/v1/vision/match`
  - `POST /api/v1/vision/reindex`
- [ ] Register routes in `internal/delivery/http/router/router.go`.
- [ ] Commit: `feat(api): add server-side vision match and reindex endpoints`.

### Task 7: E2E Integration Testing & Documentation

- [ ] Create `tests/e2e_vision_ai_test.go` covering:
  - Uploading product image without embedding -> backend auto-embeds and saves to DB.
  - Matching an image frame -> returns matched product with confidence score >= threshold.
  - Reindex missing embeddings endpoint.
- [ ] Run full test suite: `go test ./...` to ensure no regressions.
- [ ] Update `docs/FRONTEND_VISION_AI_INTEGRATION.md` and `docs/API_SPEC.md` with instructions for backend-driven vision AI.
- [ ] Commit: `docs(vision): update frontend and api docs for backend vector embedding`.
