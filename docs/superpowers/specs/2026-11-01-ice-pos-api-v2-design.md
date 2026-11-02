# Design Document: ICE POS API v2 (Simplified Single-Branch Cash & Credit POS)

**Date:** 2026-11-02  
**Version:** 2.0.0  
**Project:** `ice-pos-api-v2`  
**Target User:** ผู้สูงอายุ / ร้านค้าสาขาเดียวที่ต้องการความเรียบง่ายสูงสุด  
**Core Purpose:** ค้นหาราคาสินค้า, คิดเงิน/บันทึกการขายเงินสดและลูกหนี้เงินเชื่อ, คำนวณเงินทอน, สะสมแต้มสมาชิก, ยกเลิกบิลได้ง่าย, รองรับการทำงานออฟไลน์ (Offline-first Sync), และดูรายงานยอดขาย/รายงานสินค้าขายดี/รายงานลูกหนี้ค้างชำระ (AR Aging) แบบไดนามิก

---

## 1. Intent & The Why

ระบบเดิม (`ice-pos-api` v1) มีความซับซ้อนสูง ทั้งการจัดการสาขา, คลังสินค้า/จัดซื้อ, ระบบกะและลิ้นชัก, และช่องทางชำระเงินที่หลากหลาย

สำหรับ **v2** กลุ่มผู้ใช้งานหลักคือ **ผู้สูงอายุ / ร้านค้าสาขาเดียว** ซึ่งต้องการระบบที่:

1. **ไม่ยุ่งยาก ไม่ซับซ้อน:** เปิดระบบแล้วพร้อมขายทันที ไม่ต้องเปิดกะ ไม่ต้องนับเงินตั้งต้น
2. **ไม่สะดุดเรื่องสต็อก:** จัดการสต็อกผ่านช่องทางภายนอกอยู่แล้ว หน้าร้านต้องการขายได้ตลอดเวลา ไม่มีการบล็อกขายเพราะสต็อกหมด
3. **จ่ายเงินสดและคำนวณเงินทอน:** รับเงินสดมาเท่าไหร่ ระบบคำนวณเงินทอนให้อัตโนมัติ ป้องกันการคิดเลขผิด
4. **ระบบสมาชิกร้านค้าและลูกหนี้เงินเชื่อ (Customer Credit):** รองรับลูกค้าประจำ/สมาชิก เปิดวงเงินเชื่อ (Credit Limit) ซื้อเชื่อและชำระหนี้ภายหลัง พร้อมคำนวณแต้มสะสม
5. **ยกเลิกบิลผิดพลาดได้ง่าย (Void Sale):** แคชเชียร์สามารถกดยกเลิกบิลที่ยิงผิดหรือคิดผิดได้เอง ไม่ต้องรอสิทธิ์ Admin
6. **ทำงานออฟไลน์ได้ (Offline-First):** หากสัญญาณอินเทอร์เน็ตหลุด หน้าร้านยังสามารถดึงข้อมูลแคตตาล็อกสินค้ามาคิดเงิน และส่งบิลขายขึ้นเซิร์ฟเวอร์แบบ Batch ทันทีที่เน็ตกลับมา
7. **รูปภาพสินค้าโหลดเร็วและเบา (<= 50KB):** บีบอัดและย่อรูปภาพเป็น WebP ไม่เกิน 50KB อัตโนมัติ เพื่อประหยัดพื้นที่จัดเก็บและเพิ่มความเร็วในการโหลดหน้าร้าน
8. **ดูสรุปและสินค้าขายดีได้:** ตรวจสอบยอดขายประจำวันและดูสินค้าขายดีได้แบบ Dynamic ตามวันที่เลือก ไม่ต้องมีขั้นตอน "ปิดยอดสิ้นวัน"

---

## 2. Goals & Architecture Highlights

### Goals & Features

