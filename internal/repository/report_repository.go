package repository

import (
	"context"
	"time"

	"ice-pos-api-v2/internal/domain"

	"gorm.io/gorm"
)

type reportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) domain.ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) GetDailySummary(ctx context.Context, startOfDay, endOfDay time.Time) (*domain.DailySalesSummary, error) {
	summary := &domain.DailySalesSummary{
		Date: startOfDay.Format("2006-01-02"),
	}

	// 1. Completed sales metrics
	type SalesAgg struct {
		TotalSales int64
		BillCount  int
	}
	var agg SalesAgg
	err := r.db.WithContext(ctx).Model(&domain.Sale{}).
		Select("COALESCE(SUM(grand_total), 0) as total_sales, COUNT(id) as bill_count").
		Where("occurred_at BETWEEN ? AND ? AND status = ?", startOfDay, endOfDay, domain.SaleStatusCompleted).
		Scan(&agg).Error
	if err != nil {
		return nil, err
	}

	summary.TotalSales = agg.TotalSales
	summary.BillCount = agg.BillCount
	summary.TotalCashCollected = agg.TotalSales

	// 2. Void count
	var voidCount int64
	err = r.db.WithContext(ctx).Model(&domain.Sale{}).
		Where("occurred_at BETWEEN ? AND ? AND status = ?", startOfDay, endOfDay, domain.SaleStatusVoid).
		Count(&voidCount).Error
	if err != nil {
		return nil, err
	}
	summary.VoidCount = int(voidCount)

	// 3. Total COGS calculation
	var totalCost int64
	err = r.db.WithContext(ctx).Table("sale_items").
		Joins("JOIN sales ON sales.id = sale_items.sale_id").
		Where("sales.occurred_at BETWEEN ? AND ? AND sales.status = ?", startOfDay, endOfDay, domain.SaleStatusCompleted).
		Select("COALESCE(SUM(sale_items.cost_price * sale_items.qty), 0)").
		Scan(&totalCost).Error
	if err != nil {
		return nil, err
	}

	summary.TotalCost = totalCost
	summary.GrossProfit = summary.TotalSales - totalCost

	return summary, nil
}

func (r *reportRepository) GetBestSellers(ctx context.Context, startDate, endDate time.Time, limit int) ([]domain.BestSellerItem, error) {
	var items []domain.BestSellerItem

	err := r.db.WithContext(ctx).Table("sale_items").
		Joins("JOIN sales ON sales.id = sale_items.sale_id").
		Where("sales.occurred_at BETWEEN ? AND ? AND sales.status = ?", startDate, endDate, domain.SaleStatusCompleted).
		Select("sale_items.product_id, sale_items.product_name, sale_items.product_code, SUM(sale_items.qty) as total_qty_sold, SUM(sale_items.net_total) as total_revenue").
		Group("sale_items.product_id, sale_items.product_name, sale_items.product_code").
		Order("total_qty_sold DESC").
		Limit(limit).
		Scan(&items).Error

	return items, err
}
