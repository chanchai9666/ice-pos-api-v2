# Design Document: Product Category Management (ส่วนจัดการหมวดหมู่สินค้า)

**Date:** 2026-11-02  
**Status:** Approved by Reviewer  
**Target:** `ice-pos-api-v2`

---

## 1. Intent & Why

ระบบ `ice-pos-api-v2` ปัจจุบันมีโมเดล `Category` และเอนด์พอยต์อ่านรายการ `GET /api/v1/categories` เบื้องต้น แต่ยังขาดระบบจัดการหมวดหมู่สินค้าแบบครบวงจร (CRUD) ซึ่งจำเป็นสำหรับการจัดหมวดหมู่สินค้าในระบบขายหน้าร้าน (POS) ทำให้ผู้ดูแลระบบ (Admin) ไม่สามารถเพิ่ม หมวดหมู่ ปรับปรุง หรือลบหมวดหมู่สินค้าผ่านระบบ API ได้

เอกสารฉบับนี้กำหนดการออกแบบระบบจัดการหมวดหมู่สินค้าแบบชั้นเดียว (Flat category structure) ที่กระชับ ปลอดภัย สอดคล้องกับสถาปัตยกรรม Clean Architecture ของระบบ และสอดรับกับแนวคิด "Public list for fast POS, protected write" เดิม

---

## 2. Goals & Scope

### Goals

1. **ครบวงจร REST API สำหรับ Categories:**
   - `GET /api/v1/categories`: เรียกดูรายการหมวดหมู่ทั้งหมด เรียงตาม `sort_order ASC, name ASC` (Public: เพื่อความรวดเร็วในการโหลดข้อมูลหน้าจอ POS)
   - `GET /api/v1/categories/:id`: เรียกดูข้อมูลหมวดหมู่รายตัวตาม ID (Public)
   - `POST /api/v1/categories`: สร้างหมวดหมู่ใหม่ (เฉพาะ `Admin` ผ่าน JWT Auth)
   - `PUT /api/v1/categories/:id`: แก้ไขข้อมูลหมวดหมู่ (ชื่อ, ลำดับการแสดงผล) (เฉพาะ `Admin` ผ่าน JWT Auth)
   - `DELETE /api/v1/categories/:id`: ลบหมวดหมู่สินค้า (เฉพาะ `Admin` ผ่าน JWT Auth)
2. **พฤติกรรมเมื่อลบหมวดหมู่ (Deletion Behavior):**
   - เมื่อทำการลบหมวดหมู่ ระบบจะอนุญาตให้ลบได้ทันที โดยทำการปลดหมวดหมู่ (`category_id = ""` หรือ `NULL`) ออกจากสินค้าทั้งหมดที่เคยผูกกับหมวดหมู่นั้นโดยอัตโนมัติภายใน Database Transaction เดียวกัน (`tx.Transaction`)
3. **การตรวจสอบความถูกต้องของข้อมูล (Validation):**
   - `name`: จำเป็นต้องระบุ (Required), ตัดช่องว่างหัวท้าย (Trimmed), ความยาวระหว่าง 1 - 100 ตัวอักษร
   - `name` ต้องไม่ซ้ำกับหมวดหมู่อื่น (Case-insensitive unique check: `LOWER(TRIM(name))`)
   - `sort_order`: จำนวนเต็ม (Integer) ค่าเริ่มต้น 0
4. **ความปลอดภัยและการกำหนดสิทธิ์ (RBAC):**
   - การเขียนข้อมูล (`POST`, `PUT`, `DELETE`) ต้องผ่าน `AuthMiddleware(rc.JWTService)` และ `RequireRoles(domain.RoleAdmin)`
   - การอ่านข้อมูล (`GET`, `GET /:id`) เป็น Public เช่นเดียวกับ `GET /products` เพื่อความเร็วสูงสุดของเครื่อง POS
5. **ความสมบูรณ์ของเอกสารและการทดสอบ:**
   - Swagger / OpenAPI 2.0 annotations อัปเดตครบทุกเอนด์พอยต์
   - E2E Integration Tests สำหรับ Category CRUD ครอบคลุมทั้ง Positive และ Negative cases

### Non-Goals

- การทำหมวดหมู่ย่อยหลายชั้น (Hierarchical / Parent-Child subcategories) – ระบบใช้แบบชั้นเดียว (Flat) ตามความต้องการของผู้ใช้งาน
- การอัปโหลดรูปภาพประจำหมวดหมู่ – ปัจจุบันหมวดหมู่เน้นการแสดงผลด้วยชื่อและลำดับบนหน้าจอ POS

### Anti-Goals

- **ห้าม Cascade Delete ตัวสินค้าเด็ดขาด:** การลบหมวดหมู่ต้องไม่ลบแถวสินค้า (`Product`) หรือส่งผลกระทบต่อรายการขายในอดีต (`SaleItem`) สินค้าที่เคยอยู่ในหมวดหมู่ที่ถูกลบจะต้องคงอยู่ในระบบ เพียงแต่ไม่มีหมวดหมู่ระบุ

---

## 3. Data Models & Interface Contracts

### 3.1 Domain Model (`internal/domain/catalog.go`)

