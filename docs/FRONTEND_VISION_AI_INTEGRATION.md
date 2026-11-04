# คู่มือการเชื่อมต่อระบบ Visual AI Matching (onnxruntime-web / transformers.js) สำหรับหน้าบ้าน

เอกสารฉบับนี้อธิบายสถาปัตยกรรม วิธีการเชื่อมต่อ API และโค้ดตัวอย่างสำหรับทีมพัฒนา Frontend ในการพัฒนาระบบสแกนหาสินค้าจากภาพถ่ายผ่านกล้อง (Camera-based Product Recognition) โดยใช้ **onnxruntime-web** หรือ **@xenova/transformers (transformers.js)** พร้อมแคชข้อมูลแบบ Offline-first บน **IndexedDB**

---

## 1. ภาพรวมสถาปัตยกรรม (Architecture Overview)

```
+-----------------------------------------------------------------------------------+
|                                 ฝั่งหน้าบ้าน (Frontend)                           |
+-----------------------------------------------------------------------------------+
|                                                                                   |
|  [Admin UI - เพิ่มสินค้า]                                                          |
|    1. เลือกรูปสินค้า (หลายมุมมอง: หน้า/หลัง/ข้าง)                                      |
|    2. อัปโหลดรูปภาพปกติขึ้น Backend API (ไม่ต้องแปลง Vector ที่หน้าบ้านแล้ว!)        |
|                                                                                   |
|  [POS Cashier - หน้าขาย/แคชเชียร์ (เลือกใช้งานได้ 2 โหมด)]                          |
|    โหมด A (Server-side Match - แนะนำ เร็วและไม่กินเครื่อง):                           |
|      - ส่งเฟรมภาพไปที่ POST /api/v1/vision/match -> ได้สินค้าที่ตรงกันทันที            |
|    โหมด B (Offline Edge Match):                                                   |
|      - Sync ข้อมูลสินค้า + Pre-computed Embeddings จาก Backend ลง IndexedDB       |
|      - คำนวณ Cosine Similarity ภายในเครื่องหน้าร้านแบบ Real-time / Offline          |
|                                                                                   |
+-----------------------------------------------------------------------------------+
                                         |
                                         | REST API (HTTP / JSON / Multipart)
                                         v
+-----------------------------------------------------------------------------------+
|                              ฝั่งหลังบ้าน (Backend API)                           |
+-----------------------------------------------------------------------------------+
|  1. จัดการรูปภาพ (บีบอัดเป็น WebP <= 50KB อัตโนมัติ)                                  |
|  2. สกัด Feature Vector อัตโนมัติ (CLIP Vision AI Sidecar) -> []float32           |
|  3. บันทึก ProductImage: ImageURL + Vector Embedding ([]float32) ลง PostgreSQL     |
|  4. ให้บริการ Server-side Matching: POST /api/v1/vision/match                     |
|  5. ส่งชุดข้อมูลพร้อมซิงค์ใน GET /products และ /api/v1/sync/changes                  |
|  6. อัปเดต primary image_url ของสินค้าให้อัตโนมัติ (Backward Compatible 100%)      |
+-----------------------------------------------------------------------------------+
```

---

## 2. โครงสร้างข้อมูลใน API (Data Schema)

### ฟิลด์ใน `Product`

```typescript
interface Product {
  id: string;
  code: string;
  barcode: string;
  name: string;
  price: number; // สตางค์
  image_url: string; // รูปหลัก (Primary thumbnail สำหรับแสดงบนการ์ดสินค้า)
  images: ProductImage[]; // รายการรูปภาพทั้งหมดของสินค้า (พร้อม Embedding)
}

interface ProductImage {
  id: string;
  product_id: string;
  image_url: string; // URL รูปภาพ WebP (ขนาด <= 50KB)
  is_primary: boolean; // เป็นรูปหลักหรือไม่
  sort_order: number;
  embedding?: number[]; // Vector Array เช่น [0.124, -0.456, 0.789, ...]
  embedding_model?: string; // ชื่อโมเดลที่ใช้ เช่น "Xenova/clip-vit-base-patch32"
  created_at: string;
  updated_at: string;
}
```

---

## 3. สิ่งที่หน้าบ้านต้องเตรียม (Prerequisites & Libraries)

ติดตั้งแพ็กเกจสำหรับรันโมเดล Machine Learning ใน Browser:

```bash
# กรณีใช้ Transformers.js (แนะนำ - ใช้งานง่าย โหลดโมเดล Vision ได้สะดวกรวดเร็ว)
npm install @xenova/transformers

# หรือกรณีใช้ ONNX Runtime Web ตรง
npm install onnxruntime-web

# แนะนำ Dexie.js สำหรับจัดการ IndexedDB ได้อย่างสะดวกรวดเร็ว
npm install dexie
```

