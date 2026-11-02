# ICE POS API v2 - API Specification Reference

> **Version:** 2.0.0  
> **Base URL:** `/api/v1`  
> **Interactive Swagger UI:** `http://localhost:8080/swagger` (หรือ `/swagger/index.html`)  
> **Authentication:** Bearer JWT Token (`Authorization: Bearer <token>`) & HttpOnly Cookie สำหรับ Refresh Token  
> **Currency Format:** สตางค์ (Satang: 1 บาท = 100 สตางค์) เพื่อความแม่นยำทางคณิตศาสตร์ 100%

---

## สารบัญ (Table of Contents)

1. [ภาพรวมและมาตรฐาน API](#1-ภาพรวมและมาตรฐาน-api)
2. [ระบบยืนยันตัวตน (Authentication)](#2-ระบบยืนยันตัวตน-authentication)
3. [ระบบจัดการผู้ใช้งาน (Users - Admin Only)](#3-ระบบจัดการผู้ใช้งาน-users---admin-only)
4. [ระบบสินค้าและแคตตาล็อก (Products & Categories)](#4-ระบบสินค้าและแคตตาล็อก-products--categories)
5. [ระบบอัปโหลดรูปภาพ (Image Uploads <= 50KB)](#5-ระบบอัปโหลดรูปภาพ-image-uploads--50kb)
6. [ระบบลูกค้า สมาชิก และวงเงินเชื่อ (Customers & Credit)](#6-ระบบลูกค้า-สมาชิก-และวงเงินเชื่อ-customers--credit)
7. [ระบบขายหน้าร้านและการคิดเงิน (Sales & Checkout)](#7-ระบบขายหน้าร้านและการคิดเงิน-sales--checkout)
8. [ระบบรายงานยอดขาย (Reports & Analytics)](#8-ระบบรายงานยอดขาย-reports--analytics)
9. [ระบบซิงค์ข้อมูลออฟไลน์ (Offline-First Sync)](#9-ระบบซิงค์ข้อมูลออฟไลน์-offline-first-sync)

---

## 1. ภาพรวมและมาตรฐาน API

### 1.1 รูปแบบการตอบกลับมาตรฐาน (Standard Response Formats)

**เมื่อสำเร็จ (Success Response):**

```json
{
  "success": true,
  "data": { ... }
}
```

**เมื่อเกิดข้อผิดพลาด (Error Response):**

```json
{
  "error": {
    "code": "BAD_REQUEST",
    "message": "คำอธิบายข้อผิดพลาด"
  }
}
```

---

## 2. ระบบยืนยันตัวตน (Authentication)

### 2.1 เข้าสู่ระบบด้วย Username / Password

- **URL:** `POST /api/v1/auth/login`
- **Request Body:**
  ```json
  {
    "username": "admin",
    "password": "password123"
  }
  ```
- **Response 200 (Success):**
  ```json
  {
    "success": true,
    "data": {
      "token": "eyJhbGciOi...",
      "refresh_token": "eyJhbGciOi...",
      "user": {
        "id": "uuid",
        "username": "admin",
        "display_name": "ผู้ดูแลระบบ",
        "role": "ADMIN"
      }
    }
  }
  ```
- **หมายเหตุ:** เซิร์ฟเวอร์จะแนบ `Set-Cookie: refresh_token=...; HttpOnly; SameSite=Lax` กลับมาด้วยเสมอ

### 2.2 เข้าสู่ระบบด่วนด้วย PIN หน้าร้าน

- **URL:** `POST /api/v1/auth/pin-login`
- **Request Body:**
  ```json
  {
    "pin": "1234"
  }
  ```
- **Response 200:** เช่นเดียวกับ Login ปกติ

### 2.3 ต่ออายุ Token (Refresh Token)

- **URL:** `POST /api/v1/auth/refresh`
- **Request:** อ่าน `refresh_token` จาก Cookie อัตโนมัติ หรือส่งผ่าน JSON Body:
  ```json
  {
    "refresh_token": "eyJhbGciOi..."
  }
  ```
- **Response 200:** ได้รับ Access Token ใหม่พร้อมต่ออายุ Refresh Token

### 2.4 ออกจากระบบ (Logout)

- **URL:** `POST /api/v1/auth/logout`
- **คำอธิบาย:** เคลียร์ HttpOnly Cookie

### 2.5 ตรวจสอบข้อมูลผู้ใช้งานปัจจุบัน

- **URL:** `GET /api/v1/auth/me`
- **Header:** `Authorization: Bearer <access_token>`

---

## 3. ระบบจัดการผู้ใช้งาน (Users - Admin Only)

- **`GET /api/v1/users`**: ดึงรายชื่อผู้ใช้งานทั้งหมด
- **`POST /api/v1/users`**: สร้างบัญชีแคชเชียร์ใหม่
  ```json
  {
    "username": "cashier2",
    "password": "password123",
    "pin": "5678",
    "display_name": "แคชเชียร์ 2"
  }
  ```
- **`POST /api/v1/users/:id/reset-pin`**: เปลี่ยน/รีเซ็ตรหัส PIN ของแคชเชียร์

---

## 4. ระบบสินค้าและแคตตาล็อก (Products & Categories)

### 4.1 รายการสินค้าทั้งหมด

- **URL:** `GET /api/v1/products?limit=50&offset=0&search=น้ำดื่ม`
- **Query Parameters:**
  - `search`: ค้นหาตามชื่อสินค้า รหัสสินค้า หรือบาร์โค้ด (LIKE wildcard)
  - `barcode`: ค้นหาตรงตามบาร์โค้ด (Exact Match)
  - `category_id`: กรองตามหมวดหมู่
  - `limit`: จำนวนรายการต่อหน้า (ค่าเริ่มต้น 50)
  - `offset`: ลำดับเริ่มต้น
- **Response 200:** รายการสินค้า ราคาขาย ต้นทุน บาร์โค้ด และรูปภาพ

### 4.2 ค้นหาสินค้าด้วยบาร์โค้ด (สำหรับการสแกนด้วยกล้อง)

- **URL:** `GET /api/v1/products/barcode/:barcode`
- **คำอธิบาย:** ค้นหาสินค้าอย่างรวดเร็วด้วยรหัสบาร์โค้ด (Exact Match เฉพาะบาร์โค้ด) รองรับการถอดรหัส URL และตัดช่องว่าง (Trim Whitespace)
- **Response 200 (Success):**
  ```json
  {
    "success": true,
    "data": {
      "id": "uuid",
      "code": "P001",
      "barcode": "885000000001",
      "name": "น้ำดื่ม 600ml",
      "price": 1000,
      "cost_price": 600,
      "category_id": "category-uuid",
      "category": { "id": "category-uuid", "name": "สินค้าทั่วไป" },
      "image_url": "",
      "is_active": true
    }
  }
  ```
- **Response 404 (Not Found):**
  ```json
  {
    "error": {
      "code": "NOT_FOUND",
      "message": "Record not found"
    }
  }
  ```

### 4.3 สร้างสินค้าใหม่ (Admin)

- **URL:** `POST /api/v1/products`
- **Header:** `Authorization: Bearer <admin_token>`
- **คำอธิบาย:** เพิ่มสินค้าใหม่ หากระบุ `barcode` จะตรวจสอบความซ้ำซ้อนในระบบ หากซ้ำจะคืนค่า 400 Bad Request
- **Request Body:**
  ```json
  {
    "code": "P003",
    "barcode": "885000000003",
    "name": "ชาเขียวพร้อมดื่ม",
    "price": 2000,
    "cost_price": 1400,
    "category_id": "category-uuid",
    "image_url": "/uploads/images/example.webp"
  }
  ```

### 4.4 แก้ไขข้อมูลสินค้าและลงทะเบียนบาร์โค้ด (Admin)

- **URL:** `PUT /api/v1/products/:id`
- **Header:** `Authorization: Bearer <admin_token>`
- **คำอธิบาย:** แก้ไขข้อมูลสินค้า เช่น ชื่อ ราคา หรือผูกรหัสบาร์โค้ดใหม่ โดยระบบจะตรวจสอบไม่ให้บาร์โค้ดซ้ำกับสินค้าอื่น
- **Request Body:**
  ```json
  {
    "name": "ชาเขียวพร้อมดื่ม รสหวานน้อย",
    "barcode": "885000000003",
    "price": 2200
  }
  ```
- **Response 200 (Success):** ข้อมูลสินค้าที่ได้รับการปรับปรุง

### 4.5 อัปโหลดรูปภาพผูกกับสินค้าโดยตรง

- **URL:** `POST /api/v1/products/:id/image`
- **Content-Type:** `multipart/form-data` (Field: `file`)
- **คำอธิบาย:** บีบอัดภาพเป็น WebP <= 50KB และบันทึก `image_url` ในตัวสินค้า

### 4.6 ดึงหมวดหมู่สินค้า

- **URL:** `GET /api/v1/categories`

---

## 5. ระบบอัปโหลดรูปภาพ (Image Uploads <= 50KB)

- **URL:** `POST /api/v1/uploads/images`
- **Content-Type:** `multipart/form-data`
- **Form Field:** `file` หรือ `image`
- **เงื่อนไขและการบีบอัด:**
  - รองรับไฟล์ต้นทาง: `.jpg`, `.jpeg`, `.png`, `.webp`, `.gif` ขนาดสูงสุด 10MB
  - ประมวลผลผ่าน `imageutil`: ปรับลด Quality และ Dynamic Scaling จนได้ไฟล์ WebP ขนาด **ไม่เกิน 50KB** (<= 51,200 bytes)
  - เก็บลง Local Storage (`/uploads/images/...`) หรือ ImageKit CDN ตาม `STORAGE_DRIVER` ในคอนฟิก
- **Response 201:**
  ```json
  {
    "success": true,
    "data": {
      "url": "/uploads/images/81751b43-be4b-4cbb-8292-f05ab6c5257c.webp",
      "file_name": "81751b43-be4b-4cbb-8292-f05ab6c5257c.webp",
      "content_type": "image/webp",
      "width": 800,
      "height": 600
    }
  }
  ```

---

## 6. ระบบลูกค้า สมาชิก และวงเงินเชื่อ (Customers & Credit)

### 6.1 เพิ่มข้อมูลลูกค้าและเปิดวงเงินเครดิต

- **URL:** `POST /api/v1/customers`
- **Request Body:**
  ```json
  {
    "name": "สมชาย ใจดี",
    "phone": "0812345678",
    "email": "somchai@example.com",
    "is_member": true,
    "initial_credit_limit": 500000
  }
  ```
  _(หน่วยสตางค์: 500,000 สตางค์ = วงเงิน 5,000 บาท)_

### 6.2 ดูข้อมูลลูกค้าและสถานะวงเงิน

- **URL:** `GET /api/v1/customers/:id`
- **Response 200:** ข้อมูลลูกค้า, แต้มสะสม `points`, `credit_limit`, `outstanding_balance` (ยอดหนี้), `credit_status` (`OK` / `FROZEN`)

### 6.3 ปรับเปลี่ยนวงเงินเครดิต

- **URL:** `PUT /api/v1/customers/:id/credit-limit`
- **Request Body:**
  ```json
  {
    "credit_limit": 1000000
  }
  ```

### 6.4 รับชำระหนี้เงินเชื่อ

- **URL:** `POST /api/v1/customers/:id/pay-credit`
- **Request Body:**
  ```json
  {
    "amount": 200000,
    "note": "ชำระเงินสดหน้าร้าน"
  }
  ```

### 6.5 รายงานวิเคราะห์อายุหนี้ (AR Aging Report)

- **URL:** `GET /api/v1/customers/reports/ar-aging`
- **Response 200:** สรุปยอดหนี้แยกตามช่วงอายุ:
  - `current` (0-30 วัน)
  - `days_31_60` (31-60 วัน)
  - `days_61_90` (61-90 วัน)
  - `days_over_90` (เกิน 90 วัน)

---

## 7. ระบบขายหน้าร้านและการคิดเงิน (Sales & Checkout)

### 7.1 บันทึกการขาย (Checkout)

- **URL:** `POST /api/v1/sales`
- **Request Body (จ่ายเงินสด):**
  ```json
  {
    "payment_type": "CASH",
    "tendered_amount": 10000,
    "items": [{ "product_id": "product-uuid-1", "qty": 2 }]
  }
  ```
- **Request Body (ซื้อเงินเชื่อ / สมาชิก):**
  ```json
  {
    "customer_id": "customer-uuid",
    "payment_type": "CREDIT",
    "items": [{ "product_id": "product-uuid-1", "qty": 5 }]
  }
  ```
- **พฤติกรรมของระบบ:**
  1. ดึงราคาสินค้าจาก Database ป้องกันการปลอมแปลงราคาจาก Client
  2. คำนวณเงินทอนให้อัตโนมัติในกรณี `CASH`
  3. ตรวจสอบวงเงินคงเหลือและบันทึกหนี้ในกรณี `CREDIT`
  4. เพิ่มแต้มสะสมให้ลูกค้าสมาชิกอัตโนมัติ (1 แต้มต่อยอดซื้อทุก 100 บาท)
  5. บันทึกต้นทุนสินค้า (`cost_price`) ณ เวลาขาย สำหรับออกรายงานกำไรขั้นต้น

### 7.2 ยกเลิกบิลขาย (Void Sale)

- **URL:** `POST /api/v1/sales/:id/void`
- **Request Body:**
  ```json
  {
    "reason": "ลูกค้าเปลี่ยนใจ / คีย์สินค้าผิด"
  }
  ```
- **คำอธิบาย:** แคชเชียร์สามารถกดยกเลิกได้โดยตรง บิลจะเปลี่ยนสถานะเป็น `VOID` และตัดยอดออกจากรายงานขาย หากเป็นบิลเงินเชื่อระบบจะปรับลดยอดหนี้คืนให้อัตโนมัติ

### 7.3 ดึงข้อมูลใบเสร็จย้อนหลัง

- **URL:** `GET /api/v1/sales/receipt/:receiptNo`

---

## 8. ระบบรายงานยอดขาย (Reports & Analytics)

### 8.1 สรุปยอดขายประจำวันแบบไดนามิก

- **URL:** `GET /api/v1/reports/sales-summary?date=2026-11-02`
- **Response 200:**
  ```json
  {
    "success": true,
    "data": {
      "date": "2026-11-02",
      "total_sales": 1540000,
      "total_cost": 920000,
      "gross_profit": 620000,
      "bill_count": 48,
      "void_count": 2,
      "total_cash_collected": 1200000
    }
  }
  ```

### 8.2 รายงานสินค้าขายดี

- **URL:** `GET /api/v1/reports/best-sellers?start_date=2026-11-01&end_date=2026-11-02&limit=10`

---

## 9. ระบบซิงค์ข้อมูลออฟไลน์ (Offline-First Sync)

### 9.1 ดึงข้อมูลสินค้าที่อัปเดตลงเครื่อง POS

- **URL:** `GET /api/v1/sync/changes?since=2026-11-01T00:00:00Z`
- **คำอธิบาย:** คืนรายการสินค้าและหมวดหมู่ที่มีการสร้างหรือแก้ไขหลังจากวันเวลาที่ระบุ หากไม่ส่ง `since` จะคืนข้อมูลแคตตาล็อกทั้งหมด

### 9.2 ส่งชุดบิลขายออฟไลน์ขึ้นเซิร์ฟเวอร์ (Batch Push)

- **URL:** `POST /api/v1/sync/batch`
- **Request Body:**
  ```json
  {
    "sales": [
      {
        "id": "sale-uuid-generated-offline",
        "sale_no": "S20261102-0001",
        "occurred_at": "2026-11-02T07:15:00Z",
        "payment_type": "CASH",
        "tendered_amount": 50000,
        "change_amount": 10000,
        "grand_total": 40000,
        "status": "COMPLETED",
        "items": [
          {
            "product_id": "prod-uuid",
            "qty": 2,
            "unit_price": 20000,
            "net_total": 40000
          }
        ]
      }
    ]
  }
  ```
- **คำอธิบาย:** บันทึกข้อมูลแบบ Idempotent ป้องกันบิลซ้ำซ้อนอย่างสมบูรณ์
