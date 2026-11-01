package domain

import (
	"context"
	"time"
)

type SyncChangesResponse struct {
	Since      *time.Time `json:"since,omitempty"`
	ServerTime time.Time  `json:"server_time"`
	Categories []Category `json:"categories"`
	Products   []Product  `json:"products"`
}

type SyncBatchRequest struct {
	Sales []Sale `json:"sales"`
}

type SyncBatchResponse struct {
	SyncedSales  int      `json:"synced_sales"`
	SkippedSales int      `json:"skipped_sales"`
	FailedSales  int      `json:"failed_sales"`
	Errors       []string `json:"errors,omitempty"`
}

type SyncRepository interface {
	GetChanges(ctx context.Context, since *time.Time) ([]Category, []Product, error)
	BatchUpsertSales(ctx context.Context, sales []Sale) (*SyncBatchResponse, error)
}

type SyncUsecase interface {
	PullChanges(ctx context.Context, sinceStr string) (*SyncChangesResponse, error)
	PushBatch(ctx context.Context, req SyncBatchRequest) (*SyncBatchResponse, error)
}
