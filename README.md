# ICE POS API v2 🚀

[![Go Version](https://img.shields.io/badge/Go-1.22%2B%20%2F%201.26-00ADD8?style=flat&logo=go)](https://golang.org)
[![Fiber Framework](https://img.shields.io/badge/Fiber-v3.0.0-00ACD7?style=flat&logo=fastapi)](https://gofiber.io)
[![Swagger Docs](https://img.shields.io/badge/Swagger-OpenAPI%202.0-85EA2D?style=flat&logo=swagger)](http://localhost:8080/swagger)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

ระบบ Backend RESTful API สำหรับระบบขายหน้าร้าน (Point of Sale) ประสิทธิภาพสูง ออกแบบและพัฒนาด้วยสถาปัตยกรรม Clean Architecture / Layered Architecture ในภาษา **Go (Fiber v3)** รองรับการทำงานแบบ **Offline-First Synchronization**, การจดจำและค้นหาสินค้าด้วย **Vision AI (CLIP 512-d Vectors)**, ระบบสมาชิกและวงเงินเชื่อ (Customer Credit), การประมวลผลรูปภาพอัตโนมัติ (WebP <= 50KB), พร้อมการคำนวณทางการเงินระดับ **สตางค์ (Satang)** ที่แม่นยำ 100%

---

## 📋 สารบัญ (Table of Contents)

- [จุดเด่นและฟีเจอร์หลัก (Key Features)](#-จุดเด่นและฟีเจอร์หลัก-key-features)
- [สถาปัตยกรรมระบบ (Project Architecture)](#-สถาปัตยกรรมระบบ-project-architecture)
- [เทคโนโลยีที่ใช้ (Tech Stack)](#-เทคโนโลยีที่ใช้-tech-stack)
- [ความต้องการเบื้องต้น (Prerequisites)](#-ความต้องการเบื้องต้น-prerequisites)
- [การตั้งค่า Environment Variables](#-การตั้งค่า-environment-variables)
- [ขั้นตอนการติดตั้งและเริ่มใช้งาน (Getting Started)](#-ขั้นตอนการติดตั้งและเริ่มใช้งาน-getting-started)
- [การใช้งาน Make Commands](#-การใช้งาน-make-commands)
- [ระบบ Vision AI (Visual Search)](#-ระบบ-vision-ai-visual-search)
- [ระบบการย้ายข้อมูล (Data Migration จาก v1)](#-ระบบการย้ายข้อมูล-data-migration-จาก-v1)
- [ภาพรวม API Endpoints & Swagger](#-ภาพรวม-api-endpoints--swagger)
- [การทดสอบ (Testing)](#-การทดสอบ-testing)

---

## ✨ จุดเด่นและฟีเจอร์หลัก (Key Features)

### 1. ⚡ ประสิทธิภาพสูงด้วย Fiber v3 & GORM

- เขียนด้วยภาษา Go ร่วมกับ Fiber v3 ที่ใช้ Fasthttp ประหยัด Memory และรองรับ Concurrency สูง
- รองรับทั้ง **SQLite** (สำหรับ Single-node / Edge deployment) และ **PostgreSQL** (สำหรับ Cloud / Central deployment)

### 2. 🔄 Offline-First Synchronization

- รองรับการขายและบันทึกข้อมูลหน้าร้านขณะไม่มีอินเทอร์เน็ต
- ซิงค์แบบ Incremental Sync สองทิศทาง (Pull updates จาก Server และ Push local changes ขึ้น Server)
- มีระบบรองรับ Conflict Resolution และ Timestamp Tracking

### 3. 👁️ ระบบค้นหาสินค้าด้วยภาพ (Vision AI)

- ใช้โมเดล **CLIP (`clip-vit-base-patch32`)** แปลงรูปภาพสินค้าเป็น 512-dimension vector embedding
- ค้นหาสินค้าที่ใกล้เคียงจากภาพถ่ายด้วย Cosine Similarity
- รองรับการรัน Inference 2 รูปแบบ:
  1. **In-process Go ONNX Engine**: รันโมเดล ONNX ผ่าน `onnxruntime_go` โดยตรงใน Go Process เดียวกัน
  2. **Python FastAPI Sidecar Microservice**: Fallback อัตโนมัติหากไม่ได้ติดตั้งไลบรารี ONNX ในเครื่อง

### 4. 💰 คิดเงินแม่นยำ 100% ด้วยหน่วย "สตางค์" (Satang Integer)

- ป้องกันปัญหา Floating-point Rounding Errors ใน JavaScript/JSON โดยคำนวณราคาสินค้า, ยอดรวม, ส่วนลด และภาษีมูลค่าเพิ่มในระดับหน่วยสตางค์ (1 บาท = 100 สตางค์)

### 5. 👥 ระบบลูกค้า สมาชิก และวงเงินเชื่อ (Customer Credit)

- จัดการข้อมูลลูกค้าและสมาชิกร้านค้า
- รองรับการขายแบบเชื่อ / ติดค้างชำระ (Credit Line) พร้อมกำหนดวงเงินสูงสุด
- ระบบตัดหนี้ / ชำระค้างจ่าย และบันทึกประวัติการชำระเงินอย่างละเอียด

### 6. 🖼️ Multi-Image & Smart WebP Processing

- สินค้าหนึ่งรายการสามารถมีได้หลายรูปภาพ พร้อมจัดลำดับและตั้งรูปหลัก
- บีบอัดรูปภาพและแปลงเป็นฟอร์แมต **WebP อัตโนมัติให้มีขนาด <= 50KB** ก่อนบันทึก
- Storage Driver แบบ Pluggable: บันทึกเก็บใน Local Disk หรืออัปโหลดขึ้น **ImageKit CDN**

### 7. 🔐 ความปลอดภัยระดับองค์กร (JWT Dual-Token)

- Access Token (อายุสั้น 15 นาที) สำหรับ Header Authorization: Bearer
- Refresh Token (อายุยาว 7 วัน) ส่งผ่าน **HttpOnly / Secure Cookie** พร้อมกลไก Rotation และ Anti-Tampering
- แยกสิทธิ์การเข้าถึงอย่างชัดเจนด้วย Role-based Access Control (Admin / Cashier)

---

## 🏗️ สถาปัตยกรรมระบบ (Project Architecture)

โปรเจกต์จัดโครงสร้างตามแนวคิด Clean Architecture / DDD:

```text
ice-pos-api-v2/
├── cmd/
│   ├── api/                 # Entrypoint หลักของ REST API Server (Fiber v3)
│   └── migrate/             # CLI เครื่องมือย้ายข้อมูลจาก v1 SQLite สู่ v2
├── config/                  # โหลดและจัดการ Configuration จาก Environment
├── internal/
│   ├── delivery/
│   │   └── http/            # HTTP Layer: Handlers, Middlewares, Routers
│   ├── domain/              # Domain Models, Interfaces, Entity และ DTOs
│   ├── repository/          # Data Access Layer (GORM สำหรับ SQLite / PostgreSQL)
│   ├── usecase/             # Business Logic Layer
│   └── pkg/                 # Internal Shared Packages:
│       ├── calc/            # การคำนวณ Decimal และ Vector Similarity
│       ├── imageutil/       # Resize และ Encode รูปภาพ WebP <= 50KB
│       ├── jwt/             # การออกและตรวจสอบ JWT Token
│       ├── logger/          # Structured Logging (slog)
│       ├── password/        # Bcrypt Password Hashing
│       ├── storage/         # Storage Interface (Local / ImageKit)
│       └── vision/          # CLIP ONNX Engine & HTTP Client
├── models/                  # โมเดล ONNX สำหรับ Vision AI (clip_vision_model.onnx)
├── services/
│   └── vision-ai/           # Python Microservice สำหรับรัน CLIP Image Embedding
├── docs/                    # คู่มือสเปก API, เอกสาร Swagger และคู่มือพัฒนา
├── tests/                   # Automated E2E และ Integration Tests
├── uploads/                 # โฟลเดอร์จัดเก็บไฟล์ภาพ (กรณีใช้ Local Storage)
├── Makefile                 # เครื่องมืออำนวยความสะดวกในการ Build, Test และ Run
└── .env.example             # ตัวอย่างการตั้งค่า Environment Variables
```

---

## 🛠️ เทคโนโลยีที่ใช้ (Tech Stack)

- **Language:** Go (1.22+ / 1.26)
- **Web Framework:** [Fiber v3](https://gofiber.io)
- **ORM & Database:** [GORM](https://gorm.io) (SQLite3 / PostgreSQL via `pgx/v5`)
- **API Documentation:** [Swagger / Swag](https://github.com/swaggo/swag)
- **Authentication:** [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt)
- **AI & Computer Vision:**
  - [ONNX Runtime Go](https://github.com/yalue/onnxruntime_go)
  - Hugging Face `clip-vit-base-patch32` (Quantized ONNX)
  - Python FastAPI / Uvicorn (Sidecar Service)
- **Image Processing:** [chai2010/webp](https://github.com/chai2010/webp), `golang.org/x/image`
- **Cloud Storage:** ImageKit API

---

## 📦 ความต้องการเบื้องต้น (Prerequisites)

1. **Go:** ติดตั้ง Go เวอร์ชัน 1.22 ขึ้นไป
2. **Make:** สำหรับรันคำสั่งสะดวกผ่าน Makefile
3. **(ทางเลือก) ONNX Runtime:** หากต้องการรัน Vision AI แบบ In-process
   - macOS (Homebrew): `brew install onnxruntime`
   - Linux: ติดตั้ง `libonnxruntime.so`
4. **(ทางเลือก) Python 3.9+:** สำหรับรัน Vision AI แบบ Sidecar Microservice (`services/vision-ai`)

---

## ⚙️ การตั้งค่า Environment Variables

คัดลอกไฟล์ `.env.example` ไปเป็น `.env` แล้วปรับแต่งค่าตามสภาพแวดล้อม:

```bash
cp .env.example .env
```

| ตัวแปร                            | ค่าเริ่มต้น                              | คำอธิบาย                                          |
| --------------------------------- | ---------------------------------------- | ------------------------------------------------- |
| `APP_NAME`                        | `ICE POS API v2`                         | ชื่อแอปพลิเคชัน                                   |
| `APP_ENV`                         | `development`                            | สภาพแวดล้อม (`development` / `production`)        |
| `APP_PORT`                        | `8080`                                   | พอร์ตสำหรับรัน API Server                         |
| `ALLOW_ORIGIN`                    | `*`                                      | CORS Allowed Origins                              |
| `JWT_SECRET`                      | `super-secret-...`                       | คีย์ลับสำหรับ Sign JWT Token                      |
| `DB_TYPE`                         | `sqlite`                                 | ชนิดฐานข้อมูล (`sqlite` หรือ `postgres`)          |
| `DB_PATH`                         | `ice_pos_v2.db`                          | ที่อยู่ไฟล์ SQLite (กรณี `DB_TYPE=sqlite`)        |
| `DB_HOST`, `DB_PORT`, etc.        | -                                        | การเชื่อมต่อ PostgreSQL (กรณี `DB_TYPE=postgres`) |
| `STORAGE_DRIVER`                  | `imagekit`                               | ไดรเวอร์จัดเก็บรูป (`local` หรือ `imagekit`)      |
| `IMAGEKIT_PUBLIC_KEY`             | -                                        | Public Key จาก ImageKit.io                        |
| `IMAGEKIT_PRIVATE_KEY`            | -                                        | Private Key จาก ImageKit.io                       |
| `IMAGEKIT_URL_ENDPOINT`           | -                                        | Endpoint URL เช่น `https://ik.imagekit.io/xxxx`   |
| `VISION_MODEL_PATH`               | `models/clip_vision_model.onnx`          | ตำแหน่งไฟล์โมเดล CLIP ONNX                        |
| `ONNXRUNTIME_SHARED_LIBRARY_PATH` | `/opt/homebrew/lib/libonnxruntime.dylib` | Path ของ libonnxruntime ในเครื่อง                 |
| `AI_EMBEDDING_SERVICE_URL`        | `http://localhost:5001`                  | URL ของ Python Sidecar Service                    |

---

## 🚀 ขั้นตอนการติดตั้งและเริ่มใช้งาน (Getting Started)

### 1. ติดตั้งเครื่องมือพัฒนา (Tools)

```bash
make install-tools
```

### 2. ดาวน์โหลดโมเดล Vision AI (ถ้าต้องการใช้งานฟังก์ชันค้นหาด้วยภาพ)

```bash
make download-vision-model
```

### 3. รันเซิร์ฟเวอร์ในโหมด Development (Live-reload)

```bash
make dev
```

_(หากติดตั้ง `air` ไว้ ระบบจะ reload โค้ดให้อัตโนมัติเมื่อมีการบันทึกไฟล์)_

หรือรันตรงผ่าน Go:

```bash
make run
```

เซิร์ฟเวอร์จะเปิดทำงานที่ `http://localhost:8080`

---

## 📜 การใช้งาน Make Commands

| คำสั่ง                       | คำอธิบาย                                                            |
| ---------------------------- | ------------------------------------------------------------------- |
| `make help`                  | แสดงรายการคำสั่งทั้งหมดที่มีใน Makefile                             |
| `make dev`                   | รันเซิร์ฟเวอร์ในโหมด Development พร้อม Hot reload (Air)             |
| `make run`                   | รันโปรแกรมโดยตรงผ่าน `go run cmd/api/main.go`                       |
| `make build`                 | คอมไพล์ Binary พร้อมใช้งานที่ `bin/api`                             |
| `make swagger`               | สร้างเอกสาร Swagger OpenAPI 2.0 ใหม่                                |
| `make test`                  | รันการทดสอบ Unit Tests และ E2E ทั้งหมด                              |
| `make test-coverage`         | รันการทดสอบและสร้างรายงาน Code Coverage เป็น HTML                   |
| `make download-vision-model` | ดาวน์โหลดโมเดล CLIP ONNX ขนาดกะทัดรัดลงในโฟลเดอร์ `models/`         |
| `make run-vision-ai`         | รัน Python Vision AI Microservice บนพอร์ต 5001                      |
| `make migrate-dry-run`       | ทดสอบจำลองการโอนย้ายข้อมูลจาก v1 โดยไม่เขียนลงฐานข้อมูลจริง         |
| `make migrate`               | ดำเนินการย้ายข้อมูลจริงจาก v1 สู่ v2 พร้อมคัดลอกรูปภาพและคำนวณสต็อก |
| `make fmt`                   | จัดรูปแบบโค้ด Go ตามมาตรฐาน (`go fmt`)                              |
| `make lint`                  | ตรวจสอบคุณภาพโค้ดด้วย `golangci-lint`                               |
| `make tidy`                  | ตรวจสอบและจัดระเบียบ Go Modules (`go mod tidy`)                     |

---

## 🧠 ระบบ Vision AI (Visual Search)

ระบบรองรับการค้นหาสินค้าด้วยภาพถ่าย โดยมีกระบวนการทำงาน 2 ช่องทาง:

### 1. In-process Go ONNX Engine (แนะนำสำหรับ Standalone)

ไม่ต้องเปิด Process อื่นเพิ่ม เพียงตั้งค่า path ของ `ONNXRUNTIME_SHARED_LIBRARY_PATH` และดาวน์โหลดโมเดลด้วย `make download-vision-model` ตัว API Server จะรันโมเดล AI ผ่าน CGo เข้าสู่ ONNX Runtime โดยตรง

### 2. Python FastAPI Microservice (Sidecar Fallback)

หากเครื่อง Server ไม่สะดวกติดตั้ง C-libraries สำหรับ ONNX Runtime สามารถเปิด Microservice เสริมได้:

```bash
cd services/vision-ai
pip install -r requirements.txt
python3 -m uvicorn server:app --port 5001 --reload
```

เมื่อ API Server พบว่า In-process Engine ไม่พร้อมใช้งาน จะสลับมาเรียก Microservice ผ่าน HTTP อัตโนมัติ

---

## 🔄 ระบบการย้ายข้อมูล (Data Migration จาก v1)

สำหรับร้านค้าที่ใช้งานระบบ POS รุ่นที่ 1 (SQLite) สามารถโอนย้ายข้อมูลเดิมมายัง v2 ได้อย่างราบรื่น:

```bash
# 1. ทดสอบตรวจเช็คข้อมูลแบบ Dry-run
make migrate-dry-run

# 2. ทำการย้ายข้อมูลจริง
make migrate
```

ระบบจะทำการแปลงราคาเดิมให้อยู่ในหน่วยสตางค์ (Satang), โอนย้ายรายการสินค้า, หมวดหมู่, ลูกค้า, ยอดขาย และคัดลอกไฟล์รูปภาพเดิมไปยังพื้นที่จัดเก็บของ v2 ให้อัตโนมัติ

---

## 🌐 ภาพรวม API Endpoints & Swagger

เข้าชมเอกสาร Interactive API Documentation ผ่าน Swagger UI ได้ที่:
👉 **[http://localhost:8080/swagger](http://localhost:8080/swagger)**

### กลุ่ม Endpoints หลัก:

| Path Prefix          | ฟังก์ชันการทำงาน                                   | สิทธิ์ (Role)   |
| -------------------- | -------------------------------------------------- | --------------- |
| `/api/v1/health`     | ตรวจสอบสถานะความพร้อมของเซิร์ฟเวอร์                | Public          |
| `/api/v1/auth`       | เข้าสู่ระบบ, ต่ออายุ Token (Refresh), ออกจากระบบ   | Public / Bearer |
| `/api/v1/users`      | บริหารจัดการบัญชีผู้ใช้และพนักงานแคชเชียร์         | Admin           |
| `/api/v1/categories` | จัดการหมวดหมู่สินค้า (สร้าง/แก้ไข/ลบ)              | Cashier / Admin |
| `/api/v1/products`   | จัดการสินค้า, ราคาสตางค์, บาร์โค้ด, รูปภาพ         | Cashier / Admin |
| `/api/v1/upload`     | อัปโหลดรูปภาพ (บีบอัดเป็น WebP <= 50KB)            | Cashier / Admin |
| `/api/v1/customers`  | จัดการสมาชิกลูกค้า, วงเงินเชื่อ, ประวัติชำระหนี้   | Cashier / Admin |
| `/api/v1/sales`      | เปิดบิลขาย, คิดเงิน (เงินสด/โอน/เชื่อ), ออกใบเสร็จ | Cashier / Admin |
| `/api/v1/reports`    | รายงานสรุปยอดขายรายวัน/รายเดือน, สินค้าขายดี       | Admin           |
| `/api/v1/sync`       | ดึงและส่งข้อมูลการซิงค์สำหรับการทำงาน Offline      | Cashier / Admin |
| `/api/v1/vision`     | ค้นหาสินค้าจากภาพถ่าย (Visual Vector Search)       | Cashier / Admin |

---

## 🧪 การทดสอบ (Testing)

รันชุดการทดสอบทั้งหมดครอบคลุม Unit Tests, Integration Tests และ E2E Workflows:

```bash
# รันเทสทั้งหมด
make test

# รันเทสพร้อมสร้างรายงานความครอบคลุม (Coverage Report)
make test-coverage
```

ไฟล์รายงานจะถูกบันทึกไว้ที่ `coverage.html` สามารถเปิดดูรายละเอียดผ่านเบราว์เซอร์ได้ทันที

---

## 📄 ใบอนุญาต (License)

โปรเจกต์นี้เผยแพร่ภายใต้สัญญาอนุญาต [MIT License](LICENSE)