- **Clean Architecture Foundation:** พัฒนาด้วย Clean Architecture (Delivery -> Usecase -> Domain <- Repository) ขับเคลื่อนด้วย Go Fiber v3 + GORM (รองรับ SQLite และ PostgreSQL)
- **Financial Precision:** คำนวณเงินทุกประเภทด้วย `github.com/shopspring/decimal` (หน่วยเงินเป็นสตางค์ เพื่อความแม่นยำ 100%)
- **Cash & Credit Checkout:**
  - ค้นหาสินค้า / ยิงบาร์โค้ด
  - ดึงราคากลางที่ถูกต้อง (Server-authoritative pricing)
  - รับชำระเงินสด (`CASH`): คำนวณเงินทอนอัตโนมัติ
  - รับชำระเงินเชื่อ (`CREDIT`): ตรวจสอบวงเงินคงเหลือของลูกค้า และบันทึกยอดค้างชำระลงใน `credit_accounts` ทันที
  - คำนวณแต้มสมาชิก: 1 แต้มต่อทุก ๆ 100 บาท (10,000 สตางค์)
  - บันทึกการขาย (Sale + Sale Items) ออกเลขที่ใบเสร็จทันที
  - บันทึกต้นทุนสินค้า (`cost_price`) ณ เวลาขาย เพื่อคำนวณกำไรขั้นต้นในรายงาน
- **Void Sale (ยกเลิกบิล):** รองรับ `POST /api/v1/sales/:id/void` แคชเชียร์กดยกเลิกได้เอง บิลที่ยกเลิกจะถูกตัดออกจากยอดขายรวม และหากเป็นบิลเงินเชื่อจะคืนยอดหนี้ให้อัตโนมัติ
- **Customer & Member Management:**
  - ระบบสมาชิกและสะสมแต้ม
  - ระบบบัญชีเงินเชื่อ (Credit Limit, Outstanding Balance, Status)
  - การรับชำระหนี้เงินเชื่อ (`POST /customers/:id/pay-credit`)
  - รายงานอายุลูกหนี้ (`GET /customers/reports/ar-aging`) จัดกลุ่มตาม 0-30, 31-60, 61-90, 90+ วัน
- **Image Optimization & Storage (<= 50KB):**
  - รองรับการอัปโหลดไฟล์รูปภาพความละเอียดสูง (สูงสุด 10MB)
  - ระบบจะทำ Dynamic WebP Compression & Auto-Downscale เพื่อให้ไฟล์สุดท้ายมีขนาด **ไม่เกิน 50KB** เสมอ
  - รองรับ Storage Drivers: Local Disk และ ImageKit CDN
- **Lightweight Offline Sync Engine:**
  - `GET /api/v1/sync/changes?since=...`: ดาวน์โหลดสินค้า/หมวดหมู่ที่อัปเดตลงแท็บเล็ตหน้าร้าน
  - `POST /api/v1/sync/batch`: ส่งชุดข้อมูลบิลขาย (ทั้ง `COMPLETED` และ `VOID`) ขึ้นเซิร์ฟเวอร์แบบ Idempotent
- **Authentication & Security:**
  - เข้าสู่ระบบด้วย Username/Password หรือ Cashier PIN
  - JWT Access Token (ระยะเวลาสั้น) + Refresh Token (HttpOnly Cookie หรือ Request Body)
  - Role-based Access Control (`ADMIN`, `CASHIER`)
- **API Documentation:**
  - Interactive Swagger OpenAPI 2.0 UI ที่ `/swagger`

### Features Intentionally Removed (Non-Goals)

- **Unnecessary Checkout Ceremony:** ตัด PromptPay QR ซับซ้อน, เครื่องรูดบัตรเครดิตภายนอก (เหลือเพียง Cash และ Customer Credit สำหรับร้านค้าเดี่ยว)
- **Inventory & Procurement:** ตัด Suppliers, Purchase Orders (PO), Goods Receipts (GRN), Stock Adjustments, Stock Transfers, Stock Deduction (ไม่บล็อกการขายหน้าร้าน)
- **Shifts & Cash Drawer:** ตัดการเปิด/ปิดกะ และลิ้นชักเก็บเงิน เพื่อความสะดวกของผู้สูงอายุ
- **Multi-Branch:** ไม่รองรับ Multi-tenancy สาขาหลายแห่ง (Single branch operation)

---

## 3. Architecture & Domain Layers