> **โมเดลที่แนะนำสำหรับการเปรียบเทียบรูปภาพสินค้า (Image Feature Extraction):**
>
> 1. `Xenova/clip-vit-base-patch32` (Zero-shot visual embedding ขนาด 512 มิติ ความแม่นยำสูงมาก)
> 2. `Xenova/mobilenet_v3_small` หรือ `Xenova/mobilenet_v4_conv_small` (เน้นความเร็วและประหยัดแรมบนมือถือสเปกต่ำ)

---

## 4. ขั้นตอนที่ 1: การอัปโหลดรูปภาพพร้อม Vector (Admin Portal)

เมื่อผู้ดูแลระบบเพิ่มรูปสินค้า (เช่น ถ่ายรูปสินค้าจากหลายมุม: ด้านหน้า, ด้านหลัง, กล่องบรรจุภัณฑ์):

```typescript
import { pipeline, env } from "@xenova/transformers";

// ปรับให้แคชโมเดลใน IndexedDB ของ Browser
env.allowLocalModels = false;
env.useBrowserCache = true;

const MODEL_NAME = "Xenova/clip-vit-base-patch32";
let featureExtractor: any = null;

// โหลดโมเดลล่วงหน้า
async function initVisionModel() {
  if (!featureExtractor) {
    featureExtractor = await pipeline("image-feature-extraction", MODEL_NAME);
  }
  return featureExtractor;
}

/**
 * สกัด Embedding Vector จากไฟล์รูปภาพ
 */
export async function extractImageEmbedding(
  imageFile: File
): Promise<number[]> {
  const extractor = await initVisionModel();
  const imageURL = URL.createObjectURL(imageFile);

  // รันโมเดลเพื่อดึง Feature Vector
  const output = await extractor(imageURL, {
    pooling: "mean",
    normalize: true,
  });
  URL.revokeObjectURL(imageURL);

  // แปลง Float32Array เป็น standard JavaScript Array
  return Array.from(output.data);
}

/**
 * อัปโหลดรูปภาพสินค้าขึ้น API
 * (Backend จะทำการสกัด Feature Vector และบันทึกลง PostgreSQL ให้อัตโนมัติ โดยหน้าบ้านไม่ต้องประมวลผลเอง)
 */
export async function uploadProductImage(
  productId: string,
  imageFile: File,
  token: string,
  isPrimary: boolean = false
) {
  // ส่ง Multipart Form ไปที่ Backend API ได้โดยตรง ไม่ต้องรัน transformers.js ที่หน้าบ้าน
  const formData = new FormData();
  formData.append("file", imageFile);
  formData.append("is_primary", isPrimary ? "true" : "false");

  const res = await fetch(`/api/v1/products/${productId}/images`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
    },
    body: formData,
  });

  if (!res.ok) {
    const errorData = await res.json();
    throw new Error(
      errorData.error?.message ||
        errorData.message ||
        "เกิดข้อผิดพลาดในการอัปโหลดรูปภาพ"
    );
  }

  return await res.json();
}
```

---

## 5. ขั้นตอนที่ 2: การบันทึกและซิงค์ข้อมูลลง IndexedDB (Offline Storage)

กำหนดโครงสร้างฐานข้อมูล IndexedDB ด้วย Dexie:

```typescript
import Dexie, { Table } from "dexie";

export interface LocalProduct {
  id: string;
  code: string;
  barcode: string;
  name: string;
  price: number;
  image_url: string;
  images: Array<{
    id: string;
    image_url: string;
    is_primary: boolean;
    embedding?: number[];
    embedding_model?: string;
  }>;
}

class PosDatabase extends Dexie {
  products!: Table<LocalProduct, string>;

  constructor() {
    super("IcePosDB");
    this.version(1).stores({
      products: "id, barcode, code, name",
    });
  }
}

export const db = new PosDatabase();

/**
 * ดึงข้อมูลล่าสุดจาก API มาแคชลง IndexedDB (รองรับ Delta Sync)
 */
export async function syncCatalogFromAPI(token: string, lastSyncTime?: string) {
  const url = lastSyncTime
    ? `/api/v1/sync/changes?since=${encodeURIComponent(lastSyncTime)}`
    : `/api/v1/sync/changes`;

  const res = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!res.ok) throw new Error("Sync failed");
  const result = await res.json();
  const products: LocalProduct[] = result.data.products;

  // บันทึกลง IndexedDB
  await db.transaction("rw", db.products, async () => {
    for (const p of products) {
      await db.products.put(p);
    }
  });

  return result.data.server_time;
}
```

---

## 6. ขั้นตอนที่ 3: ระบบสแกนหาสินค้าจากกล้อง (Camera Scanner & Matching)

สร้างฟังก์ชันคำนวณ **Cosine Similarity** เพื่อเทียบความคล้ายคลึงของภาพ:

```typescript
/**
 * คำนวณ Cosine Similarity ระหว่าง Vector สองชุด (-1.0 ถึง 1.0)
 * หาก Vector ถูก Normalize มาแล้ว (L2 Normalized) ค่า Cosine Similarity คือ Dot Product
 */
function cosineSimilarity(vecA: number[], vecB: number[]): number {
  let dotProduct = 0;
  let normA = 0;
  let normB = 0;

  for (let i = 0; i < vecA.length; i++) {
    dotProduct += vecA[i] * vecB[i];
    normA += vecA[i] * vecA[i];
    normB += vecB[i] * vecB[i];
  }

  if (normA === 0 || normB === 0) return 0;
  return dotProduct / (Math.sqrt(normA) * Math.sqrt(normB));
}

/**
 * สแกนเฟรมภาพจาก <video> กล้องและค้นหาสินค้าที่ตรงกันที่สุด
 */
export async function matchProductFromCamera(
  videoElement: HTMLVideoElement,
  confidenceThreshold = 0.8 // เกณฑ์ขั้นต่ำในการยอมรับ (ปรับได้ 0.75 - 0.85)
) {
  // 1. แคปเจอร์ภาพจากกล้องลง Canvas ชั่วคราว
  const canvas = document.createElement("canvas");
  canvas.width = videoElement.videoWidth || 320;
  canvas.height = videoElement.videoHeight || 320;
  const ctx = canvas.getContext("2d");
  if (!ctx) return null;
  ctx.drawImage(videoElement, 0, 0, canvas.width, canvas.height);

  // 2. รันโมเดล Vision ดึง Query Vector
  const extractor = await initVisionModel();
  const dataURL = canvas.toDataURL("image/jpeg", 0.8);
  const output = await extractor(dataURL, { pooling: "mean", normalize: true });
  const queryVector = Array.from(output.data) as number[];

  // 3. วนลูปเทียบกับสินค้าใน IndexedDB
  const allProducts = await db.products.toArray();
  let bestMatch: LocalProduct | null = null;
  let highestScore = -1;

  for (const prod of allProducts) {
    if (!prod.images || prod.images.length === 0) continue;

    for (const img of prod.images) {
      if (!img.embedding || img.embedding.length !== queryVector.length)
        continue;

      const score = cosineSimilarity(queryVector, img.embedding);
      if (score > highestScore) {
        highestScore = score;
        bestMatch = prod;
      }
    }
  }

  // 4. ตรวจสอบเงื่อนไข Threshold
  if (highestScore >= confidenceThreshold && bestMatch) {
    return {
      product: bestMatch,
      confidence: highestScore, // เช่น 0.89 (89%)
    };
  }

  return null; // ไม่พบสินค้าที่ความมั่นใจถึงเกณฑ์
}
```

---

## 7. ขั้นตอนที่ 4: การหยิบสินค้าลงตะกร้าขาย (Auto Add to Cart)

ในหน้าจอขายของแคชเชียร์ สามารถรันลูปตรวจสอบกล้องเป็นระยะ (เช่น ทุก 500ms):

```typescript
let isScanning = false;
let lastMatchedProductId = "";
let lastMatchTime = 0;

async function onCameraFrameTick(videoElement: HTMLVideoElement) {
  if (isScanning) return;
  isScanning = true;

  try {
    const result = await matchProductFromCamera(videoElement, 0.82);

    if (result) {
      const now = Date.now();
      // Debounce ป้องกันการเพิ่มสินค้าซ้ำรัวๆ หากกล้องยังส่องค้างอยู่ที่เดิม
      if (
        result.product.id !== lastMatchedProductId ||
        now - lastMatchTime > 2500
      ) {
        lastMatchedProductId = result.product.id;
        lastMatchTime = now;

        // สั่นหรือส่งเสียง Beep แจ้งเตือน
        playBeepSound();
        if (navigator.vibrate) navigator.vibrate(100);

        // เพิ่มสินค้าลงตะกร้า
        cartStore.addItem({
          productId: result.product.id,
          name: result.product.name,
          price: result.product.price,
          qty: 1,
        });

        toast.success(
          `เพิ่ม "${result.product.name}" ลงตะกร้าแล้ว (ความแม่นยำ ${(result.confidence * 100).toFixed(1)}%)`
        );
      }
    }
  } catch (err) {
    console.error("Matching error:", err);
  } finally {
    isScanning = false;
  }
}
```

---

## 8. ทางเลือกใหม่: ค้นหาสินค้าจากกล้องผ่าน Backend API (Server-side Match)

เพื่อแก้ปัญหาเครื่องแคชเชียร์หรืออุปกรณ์สเปกต่ำรันโมเดล Vision AI ได้ช้า สามารถส่งเฟรมภาพตรงไปยัง Backend API เพื่อให้เซิร์ฟเวอร์ประมวลผล Vector และคำนวณ Cosine Similarity จากฐานข้อมูล PostgreSQL โดยตรง:

