# Design Document: ICE POS API v2 (Simplified Single-Branch Cash POS)

**Date:** 2026-11-01  
**Project:** `ice-pos-api-v2`  
**Target User:** ผู้สูงอายุ / ร้านค้าสาขาเดียวที่ต้องการความเรียบง่ายสูงสุด  
**Core Purpose:** ค้นหาราคาสินค้า, คิดเงิน/บันทึกการขายเฉพาะเงินสด, คำนวณเงินทอน, ยกเลิกบิลได้ง่าย, และดูประวัติยอดขายย้อนหลังพร้อมรายงานสินค้าขายดีและต้นทุน

---

## 1. Intent & The Why

ระบบเดิม (`ice-pos-api` v1) มีความซับซ้อนสูง ทั้งการจัดการสาขา, คลังสินค้า/จัดซื้อ, ระบบกะและลิ้นชัก, และช่องทางชำระเงินที่หลากหลาย

สำหรับ **v2** กลุ่มผู้ใช้งานหลักคือ **ผู้สูงอายุ** ซึ่งต้องการระบบที่:

1. **ไม่ยุ่งยาก ไม่ซับซ้อน:** เปิดระบบแล้วพร้อมขายทันที ไม่ต้องเปิดกะ ไม่ต้องนับเงินตั้งต้น
2. **ไม่สะดุดเรื่องสต็อก:** จัดการสต็อกผ่านช่องทางภายนอกอยู่แล้ว หน้าร้านต้องการขายได้ตลอดเวลา ไม่มีการบล็อกขายเพราะสต็อกหมด
3. **จ่ายเงินสดและคำนวณเงินทอน:** รับเงินสดมาเท่าไหร่ ระบบคำนวณเงินทอนให้อัตโนมัติ ป้องกันการคิดเลขผิด
4. **ยกเลิกบิลผิดพลาดได้ง่าย (Void Sale):** แคชเชียร์สามารถกดยกเลิกบิลที่ยิงผิดหรือคิดผิดได้เอง ไม่ต้องรอสิทธิ์ Admin
5. **ดูสรุปและสินค้าขายดีได้:** ตรวจสอบยอดขายประจำวันและดูสินค้าขายดีได้แบบ Dynamic ตามวันที่เลือก ไม่ต้องมีขั้นตอน "ปิดยอดสิ้นวัน"

---

## 2. Goals & Non-Goals

### Goals

- **Clean Architecture Foundation:** ตั้งต้นโครงสร้างจาก Clean Architecture เดิมของ v1 (Go Fiber v3 + GORM / SQLite & PostgreSQL)
- **Simplified Cash-Only Checkout:**
  - ค้นหาสินค้า / ยิงบาร์โค้ด
  - ดึงราคากลางที่ถูกต้อง (Server-authoritative pricing)
  - คำนวณเงินทุกประเภทด้วย `github.com/shopspring/decimal` เพื่อความถูกต้องแม่นยำสูงสุด
  - รับเงินสด (`tendered_amount`) และคำนวณเงินทอน (`change_amount`)
  - บันทึกการขาย (Sale + Sale Items) ออกเลขที่ใบเสร็จทันที
  - บันทึกต้นทุนสินค้า (`cost_price`) ณ เวลาขาย เพื่อนำไปออกรายงานกำไรขั้นต้น (แสดงต้นทุนเฉพาะในรายงาน ไม่แสดงบนใบเสร็จ)
- **Void Sale (ยกเลิกบิล):** รองรับ `POST /sales/:id/void` โดยแคชเชียร์สามารถกดยกเลิกได้เอง บิลที่ยกเลิกจะไม่นำไปคำนวณในยอดขายรวม
- **Dynamic Daily Reports:** สรุปยอดขายประจำวันและรายงานสินค้าขายดีแบบ On-the-fly ตามช่วงวันที่ ไม่ต้องทำ Day-Close
- **Single Branch Operation:** สาขาเดียวเป็นค่าเริ่มต้น (ไม่ต้องส่ง `branch_id`)
- **Zero Shift Friction:** ไม่ต้องเปิด/ปิดกะ ไม่ต้องมีลิ้นชักเก็บเงิน
- **Lightweight Offline Sync Engine:**
  - `GET /api/v1/sync/changes?since=...`: ดาวน์โหลดสินค้า/หมวดหมู่ลงแท็บเล็ตหน้าร้าน เพื่อให้คิดเงินออฟไลน์ได้เวลาเน็ตหลุด
  - `POST /api/v1/sync/batch`: ส่งชุดข้อมูลบิลขาย (ทั้ง `COMPLETED` และ `VOID`) ขึ้นเซิร์ฟเวอร์แบบ Idempotent เมื่อต่อเน็ตได้
