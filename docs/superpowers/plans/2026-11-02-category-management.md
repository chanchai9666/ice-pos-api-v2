# Product Category Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement end-to-end category management (CRUD) in `ice-pos-api-v2` with flat structure, atomic unlinking of products on category deletion, role-based authorization, Swagger docs, and E2E tests.

**Architecture:** Clean Architecture across `internal/domain` (models & contracts), `internal/repository` (GORM transactions & DB operations), `internal/usecase` (validation & business logic), and `internal/delivery/http` (Fiber v3 handlers, auth middleware, and routes).

**Tech Stack:** Go 1.24, Fiber v3, GORM, PostgreSQL / SQLite, Swag CLI.

**Spec:** `docs/superpowers/specs/2026-11-02-category-management-design.md`

## Global Constraints

- Go 1.24+, Fiber v3, GORM.
- Clean Architecture separation must be preserved (`domain` -> `repository` -> `usecase` -> `delivery/http`).
- Deleting a category must NEVER cascade delete products; products are unlinked (`category_id = ''`) atomically in a single DB transaction.
- `GET /categories` and `GET /categories/:id` must be public.
- `POST /categories`, `PUT /categories/:id`, and `DELETE /categories/:id` must require JWT `AuthMiddleware` with `domain.RoleAdmin`.
- Category name must be validated: non-empty, 1-100 characters, unique (case-insensitive, trimmed).

## Review Focus

1. Empty/Whitespace category name on create or update returns `400 Bad Request`.
2. Duplicate category name on create or update (case-insensitive) returns `409 Conflict`.
3. Updating an existing category with its own current name returns `200 OK` (not conflict).
4. Deleting a category that has linked products sets those products' `category_id` to empty string and deletes the category in an atomic transaction.
5. Non-admin user (e.g. cashier or unauthenticated) attempting `POST`, `PUT`, or `DELETE /categories` receives `401 Unauthorized` or `403 Forbidden`.

---

### Task 1: Domain Models and Repository Methods

**Files:**

- Modify: `internal/domain/catalog.go:31-55`
- Modify: `internal/repository/catalog_repository.go:119-139`

**Interfaces:**

- Produces:
  - `CatalogRepository.GetCategoryByName(ctx context.Context, name string) (*domain.Category, error)`
  - `CatalogRepository.UpdateCategory(ctx context.Context, category *domain.Category) error`
  - `CatalogRepository.DeleteCategory(ctx context.Context, id string) error`

- [ ] **Step 1: Update `CatalogRepository` interface in `internal/domain/catalog.go`**

Add `GetCategoryByName`, `UpdateCategory`, and `DeleteCategory` to `CatalogRepository`.

- [ ] **Step 2: Implement methods in `internal/repository/catalog_repository.go`**

Implement:

- `GetCategoryByName`: Query with `LOWER(TRIM(name)) = LOWER(TRIM(?))` returning `domain.ErrNotFound` if not found.
- `UpdateCategory`: Save category with `r.db.WithContext(ctx).Save(category).Error`.
- `DeleteCategory`: Atomically unlink products and delete category within `r.db.WithContext(ctx).Transaction(...)`:
  - `tx.Model(&domain.Product{}).Where("category_id = ?", id).Update("category_id", "").Error`
  - `tx.Delete(&domain.Category{}, "id = ?", id).Error`

- [ ] **Step 3: Verify compilation**

Run: `go vet ./internal/repository/...`
Expected: PASS with no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/catalog.go internal/repository/catalog_repository.go
git commit -m "feat(catalog): add category repository methods for getByName, update, and atomic delete"
```

---

### Task 2: Usecase Logic & Validation

**Files:**

- Modify: `internal/domain/catalog.go`
- Modify: `internal/usecase/catalog_usecase.go:104-120`

**Interfaces:**

- Consumes: `CatalogRepository.GetCategoryByID`, `GetCategoryByName`, `CreateCategory`, `UpdateCategory`, `DeleteCategory`
- Produces:
  - `CreateCategoryInput` & `UpdateCategoryInput`
  - `CatalogUsecase.GetCategoryByID(ctx context.Context, id string) (*domain.Category, error)`
  - `CatalogUsecase.CreateCategory(ctx context.Context, input CreateCategoryInput) (*domain.Category, error)`
  - `CatalogUsecase.UpdateCategory(ctx context.Context, id string, input UpdateCategoryInput) (*domain.Category, error)`
  - `CatalogUsecase.DeleteCategory(ctx context.Context, id string) error`

- [ ] **Step 1: Update `CatalogUsecase` interface and DTOs in `internal/domain/catalog.go`**

Define `CreateCategoryInput` and `UpdateCategoryInput`. Update `CatalogUsecase` method signatures.

- [ ] **Step 2: Implement category usecase methods in `internal/usecase/catalog_usecase.go`**

Implement:

- `GetCategoryByID`: Call `repo.GetCategoryByID`. Return `domain.ErrNotFound` if not found.
- `CreateCategory`:
  - Trim `input.Name`. Check empty or length > 100 -> return `domain.ErrInvalidInput`.
  - Check duplicate name via `repo.GetCategoryByName` -> return `domain.ErrConflict` if exists.
  - Create `domain.Category` with new UUID, `time.Now()`, and call `repo.CreateCategory`.
- `UpdateCategory`:
  - Call `repo.GetCategoryByID(ctx, id)` -> return `domain.ErrNotFound` if missing.
  - Trim `input.Name`. Check empty or length > 100 -> return `domain.ErrInvalidInput`.
  - If name changed, check duplicate name via `repo.GetCategoryByName`. If found with different ID -> return `domain.ErrConflict`.
  - Update `Name`, and if `input.SortOrder != nil`, update `SortOrder`. Update `UpdatedAt = time.Now()`.
  - Call `repo.UpdateCategory(ctx, existing)`.
- `DeleteCategory`:
  - Check `repo.GetCategoryByID(ctx, id)` -> return `domain.ErrNotFound` if missing.
  - Call `repo.DeleteCategory(ctx, id)`.

- [ ] **Step 3: Verify compilation**

Run: `go vet ./internal/usecase/...`
Expected: PASS with no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/catalog.go internal/usecase/catalog_usecase.go
git commit -m "feat(catalog): implement category usecase with validation, unique checks, and deletion"
```

