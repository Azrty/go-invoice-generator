package generator

import (
	"fmt"
	_ "fmt"
	"github.com/shopspring/decimal"

	_ "github.com/shopspring/decimal"
)

// Item represent a 'product' or a 'service'
type Item struct {
	Name        string `json:"name,omitempty" validate:"required"`
	Description string `json:"description,omitempty"`
	UnitCost    string `json:"unit_cost,omitempty"`
	Quantity    string `json:"quantity,omitempty"`
	Taxes       []Tax  `json:"taxes,omitempty"`

	Discount *Discount `json:"discount,omitempty"`

	_unitCost decimal.Decimal
	_quantity decimal.Decimal
}

// Prepare convert strings to decimal
func (i *Item) Prepare() error {
	// Unit cost
	unitCost, err := decimal.NewFromString(i.UnitCost)
	if err != nil {
		return err
	}
	i._unitCost = unitCost

	// Quantity
	quantity, err := decimal.NewFromString(i.Quantity)
	if err != nil {
		return err
	}
	i._quantity = quantity

	// Tax
	if i.Taxes != nil {
		for _, v := range i.Taxes {
			if err := v.Prepare(); err != nil {
				return err
			}
		}
	}

	// Discount
	if i.Discount != nil {
		if err := i.Discount.Prepare(); err != nil {
			return err
		}
	}

	return nil
}

// TotalWithoutTaxAndWithoutDiscount returns the total without tax and without discount
func (i *Item) TotalWithoutTaxAndWithoutDiscount() decimal.Decimal {
	quantity, _ := decimal.NewFromString(i.Quantity)
	price, _ := decimal.NewFromString(i.UnitCost)
	total := price.Mul(quantity).Round(2)

	return total
}

// TotalWithoutTaxAndWithDiscount returns the total without tax and with discount
func (i *Item) TotalWithoutTaxAndWithDiscount() decimal.Decimal {
	total := i.TotalWithoutTaxAndWithoutDiscount()

	// Check discount
	if i.Discount != nil {
		dType, dNum := i.Discount.getDiscount()

		if dType == DiscountTypeAmount {
			total = total.Sub(dNum)
		} else {
			// Percent
			toSub := total.Mul(dNum.Div(decimal.NewFromFloat(100)))
			total = total.Sub(toSub)
		}
	}

	return total
}

// TotalWithTaxAndDiscount returns the total with tax and discount
func (i *Item) TotalWithTaxAndDiscount() decimal.Decimal {
	return i.TotalWithoutTaxAndWithDiscount().Add(i.TaxWithTotalDiscounted())
}

// TaxWithTotalDiscounted returns the tax with total discounted
func (i *Item) TaxWithTotalDiscounted() decimal.Decimal {
	result := decimal.NewFromFloat(0)

	if i.Taxes == nil {
		return result
	}

	totalHT := i.TotalWithoutTaxAndWithDiscount()
	for _, v := range i.Taxes {
		taxType, taxAmount, taxAmountForEach := v.getTax()

		if taxType == TaxTypeAmount {
			if taxAmountForEach {
				result = result.Add(taxAmount.Mul(i._quantity))
			} else {
				result = result.Add(taxAmount)
			}
		} else {
			divider := decimal.NewFromFloat(100)
			result = result.Add(totalHT.Mul(taxAmount.Div(divider)))
		}
	}

	return result
}