- **Preserved Core Features:**
  - Login ด้วย Username/Password และ PIN (JWT auth)
  - Admin API สำหรับจัดการผู้ใช้งานและรีเซ็ตรหัส PIN (`/users`)
  - จัดการสินค้า หมวดหมู่ บาร์โค้ด และรูปภาพสินค้า (Catalog, Categories, Products, Barcodes, Uploads)

### Non-Goals (Features Cut / Removed)

- **Payment Methods:** ตัด PromptPay QR, บัตรเครดิต, บัตรเดบิต, เงินโอน และลูกหนี้เงินเชื่อ (เหลือ Cash เท่านั้น)
- **Inventory & Procurement:** ตัด Suppliers, Purchase Orders (PO), Goods Receipts (GRN), Stock Adjustments, Stock Transfers, Stock Deduction
- **Shifts & Cash Drawer:** ตัด Shift Open, Shift Close, Active Shift, Drawer Management
- **Multi-Branch:** ตัดการจัดการสาขาหลายแห่ง
- **Offline Sync Engine:** ตัด Delta Sync & Push Batch ออก

---

## 3. Architecture & Domain Changes

```
┌─────────────────────────────────────────────────────────┐
│                Delivery Layer (Go Fiber v3)             │
│  Routers: /auth, /users, /products, /categories,        │
│           /sales, /reports, /uploads, /health           │
└───────────────────────────┬─────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                  Usecase Layer                          │
│  - AuthUsecase (Login, PIN login, JWT)                  │
│  - UserUsecase (Admin Manage Users & Cashier PINs)      │
│  - CatalogUsecase (Products, Categories, Barcodes)      │
│  - SaleUsecase (Cash Checkout, Change, Void, Receipts)  │
│  - ReportUsecase (Dynamic Sales Summary, Best Sellers)  │
│  - UploadUsecase (Product Images)                       │
└───────────────────────────┬─────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                  Domain Layer                           │
│  - Entities: User, Product, Category, Sale, SaleItem,   │
│              Receipt                                    │
│  - Money Calculation: shopspring/decimal                │
│  - Sale Status: COMPLETED, VOID                         │
│  - No Inventory / Shift / PO / Supplier Entities        │
└───────────────────────────┬─────────────────────────────┘
                            ▲
                            │ implements
┌─────────────────────────────────────────────────────────┐
│               Repository Layer (GORM)                   │
│  - Database: SQLite (default) / PostgreSQL support      │
│  - Clean schema without inventory/shift/branch tables   │
└─────────────────────────────────────────────────────────┘
```

---

## 4. Database Schema Simplification

### ตารางที่เก็บไว้ (Kept Tables):

1. `users`: บัญชีผู้ใช้, รหัสผ่าน hash, PIN hash, สิทธิ์ (`ADMIN`, `CASHIER`)
2. `categories`: หมวดหมู่สินค้า
3. `products`: สินค้า (รหัส, ชื่อ, ราคาขาย, ต้นทุน, บาร์โค้ด, รูปภาพ, สถานะใช้งาน)
4. `sales`: บิลขาย (เลขที่บิล, วันเวลา, ยอดรวม, เงินที่รับมา, เงินทอน, สถานะ `COMPLETED`/`VOID`, แคชเชียร์)
5. `sale_items`: รายการสินค้าในบิล (สินค้า, จำนวน, ราคาขายต่อหน่วย, ต้นทุนต่อหน่วย, ยอดรวม)
6. `receipts`: ข้อมูลใบเสร็จรับเงินสำหรับเรียกดูย้อนหลัง
7. `system_configs`: ข้อมูลร้านค้า (ชื่อร้าน, ที่อยู่, เบอร์โทร, เลขประจำตัวผู้เสียภาษี สำหรับหัวบิล)

### ตารางที่ตัดออก (Removed Tables):

- `shifts`, `cash_drawers`, `cash_drawer_transactions`
- `inventories`, `stock_movements`, `stock_adjustments`, `stock_transfers`
- `suppliers`, `purchase_orders`, `purchase_order_items`, `goods_receipts`, `goods_receipt_items`
- `customer_credit_accounts`, `customer_credit_transactions`, `ar_invoices`
- `branches`, `devices`, `sync_logs`
