// Package catalog holds the non-sensitive shop data: products, base prices
// and discount rules (data/catalog.json).
package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Product is a catalog item.
type Product struct {
	ProductID string  `json:"product_id"`
	NumericID int     `json:"numeric_id"`
	Name      string  `json:"name"`
	BasePrice float64 `json:"base_price"`
}

// Discount is a discount rule: percent in [0, 1) and a fixed reduction.
type Discount struct {
	Code      string  `json:"code"`
	NumericID int     `json:"numeric_id"`
	Percent   float64 `json:"percent"`
	Fixed     float64 `json:"fixed"`
}

// Catalog is the full catalog document.
type Catalog struct {
	Products  []Product  `json:"products"`
	Discounts []Discount `json:"discounts"`
}

// NoDiscount is the code used when a transaction has no discount.
const NoDiscount = "NONE"

var (
	ErrProductNotFound  = errors.New("product not found")
	ErrDiscountNotFound = errors.New("discount not found")
)

// Load reads and validates a catalog file.
func Load(path string) (*Catalog, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path comes from operator configuration
	if err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	return Parse(data)
}

// Parse decodes and validates a catalog document.
func Parse(data []byte) (*Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	seen := map[string]bool{}
	for _, p := range c.Products {
		if p.ProductID == "" || p.BasePrice <= 0 || seen[p.ProductID] {
			return nil, fmt.Errorf("catalog: invalid or duplicate product %q", p.ProductID)
		}
		seen[p.ProductID] = true
	}
	hasNone := false
	for _, d := range c.Discounts {
		if d.Code == "" || d.Percent < 0 || d.Percent >= 1 || d.Fixed < 0 || seen["discount:"+d.Code] {
			return nil, fmt.Errorf("catalog: invalid or duplicate discount %q", d.Code)
		}
		seen["discount:"+d.Code] = true
		if d.Code == NoDiscount {
			hasNone = true
		}
	}
	if !hasNone {
		return nil, fmt.Errorf("catalog: discount %q must exist", NoDiscount)
	}
	return &c, nil
}

// Product looks up a product by ID.
func (c *Catalog) Product(id string) (Product, error) {
	for _, p := range c.Products {
		if p.ProductID == id {
			return p, nil
		}
	}
	return Product{}, fmt.Errorf("%w: %q", ErrProductNotFound, id)
}

// Discount looks up a discount rule; an empty code means NONE.
func (c *Catalog) Discount(code string) (Discount, error) {
	if code == "" {
		code = NoDiscount
	}
	for _, d := range c.Discounts {
		if d.Code == code {
			return d, nil
		}
	}
	return Discount{}, fmt.Errorf("%w: %q", ErrDiscountNotFound, code)
}
