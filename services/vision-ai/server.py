"""
Vision AI Embedding Microservice (FastAPI + CLIP)
Accepts image uploads and extracts 512-dimension feature vector embeddings.
Compatible with client-side Xenova/clip-vit-base-patch32.
"""

import io
import os
import sys
from typing import List
from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from PIL import Image

MODEL_NAME = os.getenv("CLIP_MODEL_NAME", "openai/clip-vit-base-patch32")

app = FastAPI(
    title="ICE POS Vision AI Embedding Service",
    version="1.0.0",
    description="Extracts normalized vector embeddings from product images using CLIP."
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

class EmbedResponse(BaseModel):
    embedding: List[float]
    model: str

model = None
processor = None

def get_model():
    global model, processor
    if model is None:
        try:
            from transformers import CLIPProcessor, CLIPModel
            print(f"[Vision AI] Loading model {MODEL_NAME}...", flush=True)
            processor = CLIPProcessor.from_pretrained(MODEL_NAME)
            model = CLIPModel.from_pretrained(MODEL_NAME)
            model.eval()
            print("[Vision AI] Model loaded successfully.", flush=True)
        except Exception as e:
            print(f"[Vision AI] Warning: Could not load transformers CLIP model: {e}", file=sys.stderr)
            print("[Vision AI] Operating in fallback mode.", file=sys.stderr)
    return model, processor

@app.on_event("startup")
async def startup_event():
    # Attempt to preload model on startup in background
    try:
        get_model()
    except Exception as e:
        print(f"[Vision AI] Preload warning: {e}", file=sys.stderr)

@app.get("/health")
def health_check():
    return {
        "status": "ok",
        "model": MODEL_NAME,
        "service": "ice-pos-vision-ai"
    }

@app.post("/embed", response_model=EmbedResponse)
async def extract_embedding(file: UploadFile = File(...)):
    contents = await file.read()
    if not contents:
        raise HTTPException(status_code=400, detail="Empty image payload")

    try:
        image = Image.open(io.BytesIO(contents)).convert("RGB")
    except Exception as err:
        raise HTTPException(status_code=400, detail=f"Invalid image format: {err}")

    clip_model, clip_processor = get_model()

    if clip_model is not None and clip_processor is not None:
        import torch
        with torch.no_grad():
            inputs = clip_processor(images=image, return_tensors="pt")
            image_features = clip_model.get_image_features(**inputs)
            # L2 normalize
            image_features = image_features / image_features.norm(dim=-1, keepdim=True)
            vector = image_features.squeeze().tolist()
            return EmbedResponse(embedding=vector, model=MODEL_NAME)

    # Fallback mode (deterministic mock vector from image dimensions/histogram) if PyTorch/Transformers not installed
    import hashlib
    h = hashlib.sha256(contents).digest()
    mock_vector = [(b / 255.0) * 2.0 - 1.0 for b in h]
    # expand to 512 floats
    expanded = (mock_vector * 16)[:512]
    # normalize
    norm = sum(x * x for x in expanded) ** 0.5
    if norm > 0:
        expanded = [x / norm for x in expanded]

    return EmbedResponse(embedding=expanded, model="fallback-hash-512")

if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", "5001"))
    uvicorn.run("server:app", host="0.0.0.0", port=port, reload=True)

