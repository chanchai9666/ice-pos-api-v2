package usecase

import (
	"context"
	"time"

	"ice-pos-api-v2/internal/domain"
)

type reportUsecase struct {
	reportRepo domain.ReportRepository
}

func NewReportUsecase(reportRepo domain.ReportRepository) domain.ReportUsecase {
	return &reportUsecase{reportRepo: reportRepo}
}

func (u *reportUsecase) GetDailySummary(ctx context.Context, dateStr string) (*domain.DailySalesSummary, error) {
	var targetDate time.Time
	if dateStr == "" {
		targetDate = time.Now()
	} else {
		var err error
		targetDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, domain.NewAppError(400, "Invalid date format, expected YYYY-MM-DD")
		}
	}

	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 23, 59, 59, 999999999, targetDate.Location())

	return u.reportRepo.GetDailySummary(ctx, startOfDay, endOfDay)
}

func (u *reportUsecase) GetBestSellers(ctx context.Context, startDateStr, endDateStr string, limit int) ([]domain.BestSellerItem, error) {
	if limit <= 0 {
		limit = 10
	}

	var startDate, endDate time.Time
	if startDateStr == "" {
		startDate = time.Now().AddDate(0, 0, -30) // last 30 days
	} else {
		var err error
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return nil, domain.NewAppError(400, "Invalid start date format, expected YYYY-MM-DD")
		}
	}

	if endDateStr == "" {
		endDate = time.Now()
	} else {
		var err error
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return nil, domain.NewAppError(400, "Invalid end date format, expected YYYY-MM-DD")
		}
	}

	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())

	return u.reportRepo.GetBestSellers(ctx, start, end, limit)
}