---

### Task 3: HTTP Delivery, Router & Swagger Documentation

**Files:**

- Modify: `internal/delivery/http/handler/catalog_handler.go`
- Modify: `internal/delivery/http/handler/response.go`
- Modify: `internal/delivery/http/router/router.go`
- Generate: `docs/swagger/*`

**Interfaces:**

- Consumes: `CatalogUsecase` category methods
- Produces:
  - `GET /api/v1/categories` (public)
  - `GET /api/v1/categories/:id` (public)
  - `POST /api/v1/categories` (Admin only)
  - `PUT /api/v1/categories/:id` (Admin only)
  - `DELETE /api/v1/categories/:id` (Admin only)

- [ ] **Step 1: Add handler request/response structs and handlers in `catalog_handler.go`**

Add:

- `GetCategoryByID`: Extract `:id`, call usecase, return `200 OK` or `404 Not Found`.
- `CreateCategory`: Parse JSON body, call usecase, return `201 Created` or appropriate error (`400`, `409`).
- `UpdateCategory`: Extract `:id`, parse JSON body, call usecase, return `200 OK` or appropriate error.
- `DeleteCategory`: Extract `:id`, call usecase, return `200 OK` or `404 Not Found`.
- Include full Swag doc annotations (`@Summary`, `@Tags Categories`, `@Security BearerAuth` where needed).

- [ ] **Step 2: Update routes in `internal/delivery/http/router/router.go`**

Register routes under `/categories`:

- `categories.Get("", rc.CatalogHandler.ListCategories)`
- `categories.Get("/:id", rc.CatalogHandler.GetCategoryByID)`
- `categories.Post("", middleware.AuthMiddleware(rc.JWTService), middleware.RequireRoles(domain.RoleAdmin), rc.CatalogHandler.CreateCategory)`
- `categories.Put("/:id", middleware.AuthMiddleware(rc.JWTService), middleware.RequireRoles(domain.RoleAdmin), rc.CatalogHandler.UpdateCategory)`
- `categories.Delete("/:id", middleware.AuthMiddleware(rc.JWTService), middleware.RequireRoles(domain.RoleAdmin), rc.CatalogHandler.DeleteCategory)`

- [ ] **Step 3: Regenerate Swagger documentation**

Run: `make swag`
Expected: Swagger docs re-generated successfully in `docs/swagger/`.

- [ ] **Step 4: Commit**

```bash
git add internal/delivery/http/handler/ internal/delivery/http/router/ docs/swagger/
git commit -m "feat(catalog): expose category CRUD HTTP endpoints with Swagger docs and RBAC"
```

---

### Task 4: E2E Integration Test Suite & Verification

**Files:**

- Create: `tests/e2e_category_test.go`

**Interfaces:**

- Exercises: Category endpoints via Fiber Test app with SQLite test database

- [ ] **Step 1: Write `tests/e2e_category_test.go`**

Create test covering:

1. `GET /categories` initially returns empty or seeded list.
2. `POST /categories` without auth fails with `401 Unauthorized`.
3. `POST /categories` with Cashier token fails with `403 Forbidden`.
4. `POST /categories` with Admin token successfully creates category with `201 Created`.
5. Duplicate name `POST /categories` returns `409 Conflict`.
6. Empty name `POST /categories` returns `400 Bad Request`.
7. `GET /categories/:id` returns created category.
8. `PUT /categories/:id` updates category name and sort order.
9. Link a test product to the category.
10. `DELETE /categories/:id` succeeds and verifies the product's `category_id` is cleared to `""`.
11. `GET /categories/:id` on deleted category returns `404 Not Found`.

- [ ] **Step 2: Run E2E tests**

Run: `go test -v ./tests -run TestCategoryLifecycle`
Expected: PASS

- [ ] **Step 3: Run entire test suite**

Run: `go test ./...`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add tests/e2e_category_test.go
git commit -m "test: add E2E integration test suite for category management"
```
