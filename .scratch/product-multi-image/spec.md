# Spec: Multi-Image & Visual AI Matching for Products

## Summary

Support multiple images per product with associated visual AI embeddings (vector arrays) and model metadata for frontend camera-based product recognition (via onnxruntime-web / transformers.js) and offline caching in IndexedDB.

## Background & Motivation

In POS cashier operations, scanning products via camera (visual AI recognition) requires comparing the camera image embedding against reference image embeddings of products. To achieve high accuracy across different product angles and lighting, a product requires multiple reference images. Since matching runs on the frontend via onnxruntime-web / transformers.js on cached IndexedDB data, the API must store multiple images and their pre-computed feature vectors and synchronize them efficiently.

## Design Decisions

1. **Model**: New `product_images` table (`domain.ProductImage`).
   - Fields: `id`, `product_id`, `image_url`, `is_primary`, `sort_order`, `embedding` (JSON []float32), `embedding_model`, `created_at`, `updated_at`.
   - `Product` has `Images []ProductImage` relation (`foreignKey:ProductID;constraint:OnDelete:CASCADE`).
   - Maintain `product.image_url` synced with the primary image for 100% backward compatibility.
2. **Feature Extraction**:
   - Frontend (Admin UI) runs onnxruntime-web/transformers.js to compute the embedding vector on client-side before or during upload, sending it in the upload request.
   - Go backend remains lightweight (no CGo/Python ONNX runtime needed).
3. **Database Portability**:
   - `Embedding` stored as serialized JSON array of floats (`[]float32`), supported natively by both SQLite and PostgreSQL.
4. **Quota & Deletion**:
   - Maximum 10 images per product.
   - Physical image file in storage is deleted when `ProductImage` is deleted.
   - If deleting primary image, promote next image to primary and update `product.image_url`.
5. **Backfill / Migration**:
   - AutoMigrate creates `product_images` table.
   - On startup / DB migration, existing `product.image_url` records are backfilled into `product_images` as primary images.
6. **Endpoints**:
   - `POST /api/v1/products/:id/images`: Upload image + embedding + embedding_model + is_primary.
   - `DELETE /api/v1/products/:id/images/:imageId`: Delete product image & physical file, auto-promote next primary if needed.
   - `PATCH /api/v1/products/:id/images/:imageId/primary`: Set as primary image.
   - `PATCH /api/v1/products/:id/images/:imageId/embedding`: Update embedding vector and model metadata without re-uploading file.
   - `POST /api/v1/products/:id/image`: Legacy endpoint kept for backward compatibility (updates primary image).
   - `GET /api/v1/products`, `GET /api/v1/products/:id`, and `GET /api/v1/sync/changes`: Preload `Images` with embeddings.