// appendColTo document doc
func (i *Item) appendColTo(options *Options, doc *Document) {
	// Get base Y (top of line)
	baseY := doc.pdf.GetY()

	// Name
	doc.pdf.SetX(ItemColNameOffset)
	doc.pdf.MultiCell(
		ItemColUnitPriceOffset-ItemColNameOffset,
		3,
		doc.encodeString(i.Name),
		"",
		"",
		false,
	)

	// Description
	if len(i.Description) > 0 {
		doc.pdf.SetX(ItemColNameOffset)
		doc.pdf.SetY(doc.pdf.GetY() + 1)

		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.GreyTextColor[0],
			doc.Options.GreyTextColor[1],
			doc.Options.GreyTextColor[2],
		)

		doc.pdf.MultiCell(
			ItemColUnitPriceOffset-ItemColNameOffset,
			3,
			doc.encodeString(i.Description),
			"",
			"",
			false,
		)

		// Reset font
		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.BaseTextColor[0],
			doc.Options.BaseTextColor[1],
			doc.Options.BaseTextColor[2],
		)
	}

	// Compute line height
	colHeight := doc.pdf.GetY() - baseY

	// Unit price
	doc.pdf.SetY(baseY)
	doc.pdf.SetX(ItemColUnitPriceOffset)
	doc.pdf.CellFormat(
		ItemColQuantityOffset-ItemColUnitPriceOffset,
		colHeight,
		doc.encodeString(doc.Options.CurrencySymbol+i._unitCost.Round(5).String()),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Quantity
	doc.pdf.SetX(ItemColQuantityOffset)
	doc.pdf.CellFormat(
		ItemColTaxOffset-ItemColQuantityOffset,
		colHeight,
		doc.encodeString(i._quantity.String()),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Total HT
	doc.pdf.SetX(ItemColTotalHTOffset)
	doc.pdf.CellFormat(
		ItemColTaxOffset-ItemColTotalHTOffset,
		colHeight,
		doc.encodeString(doc.ac.FormatMoneyDecimal(i.TotalWithoutTaxAndWithoutDiscount())),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Discount
	doc.pdf.SetX(ItemColDiscountOffset)
	if i.Discount == nil {
		doc.pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight,
			doc.encodeString("--"),
			"0",
			0,
			"",
			false,
			0,
			"",
		)
	} else {
		// If discount
		discountType, discountAmount := i.Discount.getDiscount()
		var discountTitle string
		var discountDesc string

		dCost := i.TotalWithoutTaxAndWithoutDiscount()
		if discountType == DiscountTypePercent {
			discountTitle = fmt.Sprintf("%s %s", discountAmount, doc.encodeString("%"))

			// get amount from percent
			dAmount := dCost.Mul(discountAmount.Div(decimal.NewFromFloat(100)))
			discountDesc = fmt.Sprintf("-%s", doc.ac.FormatMoneyDecimal(dAmount))
		} else {
			discountTitle = fmt.Sprintf("%s %s", discountAmount, doc.encodeString("€"))

			// get percent from amount
			dPerc := discountAmount.Mul(decimal.NewFromFloat(100))
			dPerc = dPerc.Div(dCost)
			discountDesc = fmt.Sprintf("-%s %%", dPerc.StringFixed(2))
		}

		// discount title
		// lastY := doc.pdf.GetY()
		doc.pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight/2,
			doc.encodeString(discountTitle),
			"0",
			0,
			"LB",
			false,
			0,
			"",
		)

		// discount desc
		doc.pdf.SetXY(ItemColDiscountOffset, baseY+(colHeight/2))
		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.GreyTextColor[0],
			doc.Options.GreyTextColor[1],
			doc.Options.GreyTextColor[2],
		)

		doc.pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight/2,
			doc.encodeString(discountDesc),
			"0",
			0,
			"LT",
			false,
			0,
			"",
		)

		// reset font and y
		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.BaseTextColor[0],
			doc.Options.BaseTextColor[1],
			doc.Options.BaseTextColor[2],
		)
		doc.pdf.SetY(baseY)
	}

	// Tax
	doc.pdf.SetX(ItemColTaxOffset)
	if i.Taxes == nil {
		// If no tax
		doc.pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight,
			doc.encodeString("--"),
			"0",
			0,
			"",
			false,
			0,
			"",
		)
	} else {
		// If tax
		var taxTitle string
		var taxDesc string
		var taxTotal decimal.Decimal = decimal.NewFromFloat(0)
		var taxTotalPercent decimal.Decimal = decimal.NewFromFloat(0)
		var taxTotalAmount decimal.Decimal = decimal.NewFromFloat(0)
		for _, v := range i.Taxes {
			taxType, taxAmount, taxAmountForEach := v.getTax()

			if taxType == TaxTypePercent {
				taxTotalPercent = taxTotalPercent.Add(taxAmount)
				// get amount from percent
				dCost := i.TotalWithoutTaxAndWithDiscount()
				dAmount := dCost.Mul(taxAmount.Div(decimal.NewFromFloat(100)))
				taxTotal = taxTotal.Add(dAmount)
			} else {
				taxTotalAmount = taxTotalAmount.Add(taxAmount)
				//dCost := i.TotalWithoutTaxAndWithDiscount()
				//dAmount := taxAmount.Mul(decimal.NewFromFloat(100))
				if taxAmountForEach {
					taxTotal = taxTotal.Add(taxAmount.Mul(i._quantity))
				} else {
					taxTotal = taxTotal.Add(taxAmount)
				}
				// get percent from amount
				println("taxTotalAmount", taxTotalAmount.String())
			}
			taxDesc = doc.ac.FormatMoneyDecimal(taxTotal)
		}
		if taxTotalPercent.IsPositive() && taxTotalAmount.IsPositive() {
			taxTitle = fmt.Sprintf("%s %s + %s %s", taxTotalPercent, "%", taxTotalAmount, doc.ac.Symbol)
			taxDesc = doc.ac.FormatMoneyDecimal(taxTotal)
		}
		if taxTotalPercent.IsPositive() && !taxTotalAmount.IsPositive() {
			taxTitle = fmt.Sprintf("%s %s", taxTotalPercent, "%")
			taxDesc = doc.ac.FormatMoneyDecimal(taxTotal)
		}
		if taxTotalPercent.IsZero() && taxTotalAmount.IsPositive() {
			taxTitle = fmt.Sprintf("%s %s", taxTotalAmount, doc.ac.Symbol)
			taxDesc = doc.ac.FormatMoneyDecimal(taxTotal)
		}
		taxDesc = doc.ac.FormatMoneyDecimal(taxTotal)

		// tax title
		// lastY := doc.pdf.GetY()
		doc.pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight/2,
			doc.encodeString(taxTitle),
			"0",
			0,
			"LB",
			false,
			0,
			"",
		)

		// tax desc
		doc.pdf.SetXY(ItemColTaxOffset, baseY+(colHeight/2))
		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.GreyTextColor[0],
			doc.Options.GreyTextColor[1],
			doc.Options.GreyTextColor[2],
		)

		doc.pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight/2,
			doc.encodeString(taxDesc),
			"0",
			0,
			"LT",
			false,
			0,
			"",
		)

		// reset font and y
		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.BaseTextColor[0],
			doc.Options.BaseTextColor[1],
			doc.Options.BaseTextColor[2],
		)
		doc.pdf.SetY(baseY)
	}

	// TOTAL TTC
	doc.pdf.SetX(ItemColTotalTTCOffset)
	doc.pdf.CellFormat(
		190-ItemColTotalTTCOffset,
		colHeight,
		doc.encodeString(doc.ac.FormatMoneyDecimal(i.TotalWithTaxAndDiscount())),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Set Y for next line
	doc.pdf.SetY(baseY + colHeight)
}
