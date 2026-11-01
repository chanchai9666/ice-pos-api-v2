# ICE POS API v2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build `ice-pos-api-v2` as a simplified, high-performance Go Fiber v3 Clean Architecture POS API tailored for single-branch stores and elderly cashiers, focusing on product lookup, cash sale with change calculation, easy bill voiding, and dynamic sales reporting.

**Architecture:** Clean Architecture (Delivery -> Usecase -> Domain <- Repository) powered by Go Fiber v3, GORM with SQLite/PostgreSQL, and `github.com/shopspring/decimal` for financial accuracy.

**Tech Stack:** Go 1.23+, Go Fiber v3, GORM, SQLite (`gorm.io/driver/sqlite`), `github.com/shopspring/decimal`, JWT (`golang-jwt/jwt/v5`), `golang.org/x/crypto/bcrypt`.

**Spec:** `docs/superpowers/specs/2026-11-01-ice-pos-api-v2-design.md`

---

## User Review Required

> [!IMPORTANT]
>
> - **Cash-Only Checkout:** All complex payment options (PromptPay QR, Credit/Debit, Bank Transfer, AR credit) are removed. The system expects `tendered_amount` and calculates `change_amount`.
> - **No Stock Restrictions:** Product inventory, stock deductions, adjustments, transfers, and PO/GRN are completely eliminated. Sales never block on stock levels.
> - **Direct Bill Voiding:** Cashiers can directly void mistaken sales (`POST /api/v1/sales/:id/void`) without requiring Admin overrides. Voided sales are automatically excluded from revenue reports.
> - **Zero Shift Ceremony:** No opening/closing shifts. The POS is always ready for sales upon login.

---

## Open Questions

None remaining. All requirements have been confirmed by user review.

---

## Proposed Changes

### Component 1: Scaffolding & Core Packages (`pkg`)

Set up the project foundation, dependencies, configuration, and shared mathematical & security utilities.

#### [NEW] `go.mod`

- Module: `ice-pos-api-v2`
- Dependencies: `gofiber/fiber/v3`, `gorm.io/gorm`, `gorm.io/driver/sqlite`, `shopspring/decimal`, `golang-jwt/jwt/v5`, `go-playground/validator/v10`, `joho/godotenv`.

#### [NEW] `config/config.go`

- Load app port, environment, JWT secret, database type (default sqlite `ice_pos_v2.db`).

#### [NEW] `internal/pkg/calc/decimal.go`

- Financial calculations using `shopspring/decimal`: Gross total, line net total, grand total, VAT calculations, and change calculation (`Tendered - GrandTotal`).

#### [NEW] `internal/pkg/jwt/jwt.go` & `internal/pkg/password/password.go`

- JWT token generation/validation, bcrypt hashing for passwords and Cashier PINs.

#### [NEW] `internal/pkg/validator/validator.go` & `internal/pkg/logger/logger.go`

- Request validation wrapper and structured logging (`log/slog`).

---

### Component 2: Domain Layer

Define core business entities, repository interfaces, and usecase interfaces.

#### [NEW] `internal/domain/errors.go`

- Standard `AppError` definitions and error codes.

#### [NEW] `internal/domain/user.go`

- `User` entity (`ID`, `Username`, `PasswordHash`, `PinHash`, `DisplayName`, `Role`: `ADMIN` | `CASHIER`, `IsActive`).
- `UserRepository` and `UserUsecase` interfaces.

#### [NEW] `internal/domain/catalog.go`

- `Category` and `Product` entities (`ID`, `Code`, `Barcode`, `Name`, `Price`, `CostPrice`, `CategoryID`, `ImageURL`, `IsActive`).
- `CatalogRepository` and `CatalogUsecase` interfaces.

#### [NEW] `internal/domain/sale.go`

- `Sale` entity (`ID`, `SaleNo`, `SubTotal`, `BillDiscount`, `GrandTotal`, `TenderedAmount`, `ChangeAmount`, `Status`: `COMPLETED` | `VOID`, `VoidReason`, `CashierID`, `OccurredAt`).
- `SaleItem` entity (`ID`, `SaleID`, `ProductID`, `ProductName`, `UnitPrice`, `CostPrice`, `Qty`, `LineDiscount`, `NetTotal`).
- `Receipt` entity for receipt formatting.
- `SaleRepository` and `SaleUsecase` interfaces.

#### [NEW] `internal/domain/report.go`

- `DailySalesSummary` (`Date`, `TotalSales`, `TotalCost`, `GrossProfit`, `BillCount`, `VoidCount`, `TotalCashCollected`).
- `BestSellerItem` (`ProductID`, `ProductName`, `TotalQty`, `TotalRevenue`).
- `ReportRepository` and `ReportUsecase` interfaces.

---

### Component 3: Repository & Database Layer