```typescript
/**
 * ส่งภาพเฟรมจากกล้องไป Match กับสินค้าที่ฝั่งหลังบ้าน
 */
async function matchProductViaBackend(
  blob: Blob,
  threshold = 0.75
): Promise<Product | null> {
  const formData = new FormData();
  formData.append("file", blob, "frame.jpg");
  formData.append("threshold", threshold.toString());
  formData.append("limit", "1");

  const res = await fetch("/api/v1/vision/match", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
    },
    body: formData,
  });

  if (!res.ok) return null;
  const data = await res.json();
  const matches = data.data?.matches;
  if (matches && matches.length > 0) {
    return matches[0];
  }
  return null;
}
```

---

## 9. การจัดการรูปภาพและ Vector อื่นๆ ผ่าน API

| การทำงาน                   | Method & URL                                           | ตัวอย่าง Payload / Form Field                                                  |
| :------------------------- | :----------------------------------------------------- | :----------------------------------------------------------------------------- |
| **เพิ่มรูปใหม่**           | `POST /api/v1/products/:id/images`                     | Multipart: `file`, `is_primary` (Backend สกัด Vector ให้อัตโนมัติ)             |
| **ค้นหาสินค้าจากภาพ**      | `POST /api/v1/vision/match`                            | Multipart: `file`, `threshold`, `limit`                                        |
| **สร้าง Vector ย้อนหลัง**  | `POST /api/v1/vision/reindex`                          | สแกนสร้าง Vector ให้รูปสินค้าที่ยังไม่มีใน DB                                  |
| **ดูรูปทั้งหมด**           | `GET /api/v1/products/:id/images`                      | คืนรายการ Array ของรูปภาพพร้อม Embedding                                       |
| **ลบรูปภาพ**               | `DELETE /api/v1/products/:id/images/:imageId`          | ลบเรคคอร์ด + ลบไฟล์จริงในเซิร์ฟเวอร์ (ถ้าลบรูปหลัก รูปถัดไปจะขึ้นแทนอัตโนมัติ) |
| **เปลี่ยนรูปหลัก**         | `PATCH /api/v1/products/:id/images/:imageId/primary`   | ตั้งรูปนี้เป็นรูปหลักสำหรับแสดงบน POS Card                                     |
| **อัปเดตเวกเตอร์ย้อนหลัง** | `PATCH /api/v1/products/:id/images/:imageId/embedding` | JSON Body: `{"embedding": [...], "embedding_model": "..."}`                    |

> [!TIP]
> **การอัปเกรดโมเดล Vision AI ในอนาคต:**  
> หากในอนาคตต้องการเปลี่ยนโมเดล (เช่น จาก MobileNet เป็น CLIP รุ่นใหม่) ไม่จำเป็นต้องถ่ายรูปสินค้าใหม่ทั้งหมด หน้าบ้านสามารถเขียนสคริปต์วนลูปดาวน์โหลดรูปภาพจาก `image_url` เดิม มารันโมเดลใหม่เพื่อสกัด Embedding แล้วส่ง `PATCH /products/:id/images/:imageId/embedding` ขึ้นไปอัปเดตได้ทันที หรือเรียก `POST /api/v1/vision/reindex`!

---

## 10. ข้อแนะนำเพื่อเพิ่มความแม่นยำ (Best Practices)

1. **จำนวนรูปต่อสินค้า:** ควรถ่ายสินค้าอย่างน้อย 2–4 รูปต่อสินค้า (เช่น มุมตรงหน้าซอง/ขวด, มุมด้านหลังที่มีบาร์โค้ดหรือตารางโภชนาการ, มุมเอียง 45 องศา, และมุมกล่องลัง)
2. **แสงและพื้นหลัง:** แนะนำให้ถ่ายรูปอ้างอิงบนพื้นหลังสีเรียบ (เช่น โต๊ะเคาน์เตอร์สีขาวหรือเทา) เพื่อให้ Feature Vector จับจุดเด่นของบรรจุภัณฑ์ได้คมชัดที่สุด
3. **การตั้งค่า Threshold:**
   - `0.80 - 0.85`: เหมาะสำหรับการเพิ่มลงตะกร้าอัตโนมัติ (High Precision)
   - `0.70 - 0.79`: เหมาะสำหรับแสดงเป็นรายการสินค้าแนะนำให้แคชเชียร์แตะยืนยัน (Suggested Items)
4. **Web Workers:** แนะนำให้ย้ายฟังก์ชันคำนวณ `cosineSimilarity` และการสกัดภาพไปรันใน **Web Worker** เพื่อไม่ให้กระทบต่อเฟรมเรตและความลื่นไหลของหน้าจอ UI