```
┌─────────────────────────────────────────────────────────┐
│                Delivery Layer (Go Fiber v3)             │
│  Routers: /auth, /users, /products, /categories,        │
│           /customers, /sales, /reports, /uploads,       │
│           /sync, /health, /swagger                      │
└───────────────────────────┬─────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                  Usecase Layer                          │
│  - AuthUsecase (Login, PIN login, Refresh Token, JWT)   │
│  - UserUsecase (Admin Manage Users & Cashier PINs)      │
│  - CatalogUsecase (Products, Categories, Barcodes)      │
│  - CustomerUsecase (Members, Credit, Payments, Aging)   │
│  - SaleUsecase (Cash/Credit Checkout, Change, Void)     │
│  - ReportUsecase (Sales Summary, Best Sellers)          │
│  - SyncUsecase (Offline Pull & Batch Push)              │
│  - UploadHandler (Image Compression <= 50KB)            │
└───────────────────────────┬─────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                  Domain Layer                           │
│  - Entities: User, Product, Category, Customer,         │
│              CreditAccount, CreditTransaction,          │
│              Sale, SaleItem, Receipt                    │
│  - Financial Math: shopspring/decimal                   │
│  - Statuses: Sale (COMPLETED, VOID), Credit (OK, FROZEN)│
└───────────────────────────┬─────────────────────────────┘
                            ▲
                            │ implements
┌─────────────────────────────────────────────────────────┐
│               Repository Layer (GORM)                   │
│  - Database: SQLite (default) / PostgreSQL support      │
│  - Simplified schema without shift/PO/inventory tables  │
└─────────────────────────────────────────────────────────┘
```

---

## 4. Database Schema

### ตารางในฐานข้อมูล (Active Tables):

1. **`users`**: ผู้ใช้งาน, รหัสผ่าน bcrypt, PIN hash, สิทธิ์ (`ADMIN`, `CASHIER`), สถานะใช้งาน
2. **`categories`**: หมวดหมู่สินค้า
3. **`products`**: ข้อมูลสินค้า (รหัส, ชื่อ, ราคาขาย, ต้นทุน, บาร์โค้ด, ลิงก์รูปภาพ WebP <= 50KB, สถานะใช้งาน)
4. **`customers`**: ข้อมูลลูกค้า/สมาชิก, เบอร์โทร, อีเมล, สถานะสมาชิก, แต้มสะสม (`points`), เวอร์ชัน (`version`)
5. **`credit_accounts`**: บัญชีวงเงินเชื่อของลูกค้า, วงเงินเครดิต (`credit_limit`), ยอดหนี้คงค้าง (`outstanding_balance`), สถานะ (`OK`, `FROZEN`)
6. **`credit_transactions`**: ประวัติการซื้อเชื่อ (`CHARGE`) และประวัติการชำระหนี้ (`PAYMENT`, `ADJUSTMENT`)
7. **`sales`**: บิลขาย (เลขที่บิล, วันเวลา, ยอดรวม, วิธีชำระ `CASH`/`CREDIT`, เงินที่รับมา, เงินทอน, รหัสลูกค้า, สถานะ `COMPLETED`/`VOID`, แคชเชียร์)
8. **`sale_items`**: รายการสินค้าในบิล (สินค้า, จำนวน, ราคาขายต่อหน่วย, ต้นทุนต่อหน่วย, ยอดรวม)
9. **`receipts`**: ข้อมูลใบเสร็จรับเงินสำหรับพิมพ์หรือเรียกดูย้อนหลัง
10. **`system_configs`**: ข้อมูลร้านค้า (ชื่อร้าน, ที่อยู่, เบอร์โทร, เลขประจำตัวผู้เสียภาษี สำหรับหัวบิล)

---

## 5. API Endpoint Specifications

**Base URL:** `/api/v1`  
**Interactive Swagger UI:** `/swagger` (เอกสาร OpenAPI 2.0 ครบถ้วนพร้อมทดสอบ)

### 5.1 ระบบตรวจสอบสถานะระบบ (Health)

- **`GET /api/v1/health`**
  - **สิทธิ์:** Public
  - **คำอธิบาย:** ตรวจสอบความพร้อมใช้งานของ API
  - **Response 200:**
    ```json
    {
      "status": "ok",
      "timestamp": "2026-11-02T07:30:00Z"
    }
    ```

### 5.2 ระบบยืนยันตัวตน (Authentication)