Implement data access using GORM with SQLite (and Postgres support) without legacy stock/shift tables.

#### [NEW] `internal/repository/db.go`

- Database connection initialization, auto-migration of simplified schema (`users`, `categories`, `products`, `sales`, `sale_items`, `receipts`, `system_configs`).
- Default seed: Admin user (`admin`), default cashier (`cashier`), and store config.

#### [NEW] `internal/repository/user_repository.go`

- User lookup by username, PIN, ID, and user CRUD.

#### [NEW] `internal/repository/catalog_repository.go`

- Product lookup by ID, code, barcode; category CRUD; product CRUD.

#### [NEW] `internal/repository/sale_repository.go`

- Transactional sale creation (generates `SaleNo`, calculates change, creates receipt record).
- Sale lookup and sale voiding (`status = VOID`).

#### [NEW] `internal/repository/report_repository.go`

- Dynamic queries aggregating completed sales between dates: total revenue, total cost, gross profit, bill counts, and top-selling items.

---

### Component 4: Usecase Layer

Implement business workflows with `shopspring/decimal`.

#### [NEW] `internal/usecase/auth_usecase.go`

- Login with Username/Password and PIN login for cashiers; token issuance and refresh.

#### [NEW] `internal/usecase/user_usecase.go`

- Admin user management: create cashier, change/reset PIN.

#### [NEW] `internal/usecase/catalog_usecase.go`

- Product catalog search, barcode lookup, category listing.

#### [NEW] `internal/usecase/sale_usecase.go`

- Server-authoritative price lookup from catalog.
- Calculate subtotal, discount, grand total using `shopspring/decimal`.
- Validate `tendered_amount >= grand_total`, calculate `change_amount`.
- Save sale with `status = COMPLETED`.
- Void sale: Mark `VOID` with reason and timestamp.

#### [NEW] `internal/usecase/report_usecase.go`

- Generate dynamic sales summary and best seller reports for any given date range.

---

### Component 5: Delivery Layer (Fiber v3 HTTP Handlers & Router)

Clean HTTP surface with only relevant endpoints.

#### [NEW] `internal/delivery/http/middleware/auth_middleware.go`

- JWT verification and role checks (`RequireRoles(RoleAdmin)`).

#### [NEW] `internal/delivery/http/handler/auth_handler.go`

- Endpoints: `POST /api/v1/auth/login`, `POST /api/v1/auth/pin-login`, `GET /api/v1/auth/me`.

#### [NEW] `internal/delivery/http/handler/user_handler.go`

- Endpoints: `GET /api/v1/users`, `POST /api/v1/users`, `POST /api/v1/users/:id/reset-pin`.

#### [NEW] `internal/delivery/http/handler/catalog_handler.go`

- Endpoints: `GET /api/v1/products`, `GET /api/v1/products/barcode/:barcode`, `GET /api/v1/products/:id`, `GET /api/v1/categories`.

#### [NEW] `internal/delivery/http/handler/sale_handler.go`

- Endpoints: `POST /api/v1/sales` (Cash checkout), `GET /api/v1/sales`, `GET /api/v1/sales/:id`, `POST /api/v1/sales/:id/void`, `GET /api/v1/sales/receipt/:receiptNo`.

#### [NEW] `internal/delivery/http/handler/report_handler.go`

- Endpoints: `GET /api/v1/reports/sales-summary?date=YYYY-MM-DD`, `GET /api/v1/reports/best-sellers?start=...&end=...`.

#### [NEW] `internal/delivery/http/router/router.go`

- Fiber route registration and CORS setup.

#### [NEW] `cmd/api/main.go`

- Application entry point with graceful shutdown and startup banner.

---

## Verification Plan

### Automated Tests

1. **Decimal Calculation Unit Tests (`internal/pkg/calc/decimal_test.go`):**
   - Test subtotal, grand total, VAT, tendered cash, and exact change calculation in satang precision.
2. **Sale Usecase Unit Tests (`internal/usecase/sale_usecase_test.go`):**
   - Test cash checkout with valid tendered cash -> success, returns exact change.
   - Test insufficient tendered cash (`tendered < grandTotal`) -> returns 400 error.
   - Test void sale -> status becomes `VOID`, cannot void an already voided sale.
3. **Integration / E2E Tests (`tests/e2e_sale_test.go`):**
   - Test complete flow: Login -> Lookup Product -> Create Sale -> Check Change -> Get Report -> Void Sale -> Check Report updated.

### Manual Verification

1. Run `go run cmd/api/main.go` and verify server starts cleanly on `:8080`.
2. Execute PIN login via `curl -X POST http://localhost:8080/api/v1/auth/pin-login -d '{"pin":"1234"}'`.
3. Submit a cash sale with `tendered_amount` and verify `change_amount` in response.
4. Call `GET /api/v1/reports/sales-summary` and verify summary values.
