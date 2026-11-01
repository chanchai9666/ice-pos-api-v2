package calc_test

import (
	"testing"

	"ice-pos-api-v2/internal/pkg/calc"
)

func TestDecimalCalculations(t *testing.T) {
	t.Run("LineGrossTotal with satang precision", func(t *testing.T) {
		// 25.50 THB (2550 satang) * 3 = 76.50 THB (7650 satang)
		unitPrice := int64(2550)
		qty := 3
		expected := int64(7650)
		result := calc.LineGrossTotal(unitPrice, qty)
		if result != expected {
			t.Errorf("expected %d, got %d", expected, result)
		}
	})

	t.Run("LineNetTotal with discount", func(t *testing.T) {
		gross := int64(7650)
		discount := int64(650)
		expected := int64(7000)
		result := calc.LineNetTotal(gross, discount)
		if result != expected {
			t.Errorf("expected %d, got %d", expected, result)
		}
	})

	t.Run("GrandTotal with bill discount", func(t *testing.T) {
		subtotal := int64(10000)
		billDiscount := int64(500)
		expected := int64(9500)
		result := calc.GrandTotal(subtotal, billDiscount)
		if result != expected {
			t.Errorf("expected %d, got %d", expected, result)
		}
	})

	t.Run("CalculateChange exact change calculation", func(t *testing.T) {
		// Grand total = 135.00 THB (13500 satang)
		// Tendered = 500.00 THB (50000 satang)
		// Change = 365.00 THB (36500 satang)
		grandTotal := int64(13500)
		tendered := int64(50000)
		expectedChange := int64(36500)

		change, err := calc.CalculateChange(tendered, grandTotal)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if change != expectedChange {
			t.Errorf("expected change %d, got %d", expectedChange, change)
		}
	})

	t.Run("CalculateChange insufficient cash returns error", func(t *testing.T) {
		grandTotal := int64(13500)
		tendered := int64(10000) // 100 THB vs 135 THB

		_, err := calc.CalculateChange(tendered, grandTotal)
		if err == nil {
			t.Fatalf("expected error for insufficient cash, got nil")
		}
	})
}