- **`POST /api/v1/auth/login`**
  - **สิทธิ์:** Public
  - **Request Body:** `{ "username": "admin", "password": "password" }`
  - **Response 200:** `{ "success": true, "data": { "token": "...", "user": { ... } } }`
  - **หมายเหตุ:** ส่ง Set-Cookie สำหรับ Refresh Token (`refresh_token`, HttpOnly)
- **`POST /api/v1/auth/pin-login`**
  - **สิทธิ์:** Public (แคชเชียร์หน้าร้าน)
  - **Request Body:** `{ "pin": "1234" }`
  - **Response 200:** เข้าสู่ระบบรวดเร็วด้วย PIN 4-6 หลัก
- **`POST /api/v1/auth/refresh`**
  - **สิทธิ์:** Public (รองรับ Refresh Token จาก Cookie หรือ Request Body `{ "refresh_token": "..." }`)
  - **Response 200:** ออก Access Token ตัวใหม่และต่ออายุ Refresh Token อัตโนมัติ
- **`POST /api/v1/auth/logout`**
  - **สิทธิ์:** Public / Authenticated
  - **คำอธิบาย:** ล้าง HttpOnly Cookie
- **`GET /api/v1/auth/me`**
  - **สิทธิ์:** Bearer Auth
  - **Response 200:** ข้อมูลผู้ใช้งานปัจจุบันและสิทธิ์

### 5.3 ระบบจัดการผู้ใช้งาน (User Management - Admin Only)

- **`GET /api/v1/users`**: ดึงรายชื่อผู้ใช้งานทั้งหมด
- **`POST /api/v1/users`**: สร้างบัญชีแคชเชียร์ใหม่ พร้อมกำหนด PIN
- **`POST /api/v1/users/:id/reset-pin`**: รีเซ็ตรหัส PIN ของแคชเชียร์

### 5.4 ระบบสินค้าและแคตตาล็อก (Catalog & Products)

- **`GET /api/v1/products`**: รายการสินค้าทั้งหมด (รองรับ query `limit`, `offset`, `search`)
- **`GET /api/v1/products/barcode/:barcode`**: ค้นหาสินค้าอย่างรวดเร็วด้วยการยิงบาร์โค้ด
- **`GET /api/v1/products/:id`**: ข้อมูลรายละเอียดสินค้าตาม UUID
- **`POST /api/v1/products`** _(Admin)_: เพิ่มสินค้าใหม่ลงระบบ
- **`POST /api/v1/products/:id/image`**: อัปโหลดรูปภาพสินค้าโดยตรง ปรับขนาดและบีบอัดเป็น WebP <= 50KB พร้อมบันทึก `image_url` ลงสินค้าให้อัตโนมัติ
- **`GET /api/v1/categories`**: ดึงรายการหมวดหมู่สินค้าทั้งหมด

### 5.5 ระบบอัปโหลดรูปภาพ (Uploads & Image Optimization)

- **`POST /api/v1/uploads/images`**
  - **สิทธิ์:** Bearer Auth
  - **Content-Type:** `multipart/form-data` (Field: `file` หรือ `image`)
  - **ข้อกำหนดรูปภาพ:**
    - รองรับไฟล์: `.jpg`, `.jpeg`, `.png`, `.webp`, `.gif`
    - ขนาดไฟล์อัปโหลดจากต้นทางได้สูงสุด: 10MB
    - ระบบจะทำการบีบอัด WebP และย่อขนาด (Dynamic Quality & Downscale) จนขนาดไฟล์ **ไม่เกิน 50KB (<= 51,200 bytes)**
  - **Response 201:**
    ```json
    {
      "success": true,
      "data": {
        "url": "/uploads/images/xxx.webp",
        "file_name": "xxx.webp",
        "content_type": "image/webp",
        "width": 800,
        "height": 600
      }
    }
    ```

### 5.6 ระบบลูกค้า สมาชิก และวงเงินเชื่อ (Customers & Credit)