```go
type Category struct {
    ID        string    `json:"id" gorm:"primaryKey;size:36"`
    Name      string    `json:"name" gorm:"size:100;not null;uniqueIndex"`
    SortOrder int       `json:"sort_order" gorm:"default:0;index"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 3.2 Repository Interface (`CatalogRepository`)

```go
type CatalogRepository interface {
    // Existing product methods...
    GetProductByID(ctx context.Context, id string) (*Product, error)
    // ...

    // Category methods
    ListCategories(ctx context.Context) ([]Category, error)
    GetCategoryByID(ctx context.Context, id string) (*Category, error)
    GetCategoryByName(ctx context.Context, name string) (*Category, error)
    CreateCategory(ctx context.Context, category *Category) error
    UpdateCategory(ctx context.Context, category *Category) error
    DeleteCategory(ctx context.Context, id string) error // Atomic: unlinks products and deletes category
}
```

### 3.3 Usecase Interface (`CatalogUsecase`)

```go
type CreateCategoryInput struct {
    Name      string `json:"name" validate:"required,min=1,max=100"`
    SortOrder int    `json:"sort_order"`
}

type UpdateCategoryInput struct {
    Name      string `json:"name" validate:"required,min=1,max=100"`
    SortOrder *int   `json:"sort_order"` // Pointer: allows distinction between 0 and omitted
}

type CatalogUsecase interface {
    // Product methods...

    // Category methods
    ListCategories(ctx context.Context) ([]domain.Category, error)
    GetCategoryByID(ctx context.Context, id string) (*domain.Category, error)
    CreateCategory(ctx context.Context, input CreateCategoryInput) (*domain.Category, error)
    UpdateCategory(ctx context.Context, id string, input UpdateCategoryInput) (*domain.Category, error)
    DeleteCategory(ctx context.Context, id string) error
}
```

### 3.4 HTTP Endpoints & Payloads

1. **`GET /api/v1/categories`**
   - Auth: Public
   - Response `200 OK`:
     ```json
     {
       "success": true,
       "data": [
         {
           "id": "uuid",
           "name": "เครื่องดื่ม",
           "sort_order": 1,
           "created_at": "...",
           "updated_at": "..."
         }
       ]
     }
     ```

2. **`GET /api/v1/categories/:id`**
   - Auth: Public
   - Response `200 OK`:
     ```json
     {
       "success": true,
       "data": {
         "id": "uuid",
         "name": "เครื่องดื่ม",
         "sort_order": 1,
         "created_at": "...",
         "updated_at": "..."
       }
     }
     ```
   - Errors: `404 Not Found` if id does not exist.

3. **`POST /api/v1/categories`**
   - Auth: Bearer Token (`Admin` Only)
   - Request Body:
     ```json
     {
       "name": "เครื่องดื่ม",
       "sort_order": 1
     }
     ```
   - Response `201 Created`:
     ```json
     {
       "success": true,
       "message": "Category created successfully",
       "data": { ... }
     }
     ```
   - Errors: `400 Bad Request` (validation failed), `409 Conflict` (category name already exists).

4. **`PUT /api/v1/categories/:id`**
   - Auth: Bearer Token (`Admin` Only)
   - Request Body:
     ```json
     {
       "name": "เครื่องดื่มเย็น",
       "sort_order": 2
     }
     ```
   - Response `200 OK`:
     ```json
     {
       "success": true,
       "message": "Category updated successfully",
       "data": { ... }
     }
     ```
   - Errors: `404 Not Found`, `400 Bad Request`, `409 Conflict` (name duplicate with another category).

5. **`DELETE /api/v1/categories/:id`**
   - Auth: Bearer Token (`Admin` Only)
   - Response `200 OK`:
     ```json
     {
       "success": true,
       "message": "Category deleted successfully"
     }
     ```
   - Errors: `404 Not Found`

---

## 4. Business Logic & Edge Cases

1. **Category Name Duplication:**
   - ตรวจสอบชื่อซ้ำแบบ case-insensitive (`LOWER(TRIM(name))`) ทั้งตอนสร้างและแก้ไข
   - กรณีแก้ไขหมวดหมู่เดิมด้วยชื่อเดิมของตัวเอง จะไม่แจ้งเตือนว่าชื่อซ้ำ
2. **Atomicity of Deletion:**
   - ใน `catalogRepository.DeleteCategory` ครอบด้วย `db.Transaction(...)`:
     - ปรับปรุงตาราง `products` ที่มี `category_id = :id` ให้เป็นค่าว่าง (`category_id = ''`)
     - ลบข้อมูลในตาราง `categories` ที่ตรงกับ `:id`
     - Commit transaction
3. **Partial vs Full Update:**
   - `sort_order` ใช้ `*int` เพื่อให้ส่งค่า `0` ได้อย่างชัดเจน หาก client ไม่ส่งฟิลด์นี้มา จะคงค่าเดิมของหมวดหมู่นั้นไว้

---

## 5. What's Left to the Builder

- การสร้าง DTO structs ใน `internal/delivery/http/handler/`
- การอัปเดตรหัส Swagger annotations และรัน `make swag`
- การเขียน E2E Test Suite ใน `tests/e2e_category_test.go`
