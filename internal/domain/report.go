package domain

import (
	"context"
	"time"
)

type DailySalesSummary struct {
	Date               string `json:"date"`
	TotalSales         int64  `json:"total_sales"`          // Satang
	TotalCost          int64  `json:"total_cost"`           // Satang
	GrossProfit        int64  `json:"gross_profit"`         // Satang
	TotalCashCollected int64  `json:"total_cash_collected"` // Satang (Net sales collected)
	BillCount          int    `json:"bill_count"`
	VoidCount          int    `json:"void_count"`
}

type BestSellerItem struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	ProductCode  string `json:"product_code"`
	TotalQtySold int    `json:"total_qty_sold"`
	TotalRevenue int64  `json:"total_revenue"` // Satang
}

type ReportRepository interface {
	GetDailySummary(ctx context.Context, startOfDay, endOfDay time.Time) (*DailySalesSummary, error)
	GetBestSellers(ctx context.Context, startDate, endDate time.Time, limit int) ([]BestSellerItem, error)
}

type ReportUsecase interface {
	GetDailySummary(ctx context.Context, dateStr string) (*DailySalesSummary, error)
	GetBestSellers(ctx context.Context, startDateStr, endDateStr string, limit int) ([]BestSellerItem, error)
}