- **`POST /api/v1/customers`**: สร้างลูกค้าใหม่ กำหนดสถานะสมาชิก และเปิดวงเงินเชื่อเริ่มต้น (หน่วยสตางค์)
- **`GET /api/v1/customers`**: ดึงรายชื่อลูกค้า (ค้นหาตามชื่อ เบอร์โทร ได้)
- **`GET /api/v1/customers/:id`**: ดูข้อมูลลูกค้า ยอดแต้มสะสม และสถานะบัญชีวงเงินเชื่อ
- **`PUT /api/v1/customers/:id`**: แก้ไขข้อมูลลูกค้า
- **`DELETE /api/v1/customers/:id`**: ลบข้อมูลลูกค้า (Soft Delete)
- **`PUT /api/v1/customers/:id/credit-limit`**: ปรับเปลี่ยนวงเงินเครดิตสูงสุด
- **`POST /api/v1/customers/:id/pay-credit`**: รับชำระหนี้เงินเชื่อ (หักลดยอดหนี้คงค้างและบันทึกประวัติการรับชำระ)
- **`GET /api/v1/customers/:id/credit-history`**: ประวัติความเคลื่อนไหวทางเครดิตทั้งหมดของลูกค้า
- **`GET /api/v1/customers/reports/ar-aging`**: รายงานวิเคราะห์อายุหนี้ (0-30 วัน, 31-60 วัน, 61-90 วัน, 90+ วัน)

### 5.7 ระบบบันทึกการขายและใบเสร็จ (Sales & Receipts)

- **`POST /api/v1/sales`** (Checkout):
  - **Request:**
    ```json
    {
      "customer_id": "optional-customer-uuid",
      "payment_type": "CASH", // หรือ "CREDIT"
      "tendered_amount": 50000, // สตางค์ (กรณีเงินสด)
      "items": [{ "product_id": "uuid", "qty": 2 }]
    }
    ```
  - **พฤติกรรม:**
    - ดึงราคาขายปัจจุบันจากฐานข้อมูลโดยตรง (ป้องกันการโกงราคาจากหน้าร้าน)
    - คำนวณเงินทอนอัตโนมัติหากเป็น `CASH`
    - หากเป็น `CREDIT` ตรวจสอบวงเงินคงเหลือและบันทึกลงบัญชีลูกหนี้
    - สะสมแต้มสมาชิกร้านค้าอัตโนมัติ (1 แต้มต่อทุก 100 บาท)
  - **Response 201:** รายละเอียดบิลขาย เลขที่บิล และข้อมูลใบเสร็จ
- **`GET /api/v1/sales`**: รายการประวัติการขาย
- **`GET /api/v1/sales/:id`**: ข้อมูลบิลขายตาม ID
- **`GET /api/v1/sales/receipt/:receiptNo`**: ข้อมูลสำหรับพิมพ์ใบเสร็จย้อนหลัง
- **`POST /api/v1/sales/:id/void`**: ยกเลิกบิลโดยตรง ไม่ต้องขอสิทธิ์ Admin (หากเป็นบิลเงินเชื่อจะปรับลดยอดหนี้คืนให้อัตโนมัติ)

### 5.8 ระบบรายงานยอดขาย (Reports & Analytics)

- **`GET /api/v1/reports/sales-summary?date=YYYY-MM-DD`**:
  - ยอดขายรวมสุทธิ
  - ต้นทุนขายรวม และกำไรขั้นต้น
  - จำนวนบิลที่สำเร็จ และจำนวนบิลที่ยกเลิก
  - ยอดเงินสดที่ได้รับจริง
- **`GET /api/v1/reports/best-sellers?start_date=...&end_date=...&limit=10`**:
  - อันดับสินค้าขายดี จำนวนที่ขายได้ และยอดขายรวม

### 5.9 ระบบซิงค์ข้อมูลออฟไลน์ (Offline-first Synchronization)

- **`GET /api/v1/sync/changes?since=2026-11-01T00:00:00Z`**:
  - ดึงรายการสินค้าและหมวดหมู่ที่มีการเปลี่ยนแปลงตั้งแต่เวลาที่ระบุ (Delta sync) หรือทั้งหมดเมื่อต่อเน็ตครั้งแรก
- **`POST /api/v1/sync/batch`**:
  - ส่งบิลขายที่ถูกบันทึกหรือยกเลิกระหว่างออฟไลน์ขึ้นเซิร์ฟเวอร์แบบ Batch ประมวลผลแบบ Idempotent ป้องกันการเกิดบิลซ้ำซ้อน
