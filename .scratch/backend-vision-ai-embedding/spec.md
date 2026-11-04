# Spec: Backend Vision AI Vector Embedding & Matching

## Status

ready-for-agent

## Background & Motivation

In the previous client-side vision AI architecture, Frontend (Admin UI and POS Cashier) used `@xenova/transformers` (running `Xenova/clip-vit-base-patch32` inside browser Web Workers). Testing revealed that client-side vector extraction is prohibitively slow and resource-intensive on browser/POS client devices (causing UI freezes, heavy downloads of ~100MB model weights, and 2-10 second latency per frame/image).

To resolve this bottleneck, feature vector extraction and matching must be relocated to the Backend (Go API + dedicated AI Vision embedding service/worker).

## Goals & Requirements

1. **Automatic Vector Generation on Upload**:
   - When an Admin uploads a product image via `POST /api/v1/products/:id/images` without an `embedding` field, the Backend automatically generates the feature vector (512-dim float array) and model name (`clip-vit-base-patch32`) before saving to PostgreSQL.
   - Backward compatible: if a client explicitly provides an embedding, it can still be accepted.
2. **Server-side Vision Matching API**:
   - Provide `POST /api/v1/vision/match` (or `POST /api/v1/products/match-frame`):
     - Accepts a camera frame image (multipart form-data or JSON with base64 / vector).
     - Backend extracts query embedding (if image provided).
     - Calculates Cosine Similarity against product image embeddings stored in PostgreSQL.
     - Returns top matches above threshold with product details and confidence score.
3. **Pluggable Vision Embedding Service in Backend**:
   - Clean Architecture service interface: `VisionEmbeddingService` with method:
     `ExtractEmbedding(ctx context.Context, imageBytes []byte) (domain.FloatVector, string, error)`
   - Implementation: HTTP Client connecting to a fast Python/FastAPI embedding worker or external embedding endpoint (configured via `AI_EMBEDDING_SERVICE_URL`, e.g., `http://localhost:5001/embed` or external provider).
   - Graceful fallback: If AI embedding service is offline or not configured, image upload still succeeds (with warning logged), allowing re-indexing later via batch endpoint or patch.
4. **Batch Re-index Endpoint for Existing Images**:
   - `POST /api/v1/vision/reindex`: Recomputes embeddings for all product images in PostgreSQL that currently lack embeddings.
5. **PostgreSQL Database Storage**:
   - Stored in `product_images.embedding` and `product_images.embedding_model`.
