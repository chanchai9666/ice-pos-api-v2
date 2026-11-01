package calc

import (
	"errors"

	"github.com/shopspring/decimal"
)

var (
	ErrInsufficientPayment = errors.New("insufficient payment: tendered amount is less than grand total")
)

// LineGrossTotal calculates unitPrice * qty using decimal precision
func LineGrossTotal(unitPrice int64, qty int) int64 {
	priceDec := decimal.NewFromInt(unitPrice)
	qtyDec := decimal.NewFromInt(int64(qty))
	return priceDec.Mul(qtyDec).IntPart()
}

// LineNetTotal calculates gross - discount
func LineNetTotal(gross int64, discount int64) int64 {
	grossDec := decimal.NewFromInt(gross)
	discountDec := decimal.NewFromInt(discount)
	net := grossDec.Sub(discountDec)
	if net.IsNegative() {
		return 0
	}
	return net.IntPart()
}

// GrandTotal calculates subtotal - billDiscount
func GrandTotal(subtotal int64, billDiscount int64) int64 {
	subtotalDec := decimal.NewFromInt(subtotal)
	discDec := decimal.NewFromInt(billDiscount)
	grand := subtotalDec.Sub(discDec)
	if grand.IsNegative() {
		return 0
	}
	return grand.IntPart()
}

// CalculateChange calculates tendered - grandTotal. Returns ErrInsufficientPayment if tendered < grandTotal.
func CalculateChange(tendered int64, grandTotal int64) (int64, error) {
	tenderedDec := decimal.NewFromInt(tendered)
	grandTotalDec := decimal.NewFromInt(grandTotal)

	if tenderedDec.LessThan(grandTotalDec) {
		return 0, ErrInsufficientPayment
	}

	change := tenderedDec.Sub(grandTotalDec)
	return change.IntPart(), nil
}

// AddSatang adds two satang values using decimal precision
func AddSatang(a, b int64) int64 {
	return decimal.NewFromInt(a).Add(decimal.NewFromInt(b)).IntPart()
}

// SubSatang subtracts b from a using decimal precision
func SubSatang(a, b int64) int64 {
	return decimal.NewFromInt(a).Sub(decimal.NewFromInt(b)).IntPart()
}
