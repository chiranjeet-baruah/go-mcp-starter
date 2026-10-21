// Package core contains the arithmetic use case, independent of its callers.
package core

import (
	"context"
	"errors"
	"math"
)

// Adder is the application's inbound port.
type Adder interface {
	Add(context.Context, int64, int64) (int64, error)
}

// ErrOverflow means the sum cannot be represented as a signed 64-bit integer.
var ErrOverflow = errors.New("sum exceeds the signed 64-bit integer range")

// Calculator implements Adder without external dependencies or mutable state.
type Calculator struct{}

func (Calculator) Add(ctx context.Context, a, b int64) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return 0, ErrOverflow
	}
	return a + b, nil
}
